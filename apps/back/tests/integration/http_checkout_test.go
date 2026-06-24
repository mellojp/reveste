package integration_test

import (
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckoutExigeAutenticacao(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/checkout", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusUnauthorized {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusUnauthorized)
	}
}

func TestCheckoutComCarrinhoVazioRetorna422(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/checkout", nil)
	requisicao.Header.Set("Authorization", "Bearer sessao-valida")
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusUnprocessableEntity {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusUnprocessableEntity)
	}
	if !strings.Contains(resposta.Body.String(), "CARRINHO_VAZIO") {
		t.Fatalf("resposta inesperada: %s", resposta.Body.String())
	}
}

func TestPaginaMeusPedidosRenderiza(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/meus-pedidos", nil)
	requisicao.AddCookie(&nethttp.Cookie{Name: "reveste_session", Value: "sessao-valida"})
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	if !strings.Contains(resposta.Body.String(), "Meus pedidos") {
		t.Fatalf("pagina de pedidos inesperada: %s", resposta.Body.String())
	}
}
