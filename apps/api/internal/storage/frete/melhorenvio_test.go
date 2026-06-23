package frete

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
)

func itensExemplo() []casosdeuso.ItemFrete {
	return []casosdeuso.ItemFrete{
		{PesoGramas: 600, AlturaCm: 5, LarguraCm: 30, ComprimentoCm: 40, ValorCentavos: 12000},
	}
}

func TestMelhorEnvioEscolheServicoMaisBarato(t *testing.T) {
	var corpoRecebido string
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-teste" {
			t.Errorf("Authorization ausente ou incorreto: %q", r.Header.Get("Authorization"))
		}
		corpo, _ := io.ReadAll(r.Body)
		corpoRecebido = string(corpo)
		_, _ = w.Write([]byte(`[
			{"name":"SEDEX","price":"38.90","delivery_time":3,"company":{"name":"Correios"}},
			{"name":"PAC","price":"25.50","delivery_time":7,"company":{"name":"Correios"}},
			{"name":"Indisponível","error":"serviço indisponível para a rota"}
		]`))
	}))
	defer servidor.Close()

	cotador := NovoMelhorEnvio(servidor.URL, "token-teste", "ReVeste (teste)")
	cotacao, err := cotador.Cotar(context.Background(), "01310100", "20040002", itensExemplo())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cotacao.ValorCentavos != 2550 {
		t.Fatalf("valor = %d; esperado 2550 (PAC, o mais barato)", cotacao.ValorCentavos)
	}
	if cotacao.Servico != "Correios PAC" {
		t.Fatalf("servico = %q; esperado \"Correios PAC\"", cotacao.Servico)
	}
	if cotacao.PrazoDias != 7 || cotacao.Provedor != "melhor_envio" {
		t.Fatalf("cotacao inesperada: %+v", cotacao)
	}
	// confirma que peso (kg) e CEPs foram enviados ao provedor.
	if !strings.Contains(corpoRecebido, `"weight":0.6`) || !strings.Contains(corpoRecebido, `"postal_code":"01310100"`) {
		t.Fatalf("corpo enviado inesperado: %s", corpoRecebido)
	}
}

func TestMelhorEnvioSemServicoValido(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"PAC","error":"indisponível"}]`))
	}))
	defer servidor.Close()

	_, err := NovoMelhorEnvio(servidor.URL, "token", "ua").Cotar(context.Background(), "01310100", "20040002", itensExemplo())
	if !errors.Is(err, common.ErrCotacaoFreteIndisponivel) {
		t.Fatalf("esperava ErrCotacaoFreteIndisponivel, veio %v", err)
	}
}

func TestMelhorEnvioStatusInesperado(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer servidor.Close()

	_, err := NovoMelhorEnvio(servidor.URL, "token", "ua").Cotar(context.Background(), "01310100", "20040002", itensExemplo())
	if !errors.Is(err, common.ErrCotacaoFreteIndisponivel) {
		t.Fatalf("esperava ErrCotacaoFreteIndisponivel, veio %v", err)
	}
}

func TestFixoDevolveValorFixo(t *testing.T) {
	cotacao, err := NovoFixo(1990).Cotar(context.Background(), "x", "y", nil)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cotacao.ValorCentavos != 1990 || cotacao.Provedor != "fixo" {
		t.Fatalf("cotacao inesperada: %+v", cotacao)
	}
}
