package integration_test

import (
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCatalogoRejeitaFiltroInvalido(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/anuncios?deslocamento=-1", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusBadRequest {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusBadRequest)
	}
}

func TestCatalogoRejeitaCategoriaLivre(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/anuncios?categoria=inventada", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusBadRequest {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusBadRequest)
	}
}

func TestCadastroRejeitaSegundoValorJSON(t *testing.T) {
	corpo := `{"nome":"Teste"} {"nome":"Outro"}`
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/usuarios", strings.NewReader(corpo))
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusBadRequest {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusBadRequest)
	}
}

func TestCadastroInformaCamposInvalidos(t *testing.T) {
	corpo := `{
		"nome":"A",
		"cpf":"123",
		"email":"invalido",
		"senha":"12345678",
		"endereco":{
			"cep":"1",
			"logradouro":"",
			"numero":"",
			"bairro":"",
			"cidade":"",
			"estado":"S"
		}
	}`
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/usuarios", strings.NewReader(corpo))
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusUnprocessableEntity {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusUnprocessableEntity)
	}
	for _, campo := range []string{`"nome"`, `"cpf"`, `"email"`, `"endereco.cep"`} {
		if !strings.Contains(resposta.Body.String(), campo) {
			t.Fatalf("campo %s ausente da resposta: %s", campo, resposta.Body.String())
		}
	}
}

func TestLoginLimitaTentativasRepetidas(t *testing.T) {
	handler := novoHandler()
	corpo := `{"identificador":"inexistente@teste.local","senha":"senha-invalida"}`

	for tentativa := 1; tentativa <= 5; tentativa++ {
		requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/sessoes", strings.NewReader(corpo))
		requisicao.RemoteAddr = "192.0.2.10:1234"
		resposta := httptest.NewRecorder()
		handler.ServeHTTP(resposta, requisicao)
		if resposta.Code != nethttp.StatusUnauthorized {
			t.Fatalf("tentativa %d: status = %d; esperado %d", tentativa, resposta.Code, nethttp.StatusUnauthorized)
		}
	}

	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/sessoes", strings.NewReader(corpo))
	requisicao.RemoteAddr = "192.0.2.10:1234"
	resposta := httptest.NewRecorder()
	handler.ServeHTTP(resposta, requisicao)
	if resposta.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusTooManyRequests)
	}
}
