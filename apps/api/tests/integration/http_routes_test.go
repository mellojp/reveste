package integration_test

import (
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRotaSaudePermanecePublica(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/saude", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	if !strings.Contains(resposta.Body.String(), `"status":"ok"`) {
		t.Fatalf("resposta inesperada: %s", resposta.Body.String())
	}
}

func TestFrontendEEntreguePelaAPI(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	if !strings.Contains(resposta.Body.String(), "ReVeste") {
		t.Fatalf("frontend inesperado: %s", resposta.Body.String())
	}
}

func TestCatalogoPermanecePublico(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/anuncios", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	if !strings.Contains(resposta.Body.String(), `"quantidade":0`) {
		t.Fatalf("resposta inesperada: %s", resposta.Body.String())
	}
}

func TestCarrinhoContinuaExigindoAutenticacao(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/carrinho", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusUnauthorized {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusUnauthorized)
	}
}
