package integration_test

import (
	nethttp "net/http"
	"net/http/httptest"
	"net/url"
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
	if !strings.Contains(resposta.Body.String(), `src="/js/htmx.min.js"`) {
		t.Fatalf("HTMX local nao foi carregado: %s", resposta.Body.String())
	}
}

func TestRotaProtegidaDoFrontendRedirecionaParaLogin(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/perfil", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusSeeOther {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusSeeOther)
	}
	if local := resposta.Header().Get("Location"); !strings.HasPrefix(local, "/entrar?retorno=") {
		t.Fatalf("redirecionamento inesperado: %s", local)
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

func TestDetalheSSRPreservaDadosPublicosSemExporContatoPrivado(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/anuncios/anuncio-publico", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	corpo := resposta.Body.String()
	if !strings.Contains(corpo, "Casaco de lã") || !strings.Contains(corpo, "Vendedora Teste") {
		t.Fatalf("detalhe SSR inesperado: %s", corpo)
	}
	if strings.Contains(corpo, "privado@teste.local") || strings.Contains(corpo, "79999999999") {
		t.Fatalf("dados privados expostos no SSR: %s", corpo)
	}
	if !strings.Contains(corpo, "data-gallery-photo=") {
		t.Fatalf("controles da galeria ausentes: %s", corpo)
	}
}

func TestFormularioSSRDeAnuncioMantemContratoComJavascriptDeUpload(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/vender", nil)
	requisicao.AddCookie(&nethttp.Cookie{Name: "reveste_session", Value: "sessao-valida"})
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	corpo := resposta.Body.String()
	for _, atributo := range []string{"data-ad-form", "data-photo-input", "data-photo-preview"} {
		if !strings.Contains(corpo, atributo) {
			t.Fatalf("atributo %s ausente no formulario SSR", atributo)
		}
	}
}

func TestCadastroSSROfereceFluxoAcessivelEProgressivo(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/cadastro", nil)
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusOK {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
	}
	corpo := resposta.Body.String()
	for _, trecho := range []string{
		`data-register-form`,
		`data-registration-section="identity"`,
		`data-registration-section="address"`,
		`name="confirmar_senha"`,
		`autocomplete="new-password"`,
		`data-password-strength`,
	} {
		if !strings.Contains(corpo, trecho) {
			t.Fatalf("contrato do cadastro SSR não contém %q", trecho)
		}
	}
}

func TestCadastroWebRejeitaConfirmacaoDeSenhaDiferente(t *testing.T) {
	formulario := url.Values{
		"nome":            {"Pessoa Teste"},
		"cpf":             {"529.982.247-25"},
		"email":           {"pessoa@teste.local"},
		"senha":           {"senha-segura"},
		"confirmar_senha": {"outra-senha"},
		"cep":             {"49000-000"},
		"logradouro":      {"Rua Teste"},
		"numero":          {"10"},
		"bairro":          {"Centro"},
		"cidade":          {"Aracaju"},
		"estado":          {"SE"},
	}
	requisicao := httptest.NewRequest(
		nethttp.MethodPost,
		"/cadastro",
		strings.NewReader(formulario.Encode()),
	)
	requisicao.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	requisicao.Header.Set("Origin", "http://example.com")
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusUnprocessableEntity {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusUnprocessableEntity)
	}
	corpo := resposta.Body.String()
	if !strings.Contains(corpo, "As senhas informadas não coincidem.") ||
		!strings.Contains(corpo, `aria-describedby="erro-confirmar_senha"`) ||
		!strings.Contains(corpo, `id="erro-confirmar_senha"`) {
		t.Fatalf("mensagem acessível de confirmação ausente: %s", corpo)
	}
}

func TestLoginHTMXUsaLocationParaPreservarTransicao(t *testing.T) {
	formulario := url.Values{
		"identificador": {"vendedora@teste.local"},
		"senha":         {"senha-segura"},
		"retorno":       {"/perfil"},
	}
	requisicao := httptest.NewRequest(
		nethttp.MethodPost,
		"/entrar",
		strings.NewReader(formulario.Encode()),
	)
	requisicao.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	requisicao.Header.Set("Origin", "http://example.com")
	requisicao.Header.Set("HX-Request", "true")
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusNoContent {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusNoContent)
	}
	if local := resposta.Header().Get("HX-Location"); local != "/perfil" {
		t.Fatalf("HX-Location = %q; esperado /perfil", local)
	}
	if redirecionamento := resposta.Header().Get("HX-Redirect"); redirecionamento != "" {
		t.Fatalf("HX-Redirect inesperado: %q", redirecionamento)
	}
}

