package http_test

import (
	"context"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
	httptransport "reveste/apps/api/internal/http"
)

type operacoesHTTP struct{}

func (operacoesHTTP) CriarUsuario(context.Context, cadastros.Usuario) error {
	return nil
}

func (operacoesHTTP) BuscarUsuarioPorID(context.Context, string) (cadastros.Usuario, error) {
	return cadastros.Usuario{}, common.ErrNaoEncontrado
}

func (operacoesHTTP) BuscarUsuarioPorEmailOuCPF(context.Context, string) (cadastros.Usuario, error) {
	return cadastros.Usuario{}, common.ErrNaoEncontrado
}

func (operacoesHTTP) CriarAnuncio(context.Context, anuncios.Anuncio) error {
	return nil
}

func (operacoesHTTP) BuscarAnuncioPorID(context.Context, string) (anuncios.Anuncio, error) {
	return anuncios.Anuncio{}, common.ErrNaoEncontrado
}

func (operacoesHTTP) ListarAnuncios(
	context.Context,
	casosdeuso.FiltroAnuncios,
) ([]anuncios.Anuncio, error) {
	return []anuncios.Anuncio{}, nil
}

func (operacoesHTTP) ObterOuCriarCarrinho(context.Context, string, string, time.Time) (compras.Carrinho, error) {
	return compras.Carrinho{}, nil
}

func (operacoesHTTP) AdicionarAnuncioAoCarrinho(
	context.Context, string, string, string, time.Time,
) (compras.Carrinho, error) {
	return compras.Carrinho{}, nil
}

func (operacoesHTTP) RemoverAnuncioDoCarrinho(
	context.Context, string, string, string, time.Time,
) (compras.Carrinho, error) {
	return compras.Carrinho{}, nil
}

func (operacoesHTTP) CriarSessao(context.Context, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) BuscarUsuarioDaSessao(context.Context, string, time.Time) (string, error) {
	return "", common.ErrNaoAutorizado
}

func (operacoesHTTP) RemoverSessao(context.Context, string) error {
	return nil
}

type idHTTP struct{}

func (idHTTP) Novo() string {
	return "00000000-0000-4000-8000-000000000001"
}

type senhaHTTP struct{}

func (senhaHTTP) Gerar(string) (string, error) {
	return "hash", nil
}

func (senhaHTTP) Comparar(string, string) bool {
	return true
}

type relogioHTTP struct{}

func (relogioHTTP) Agora() time.Time {
	return time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
}

func novoHandler() nethttp.Handler {
	operacoes := operacoesHTTP{}
	cadastrosCU := casosdeuso.NovoControladorCadastro(
		operacoes, operacoes, idHTTP{}, senhaHTTP{}, relogioHTTP{},
	)
	anunciosCU := casosdeuso.NovoControladorAnuncio(
		operacoes, operacoes, idHTTP{}, relogioHTTP{},
	)
	comprasCU := casosdeuso.NovoControladorCarrinho(
		operacoes, operacoes, idHTTP{}, relogioHTTP{},
	)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httptransport.NovaAPI(cadastrosCU, anunciosCU, comprasCU, logger)
}

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

func TestCatalogoRejeitaFiltroInvalido(t *testing.T) {
	requisicao := httptest.NewRequest(nethttp.MethodGet, "/v1/anuncios?deslocamento=-1", nil)
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
