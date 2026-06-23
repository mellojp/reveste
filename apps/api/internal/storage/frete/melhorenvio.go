package frete

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
)

const (
	provedorMelhorEnvio = "melhor_envio"
	caminhoCalcular     = "/api/v2/me/shipment/calculate"
)

// MelhorEnvio implementa casosdeuso.CotadorFrete usando a API do agregador Melhor Envio
// (https://docs.melhorenvio.com.br). Cota os servicos disponiveis entre dois CEPs para os
// produtos informados e escolhe a opcao valida mais barata.
type MelhorEnvio struct {
	cliente   *http.Client
	urlBase   string
	token     string
	userAgent string
}

// NovoMelhorEnvio cria o cotador. A urlBase deve apontar para o ambiente desejado
// (sandbox ou producao). O token e um Bearer de aplicacao e o userAgent identifica a
// integracao, como exige a documentacao do provedor (nome e contato).
func NovoMelhorEnvio(urlBase, token, userAgent string) *MelhorEnvio {
	return &MelhorEnvio{
		cliente:   &http.Client{Timeout: 8 * time.Second},
		urlBase:   strings.TrimRight(urlBase, "/"),
		token:     token,
		userAgent: userAgent,
	}
}

type pontoCEP struct {
	PostalCode string `json:"postal_code"`
}

type produtoCalculo struct {
	ID             string  `json:"id"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Length         int     `json:"length"`
	Weight         float64 `json:"weight"`
	InsuranceValue float64 `json:"insurance_value"`
	Quantity       int     `json:"quantity"`
}

type requisicaoCalculo struct {
	From     pontoCEP         `json:"from"`
	To       pontoCEP         `json:"to"`
	Products []produtoCalculo `json:"products"`
}

// servicoCalculo e cada opcao devolvida. Servicos indisponiveis trazem o campo "error"
// preenchido e nao trazem preco; sao descartados.
type servicoCalculo struct {
	Name         string `json:"name"`
	Price        string `json:"price"`
	DeliveryTime int    `json:"delivery_time"`
	Company      struct {
		Name string `json:"name"`
	} `json:"company"`
	Error string `json:"error"`
}

// Cotar monta a requisicao a partir dos itens (peso em kg, valor declarado em reais) e
// devolve a opcao valida mais barata. Qualquer falha vira ErrCotacaoFreteIndisponivel.
func (m *MelhorEnvio) Cotar(
	ctx context.Context,
	origemCEP, destinoCEP string,
	itens []casosdeuso.ItemFrete,
) (casosdeuso.CotacaoFrete, error) {
	produtos := make([]produtoCalculo, 0, len(itens))
	for indice, item := range itens {
		produtos = append(produtos, produtoCalculo{
			ID:             strconv.Itoa(indice + 1),
			Width:          item.LarguraCm,
			Height:         item.AlturaCm,
			Length:         item.ComprimentoCm,
			Weight:         float64(item.PesoGramas) / 1000,
			InsuranceValue: float64(item.ValorCentavos) / 100,
			Quantity:       1,
		})
	}

	corpo, err := json.Marshal(requisicaoCalculo{
		From:     pontoCEP{PostalCode: origemCEP},
		To:       pontoCEP{PostalCode: destinoCEP},
		Products: produtos,
	})
	if err != nil {
		return casosdeuso.CotacaoFrete{}, fmt.Errorf("%w: %v", common.ErrCotacaoFreteIndisponivel, err)
	}

	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, m.urlBase+caminhoCalcular, bytes.NewReader(corpo))
	if err != nil {
		return casosdeuso.CotacaoFrete{}, fmt.Errorf("%w: %v", common.ErrCotacaoFreteIndisponivel, err)
	}
	requisicao.Header.Set("Content-Type", "application/json")
	requisicao.Header.Set("Accept", "application/json")
	requisicao.Header.Set("Authorization", "Bearer "+m.token)
	if m.userAgent != "" {
		requisicao.Header.Set("User-Agent", m.userAgent)
	}

	resposta, err := m.cliente.Do(requisicao)
	if err != nil {
		return casosdeuso.CotacaoFrete{}, fmt.Errorf("%w: %v", common.ErrCotacaoFreteIndisponivel, err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		return casosdeuso.CotacaoFrete{}, fmt.Errorf("%w: status %d", common.ErrCotacaoFreteIndisponivel, resposta.StatusCode)
	}

	var servicos []servicoCalculo
	if err := json.NewDecoder(io.LimitReader(resposta.Body, 1<<20)).Decode(&servicos); err != nil {
		return casosdeuso.CotacaoFrete{}, fmt.Errorf("%w: %v", common.ErrCotacaoFreteIndisponivel, err)
	}

	return escolherMaisBarato(servicos)
}

// escolherMaisBarato seleciona a opcao valida (sem erro e com preco positivo) de menor valor.
func escolherMaisBarato(servicos []servicoCalculo) (casosdeuso.CotacaoFrete, error) {
	var (
		melhor    casosdeuso.CotacaoFrete
		encontrou bool
	)
	for _, servico := range servicos {
		if servico.Error != "" {
			continue
		}
		centavos, ok := reaisParaCentavos(servico.Price)
		if !ok || centavos <= 0 {
			continue
		}
		if !encontrou || centavos < melhor.ValorCentavos {
			nome := servico.Name
			if servico.Company.Name != "" {
				nome = servico.Company.Name + " " + servico.Name
			}
			melhor = casosdeuso.CotacaoFrete{
				ValorCentavos: centavos,
				Provedor:      provedorMelhorEnvio,
				Servico:       strings.TrimSpace(nome),
				PrazoDias:     servico.DeliveryTime,
			}
			encontrou = true
		}
	}
	if !encontrou {
		return casosdeuso.CotacaoFrete{}, fmt.Errorf("%w: nenhum servico disponivel", common.ErrCotacaoFreteIndisponivel)
	}
	return melhor, nil
}

// reaisParaCentavos converte o preco em reais (string com ponto decimal, ex.: "12.50")
// para centavos, arredondando.
func reaisParaCentavos(preco string) (int64, bool) {
	valor, err := strconv.ParseFloat(strings.TrimSpace(preco), 64)
	if err != nil || valor < 0 {
		return 0, false
	}
	return int64(valor*100 + 0.5), true
}
