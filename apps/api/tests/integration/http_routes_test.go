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

func TestRotaProntidaoVerificaDependencias(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/saude/prontidao", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	if !strings.Contains(resposta.Body.String(), `"status":"pronto"`) {
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

func TestRotaDoFrontendRecebeIndexParaNavegacaoDireta(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/perfil", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	if !strings.Contains(resposta.Body.String(), `id="page"`) {
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

func TestDetalheDoAnuncioPermanecePublico(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/anuncios/anuncio-publico", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	if !strings.Contains(resposta.Body.String(), `"titulo":"Casaco de lã"`) {
		t.Fatalf("resposta inesperada: %s", resposta.Body.String())
	}
	if !strings.Contains(resposta.Body.String(), `"nome":"Vendedora Teste"`) {
		t.Fatalf("vendedor ausente da resposta: %s", resposta.Body.String())
	}
}

func TestPerfilPublicoDoVendedorNaoExpoeContato(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/vendedores/vendedor-1", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	corpo := resposta.Body.String()
	if !strings.Contains(corpo, `"nome":"Vendedora Teste"`) {
		t.Fatalf("perfil inesperado: %s", corpo)
	}
	if strings.Contains(corpo, "privado@teste.local") || strings.Contains(corpo, "79999999999") {
		t.Fatalf("perfil publico vazou dados privados: %s", corpo)
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

func TestRespostasIncluemHeadersDeSeguranca(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("Content-Security-Policy nao foi definido")
	}
	if resposta.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", resposta.Header().Get("X-Content-Type-Options"))
	}
}
