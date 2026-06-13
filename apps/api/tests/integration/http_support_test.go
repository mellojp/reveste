package integration_test

import (
	"context"
	"io"
	"log/slog"
	nethttp "net/http"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
	httptransport "reveste/apps/api/internal/http"
)

type operacoesHTTP struct{}

func (operacoesHTTP) AutorizarUpload(
	context.Context,
	casosdeuso.SolicitacaoUpload,
) (casosdeuso.AutorizacaoUpload, error) {
	return casosdeuso.AutorizacaoUpload{
		URLUpload: "https://vercel.com/api/blob/", Pathname: "anuncios/teste/foto.jpg",
		Token: "token", TiposAceitos: casosdeuso.TiposImagemPermitidos,
		TamanhoMaximoBytes: casosdeuso.TamanhoMaximoImagemBytes,
	}, nil
}

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

func (operacoesHTTP) BuscarAnuncioPorID(_ context.Context, id string) (anuncios.Anuncio, error) {
	if id == "anuncio-publico" {
		return anuncios.Anuncio{
			ID: id, IDVendedor: "vendedor-1", Titulo: "Casaco de lã",
			Descricao: "Casaco em ótimo estado.", Categoria: anuncios.CategoriaCasacos,
			Tamanho: "M", Cor: "verde", EstadoConservacao: anuncios.EstadoSeminovo,
			PrecoCentavos: 12_000, Status: anuncios.StatusAnuncioDisponivel,
			Fotos: []anuncios.Foto{
				{ID: "foto-1", URL: "https://example.com/casaco-1.jpg", Ordem: 0},
				{ID: "foto-2", URL: "https://example.com/casaco-2.jpg", Ordem: 1},
			},
		}, nil
	}
	return anuncios.Anuncio{}, common.ErrNaoEncontrado
}

func (operacoesHTTP) ListarAnuncios(
	context.Context,
	casosdeuso.FiltroAnuncios,
) ([]anuncios.Anuncio, error) {
	return []anuncios.Anuncio{}, nil
}

func (operacoesHTTP) ObterOuCriarCarrinho(
	context.Context,
	string,
	string,
	time.Time,
) (compras.Carrinho, error) {
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
	uploadsCU := casosdeuso.NovoControladorUpload(operacoes, idHTTP{}, relogioHTTP{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httptransport.NovaAPI(cadastrosCU, anunciosCU, comprasCU, uploadsCU, logger)
}
