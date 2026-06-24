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
		requisicao.Header.Set("Origin", "http://example.com")
		resposta := httptest.NewRecorder()
		handler.ServeHTTP(resposta, requisicao)
		if resposta.Code != nethttp.StatusUnauthorized {
			t.Fatalf("tentativa %d: status = %d; esperado %d", tentativa, resposta.Code, nethttp.StatusUnauthorized)
		}
	}

	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/sessoes", strings.NewReader(corpo))
	requisicao.RemoteAddr = "192.0.2.10:1234"
	requisicao.Header.Set("Origin", "http://example.com")
	resposta := httptest.NewRecorder()
	handler.ServeHTTP(resposta, requisicao)
	if resposta.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusTooManyRequests)
	}
}

func TestSessaoDoNavegadorUsaCookieHttpOnlySemExporToken(t *testing.T) {
	corpo := `{"identificador":"vendedora@teste.local","senha":"senha-valida"}`
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/sessoes", strings.NewReader(corpo))
	requisicao.Header.Set("X-Forwarded-Proto", "https")
	requisicao.Header.Set("Origin", "https://example.com")
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusCreated)
	}
	cookies := resposta.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d; esperado 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "reveste_session" || !cookie.HttpOnly || !cookie.Secure ||
		cookie.SameSite != nethttp.SameSiteLaxMode {
		t.Fatalf("cookie de sessao inseguro: %#v", cookie)
	}
	if strings.Contains(resposta.Body.String(), `"token"`) {
		t.Fatalf("token exposto no JSON: %s", resposta.Body.String())
	}
}

func TestRequisicaoComCookieRejeitaOrigemExterna(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodPatch, "/v1/me", strings.NewReader(`{}`))
	requisicao.AddCookie(&nethttp.Cookie{Name: "reveste_session", Value: "sessao-valida"})
	requisicao.Header.Set("Origin", "https://atacante.example")
	requisicao.Header.Set("Sec-Fetch-Site", "cross-site")
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusForbidden {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusForbidden)
	}
	if !strings.Contains(resposta.Body.String(), `"ORIGEM_NAO_PERMITIDA"`) {
		t.Fatalf("resposta inesperada: %s", resposta.Body.String())
	}
}

func TestCookieAutenticaRequisicaoDoFrontend(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/me", nil)
	requisicao.AddCookie(&nethttp.Cookie{Name: "reveste_session", Value: "sessao-valida"})
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	if !strings.Contains(resposta.Body.String(), `"nome":"Vendedora Teste"`) {
		t.Fatalf("perfil inesperado: %s", resposta.Body.String())
	}
}

func TestLoginDoNavegadorRejeitaOrigemExterna(t *testing.T) {
	corpo := `{"identificador":"vendedora@teste.local","senha":"senha-valida"}`
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/sessoes", strings.NewReader(corpo))
	requisicao.Header.Set("Origin", "https://atacante.example")
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusForbidden {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusForbidden)
	}
}

func TestClienteBearerPodeAutenticarSemOrigin(t *testing.T) {
	corpo := `{"identificador":"vendedora@teste.local","senha":"senha-valida"}`
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/v1/sessoes", strings.NewReader(corpo))
	requisicao.Header.Set("X-Reveste-Session-Transport", "bearer")
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusCreated)
	}
	if !strings.Contains(resposta.Body.String(), `"token"`) {
		t.Fatalf("token Bearer ausente: %s", resposta.Body.String())
	}
}