func TestLogoutHTMXUsaLocationParaPreservarTransicao(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/sair", nil)
	requisicao.Header.Set("Origin", "http://example.com")
	requisicao.Header.Set("HX-Request", "true")
	requisicao.AddCookie(&nethttp.Cookie{Name: "reveste_session", Value: "sessao-valida"})
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusNoContent {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusNoContent)
	}
	if local := resposta.Header().Get("HX-Location"); local != "/" {
		t.Fatalf("HX-Location = %q; esperado /", local)
	}
	if redirecionamento := resposta.Header().Get("HX-Redirect"); redirecionamento != "" {
		t.Fatalf("HX-Redirect inesperado: %q", redirecionamento)
	}
}

func TestFormularioWebRejeitaPostSemOrigem(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodPost, "/cadastro", strings.NewReader(""))
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Code != nethttp.StatusForbidden {
		t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusForbidden)
	}
}

func TestPaginasSSRRenderizamDocumentosCompletos(t *testing.T) {
	casos := []struct {
		rota        string
		autenticada bool
	}{
		{rota: "/"},
		{rota: "/catalogo"},
		{rota: "/anuncios/anuncio-publico"},
		{rota: "/vendedores/vendedor-1"},
		{rota: "/entrar"},
		{rota: "/cadastro"},
		{rota: "/perfil", autenticada: true},
		{rota: "/perfil/editar", autenticada: true},
		{rota: "/perfil/enderecos", autenticada: true},
		{rota: "/perfil/enderecos/endereco-2/editar", autenticada: true},
		{rota: "/meus-anuncios", autenticada: true},
		{rota: "/vender", autenticada: true},
		{rota: "/carrinho", autenticada: true},
		{rota: "/meus-pedidos", autenticada: true},
		{rota: "/meus-pedidos/pedido-1", autenticada: true},
		{rota: "/minhas-vendas", autenticada: true},
		{rota: "/minhas-vendas/venda-1", autenticada: true},
	}

	for _, caso := range casos {
		t.Run(caso.rota, func(t *testing.T) {
			requisicao := httptest.NewRequest(nethttp.MethodGet, caso.rota, nil)
			if caso.autenticada {
				requisicao.AddCookie(&nethttp.Cookie{Name: "reveste_session", Value: "sessao-valida"})
			}
			resposta := httptest.NewRecorder()

			novoHandler().ServeHTTP(resposta, requisicao)

			if resposta.Code != nethttp.StatusOK {
				t.Fatalf("status = %d; esperado %d", resposta.Code, nethttp.StatusOK)
			}
			if !strings.Contains(resposta.Body.String(), "</html>") {
				t.Fatalf("documento SSR incompleto: %s", resposta.Body.String())
			}
		})
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
	requisicao.Header.Set("X-Forwarded-Proto", "https")
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("Content-Security-Policy nao foi definido")
	}
	if resposta.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", resposta.Header().Get("X-Content-Type-Options"))
	}
	if resposta.Header().Get("Cross-Origin-Opener-Policy") != "same-origin" {
		t.Fatalf("Cross-Origin-Opener-Policy = %q", resposta.Header().Get("Cross-Origin-Opener-Policy"))
	}
	if resposta.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("Strict-Transport-Security nao foi definido para HTTPS")
	}
	csp := resposta.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, "img-src 'self' blob: https:;") ||
		strings.Contains(csp, "*.public.blob.vercel-storage.com") ||
		strings.Contains(csp, "fonts.googleapis.com") {
		t.Fatalf("CSP possui origem ampla ou externa inesperada: %s", csp)
	}
	if !strings.Contains(csp, "https://reveste-test.public.blob.vercel-storage.com") {
		t.Fatalf("CSP nao restringe imagens ao store configurado: %s", csp)
	}
}

func TestAPIImpedeCacheDeDadosPrivados(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/me", nil)
	requisicao.AddCookie(&nethttp.Cookie{Name: "reveste_session", Value: "sessao-valida"})
	resposta := httptest.NewRecorder()

	novoHandler().ServeHTTP(resposta, requisicao)

	if resposta.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", resposta.Header().Get("Cache-Control"))
	}
}
