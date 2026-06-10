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
	casosdeusoanuncios "reveste/apps/api/internal/casosdeuso/anuncios"
	casosdeusocadastros "reveste/apps/api/internal/casosdeuso/cadastros"
	casosdeusocompras "reveste/apps/api/internal/casosdeuso/compras"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
	errosdominio "reveste/apps/api/internal/dominio/erros"
	httptransport "reveste/apps/api/internal/http"
)

type operacoesHTTP struct{}

func (operacoesHTTP) CriarUsuario(context.Context, cadastros.Usuario) error {
	return nil
}

func (operacoesHTTP) BuscarUsuarioPorID(context.Context, string) (cadastros.Usuario, error) {
	return cadastros.Usuario{}, errosdominio.ErrNaoEncontrado
}

func (operacoesHTTP) BuscarUsuarioPorEmailOuCPF(context.Context, string) (cadastros.Usuario, error) {
	return cadastros.Usuario{}, errosdominio.ErrNaoEncontrado
}

func (operacoesHTTP) CriarAnuncio(context.Context, anuncios.Anuncio) error {
	return nil
}

func (operacoesHTTP) BuscarAnuncioPorID(context.Context, string) (anuncios.Anuncio, error) {
	return anuncios.Anuncio{}, errosdominio.ErrNaoEncontrado
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

func (operacoesHTTP) SalvarCarrinho(context.Context, compras.Carrinho) error {
	return nil
}

func (operacoesHTTP) CriarSessao(context.Context, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) BuscarUsuarioDaSessao(context.Context, string, time.Time) (string, error) {
	return "", errosdominio.ErrNaoAutorizado
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
	cadastrosCU := casosdeusocadastros.NovoFluxoCadastro(
		operacoes, operacoes, idHTTP{}, senhaHTTP{}, relogioHTTP{},
	)
	anunciosCU := casosdeusoanuncios.NovoFluxoAnuncio(
		operacoes, operacoes, idHTTP{}, relogioHTTP{},
	)
	comprasCU := casosdeusocompras.NovoFluxoCarrinho(
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
