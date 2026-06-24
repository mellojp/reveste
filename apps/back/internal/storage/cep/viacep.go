// Package cep contem adaptadores para provedores externos de consulta de CEP.
package cep

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/cadastros"
)

const urlBaseViaCEP = "https://viacep.com.br/ws/"

// ViaCEP implementa casosdeuso.ConsultorCEP usando o servico publico viacep.com.br,
// que nao exige autenticacao. A porta ConsultorCEP permite substitui-lo (BrasilAPI,
// Correios) sem alterar o caso de uso.
type ViaCEP struct {
	cliente *http.Client
	urlBase string
}

// NovoViaCEP cria um consultor com timeout curto: a consulta acontece durante o
// preenchimento de um formulario e nao pode prender a requisicao do usuario.
func NovoViaCEP() *ViaCEP {
	return &ViaCEP{
		cliente: &http.Client{Timeout: 5 * time.Second},
		urlBase: urlBaseViaCEP,
	}
}

// flagErro aceita tanto o booleano `true` quanto a string `"true"`: o ViaCEP ja devolveu
// o campo "erro" nos dois formatos ao longo das versoes da API.
type flagErro bool

func (f *flagErro) UnmarshalJSON(dados []byte) error {
	*f = flagErro(strings.Trim(string(dados), `"`) == "true")
	return nil
}

type respostaViaCEP struct {
	Logradouro  string   `json:"logradouro"`
	Complemento string   `json:"complemento"`
	Bairro      string   `json:"bairro"`
	Localidade  string   `json:"localidade"`
	UF          string   `json:"uf"`
	Erro        flagErro `json:"erro"`
}

// ConsultarCEP busca o endereco do CEP. Devolve ErrNaoEncontrado quando o CEP nao existe
// e ErrConsultaCEPIndisponivel em falhas transitorias (rede, timeout, status inesperado).
func (v *ViaCEP) ConsultarCEP(ctx context.Context, cep string) (cadastros.Endereco, error) {
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodGet, v.urlBase+cep+"/json/", nil)
	if err != nil {
		return cadastros.Endereco{}, fmt.Errorf("%w: %v", common.ErrConsultaCEPIndisponivel, err)
	}

	resposta, err := v.cliente.Do(requisicao)
	if err != nil {
		return cadastros.Endereco{}, fmt.Errorf("%w: %v", common.ErrConsultaCEPIndisponivel, err)
	}
	defer resposta.Body.Close()

	// O ViaCEP responde 400 para CEP com formato invalido. Como o caso de uso ja valida
	// o formato antes de chegar aqui, tratamos isso como "nao encontrado".
	if resposta.StatusCode == http.StatusBadRequest {
		return cadastros.Endereco{}, common.ErrNaoEncontrado
	}
	if resposta.StatusCode != http.StatusOK {
		return cadastros.Endereco{}, common.ErrConsultaCEPIndisponivel
	}

	var dados respostaViaCEP
	if err := json.NewDecoder(io.LimitReader(resposta.Body, 1<<16)).Decode(&dados); err != nil {
		return cadastros.Endereco{}, fmt.Errorf("%w: %v", common.ErrConsultaCEPIndisponivel, err)
	}
	// CEP inexistente: o ViaCEP devolve 200 com {"erro": true}. Sem localidade nao ha o que
	// preencher, entao tratamos como nao encontrado.
	if bool(dados.Erro) || dados.Localidade == "" {
		return cadastros.Endereco{}, common.ErrNaoEncontrado
	}

	endereco := cadastros.Endereco{
		CEP:         cep,
		Logradouro:  dados.Logradouro,
		Complemento: dados.Complemento,
		Bairro:      dados.Bairro,
		Cidade:      dados.Localidade,
		Estado:      dados.UF,
	}
	endereco.Normalizar()
	return endereco, nil
}
