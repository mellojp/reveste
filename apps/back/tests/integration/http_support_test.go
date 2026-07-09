package integration_test

import (
	"context"
	"io"
	"log/slog"
	nethttp "net/http"
	"time"

	"reveste/apps/back/internal/adaptadores/pagamentos"
	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/anuncios"
	"reveste/apps/back/internal/dominio/cadastros"
	"reveste/apps/back/internal/dominio/compras"
	"reveste/apps/back/internal/dominio/interacao"
	httptransport "reveste/apps/back/internal/http"
	"reveste/apps/back/internal/transporte"
	"reveste/apps/back/internal/web"
)

type operacoesHTTP struct{}

const hostBlobTeste = "reveste-test.public.blob.vercel-storage.com"

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

func (operacoesHTTP) AtualizarUsuario(context.Context, cadastros.Usuario) error {
	return nil
}

func (operacoesHTTP) BuscarUsuarioPorID(_ context.Context, id string) (cadastros.Usuario, error) {
	if id == "vendedor-1" {
		return cadastros.Usuario{
			ID: id, Nome: "Vendedora Teste", Email: "privado@teste.local",
			Telefone:          "79999999999",
			EnderecoPrincipal: cadastros.Endereco{Cidade: "Aracaju", Estado: "SE"},
			CriadoEm:          time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
		}, nil
	}
	return cadastros.Usuario{}, common.ErrNaoEncontrado
}

func (operacoesHTTP) ReativarVendedor(context.Context, string, time.Time) (bool, error) {
	return true, nil
}

func (operacoesHTTP) CriarNotificacao(context.Context, interacao.Notificacao) error {
	return nil
}

func (operacoesHTTP) ListarNotificacoes(context.Context, string, int) ([]interacao.Notificacao, error) {
	return nil, nil
}

func (operacoesHTTP) ContarNotificacoesNaoLidas(context.Context, string) (int, error) {
	return 0, nil
}

func (operacoesHTTP) MarcarNotificacoesLidas(context.Context, string, time.Time) error {
	return nil
}

func (operacoesHTTP) RemoverNotificacao(context.Context, string, string) error {
	return nil
}

func (operacoesHTTP) LimparNotificacoes(context.Context, string) error {
	return nil
}

func (operacoesHTTP) BuscarParticipantesPedido(_ context.Context, idPedido string) (string, string, error) {
	if idPedido == "pedido-http" {
		return "comprador-1", "vendedor-1", nil
	}
	return "", "", common.ErrNaoEncontrado
}

func (operacoesHTTP) ObterOuCriarConversa(context.Context, string, string, time.Time) (string, error) {
	return "conversa-http", nil
}

func (operacoesHTTP) ListarMensagens(context.Context, string) ([]interacao.Mensagem, error) {
	return nil, nil
}

func (operacoesHTTP) CriarMensagem(context.Context, interacao.Mensagem) error {
	return nil
}

func (operacoesHTTP) BuscarUsuarioPorEmailOuCPF(
	_ context.Context,
	identificador string,
) (cadastros.Usuario, error) {
	if identificador != "vendedora@teste.local" {
		return cadastros.Usuario{}, common.ErrNaoEncontrado
	}
	return cadastros.Usuario{
		ID: "vendedor-1", Nome: "Vendedora Teste", Email: identificador, HashSenha: "hash",
		EnderecoPrincipal: cadastros.Endereco{
			CEP: "49000000", Logradouro: "Rua Teste", Numero: "10",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "SE",
		},
		CriadoEm: time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (operacoesHTTP) ListarEnderecos(context.Context, string) ([]cadastros.Endereco, error) {
	return []cadastros.Endereco{
		{
			ID: "endereco-1", CEP: "49000000", Logradouro: "Rua Teste", Numero: "10",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "SE", Principal: true,
		},
		{
			ID: "endereco-2", CEP: "01310200", Logradouro: "Avenida Paulista", Numero: "1000",
			Bairro: "Bela Vista", Cidade: "São Paulo", Estado: "SP",
		},
	}, nil
}

func (operacoesHTTP) BuscarEndereco(_ context.Context, _, idEndereco string) (cadastros.Endereco, error) {
	if idEndereco != "endereco-2" {
		return cadastros.Endereco{}, common.ErrNaoEncontrado
	}
	return cadastros.Endereco{
		ID: "endereco-2", CEP: "01310200", Logradouro: "Avenida Paulista", Numero: "1000",
		Bairro: "Bela Vista", Cidade: "São Paulo", Estado: "SP",
	}, nil
}

func (operacoesHTTP) AdicionarEndereco(context.Context, string, cadastros.Endereco, time.Time) error {
	return nil
}

func (operacoesHTTP) AtualizarEndereco(context.Context, string, string, cadastros.Endereco, time.Time) error {
	return nil
}

func (operacoesHTTP) RemoverEndereco(context.Context, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) DefinirEnderecoPrincipal(context.Context, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) CriarAnuncio(context.Context, anuncios.Anuncio) error {
	return nil
}

func (operacoesHTTP) AtualizarAnuncio(context.Context, anuncios.Anuncio) error {
	return nil
}

func (operacoesHTTP) ExcluirAnuncio(context.Context, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) BuscarAnuncioPorID(_ context.Context, id string) (anuncios.Anuncio, error) {
	if id == "anuncio-publico" {
		return anuncios.Anuncio{
			ID: id, IDVendedor: "vendedor-1", Titulo: "Casaco de lã",
			Descricao: "Casaco em ótimo estado.", Categoria: anuncios.CategoriaCasacos,
			Tamanho: "M", Cor: "verde", EstadoConservacao: anuncios.EstadoSeminovo,
			PrecoCentavos: 12_000, Status: anuncios.StatusAnuncioDisponivel,
			PesoGramas: 900, AlturaCm: 8, LarguraCm: 30, ComprimentoCm: 40,
			Fotos: []anuncios.Foto{
				{ID: "foto-1", URL: "https://reveste-test.public.blob.vercel-storage.com/casaco-1.jpg", Ordem: 0},
				{ID: "foto-2", URL: "https://reveste-test.public.blob.vercel-storage.com/casaco-2.jpg", Ordem: 1},
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

func (operacoesHTTP) BuscarCompraPorChave(context.Context, string) (compras.Compra, error) {
	return compras.Compra{}, common.ErrNaoEncontrado
}

func (operacoesHTTP) BuscarCompraPendenteDoComprador(context.Context, string) (compras.Compra, error) {
	return compras.Compra{}, common.ErrNaoEncontrado
}

func (operacoesHTTP) IniciarCompra(_ context.Context, compra compras.Compra, _ compras.Pagamento, _ string) (compras.Compra, bool, error) {
	return compra, true, nil
}

func (operacoesHTTP) ConfirmarCompraAprovada(_ context.Context, _ string, _, _ string, _ time.Time) (compras.Compra, error) {
	return compras.Compra{ID: "compra-1", Status: compras.StatusCompraAprovada}, nil
}

func (operacoesHTTP) RecusarCompra(context.Context, string, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) ExpirarComprasPendentes(context.Context, time.Time) (int, error) {
	return 0, nil
}

func (operacoesHTTP) ListarPedidosDoComprador(context.Context, string) ([]compras.Pedido, error) {
	return []compras.Pedido{{
		ID: "pedido-1", Status: compras.StatusPedidoAguardandoEnvio,
		ValorTotalItensCentavos: 12_000, ValorFreteCentavos: 1990,
		NomeDestinatario: "Comprador Teste",
		EnderecoEntrega: cadastros.Endereco{
			Logradouro: "Rua Teste", Numero: "10", Cidade: "Aracaju", Estado: "SE",
		},
		Itens: []compras.ItemPedido{{
			IDAnuncio: "anuncio-publico", Titulo: "Casaco de lã",
			Categoria: anuncios.CategoriaCasacos, Tamanho: "M",
			EstadoConservacao: anuncios.EstadoSeminovo, Status: compras.StatusItemAguardandoEnvio,
			ValorUnitarioCentavos: 12_000,
		}},
		CriadoEm: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	}}, nil
}

func (operacoesHTTP) ListarPedidosDoVendedor(context.Context, string) ([]compras.Pedido, error) {
	return []compras.Pedido{{
		ID: "venda-1", Status: compras.StatusPedidoAguardandoEnvio,
		ValorTotalItensCentavos: 12_000, ValorFreteCentavos: 1990,
		ValorLiquidoVendedorCentavos: 12_990, NomeDestinatario: "Comprador Teste",
		EnderecoEntrega: cadastros.Endereco{
			CEP: "49000000", Logradouro: "Rua Teste", Numero: "10",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "SE",
		},
		Itens: []compras.ItemPedido{{
			IDAnuncio: "anuncio-publico", Titulo: "Casaco de lã",
			Categoria: anuncios.CategoriaCasacos, Tamanho: "M",
			EstadoConservacao: anuncios.EstadoSeminovo, Status: compras.StatusItemAguardandoEnvio,
			ValorUnitarioCentavos: 12_000,
		}},
		CriadoEm: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	}}, nil
}

func (operacoesHTTP) BuscarPedidoDoComprador(_ context.Context, _, idPedido string) (compras.Pedido, error) {
	return compras.Pedido{
		ID: idPedido, Status: compras.StatusPedidoFinalizado,
		ValorTotalItensCentavos: 12_000, ValorFreteCentavos: 1990,
		NomeDestinatario: "Comprador Teste",
		EnderecoEntrega: cadastros.Endereco{
			Logradouro: "Rua Teste", Numero: "10", Cidade: "Aracaju", Estado: "SE",
		},
		Itens: []compras.ItemPedido{{
			IDAnuncio: "anuncio-publico", Titulo: "Casaco de lã",
			Categoria: anuncios.CategoriaCasacos, Tamanho: "M",
			EstadoConservacao: anuncios.EstadoSeminovo, Status: compras.StatusItemRecebido,
			ValorUnitarioCentavos: 12_000,
		}},
		CriadoEm: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (operacoesHTTP) BuscarPedidoDoVendedor(_ context.Context, _, idPedido string) (compras.Pedido, error) {
	return compras.Pedido{
		ID: idPedido, Status: compras.StatusPedidoAguardandoEnvio,
		ValorTotalItensCentavos: 12_000, ValorFreteCentavos: 1990,
		ValorLiquidoVendedorCentavos: 12_990, NomeDestinatario: "Comprador Teste",
		EnderecoEntrega: cadastros.Endereco{
			CEP: "49000000", Logradouro: "Rua Teste", Numero: "10",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "SE",
		},
		Itens: []compras.ItemPedido{{
			IDAnuncio: "anuncio-publico", Titulo: "Casaco de lã",
			Categoria: anuncios.CategoriaCasacos, Tamanho: "M",
			EstadoConservacao: anuncios.EstadoSeminovo, Status: compras.StatusItemAguardandoEnvio,
			ValorUnitarioCentavos: 12_000,
		}},
		CriadoEm: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (operacoesHTTP) BuscarAvaliacaoDoPedido(context.Context, string) (interacao.Avaliacao, error) {
	return interacao.Avaliacao{}, common.ErrNaoEncontrado
}

func (operacoesHTTP) MarcarPedidoEnviado(context.Context, string, string, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) ConfirmarRecebimentoPedido(context.Context, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) RegistrarAvaliacao(context.Context, interacao.Avaliacao) error {
	return nil
}

func (operacoesHTTP) ProcessarItensVencidos(context.Context, time.Time, int) (int, error) {
	return 0, nil
}

func (operacoesHTTP) MediaAvaliacoesVendedor(context.Context, string) (casosdeuso.MediaAvaliacoes, error) {
	return casosdeuso.MediaAvaliacoes{Media: 4.5, Quantidade: 2}, nil
}

func (operacoesHTTP) CriarSessao(context.Context, string, string, time.Time) error {
	return nil
}

func (operacoesHTTP) BuscarUsuarioDaSessao(_ context.Context, token string, _ time.Time) (string, error) {
	if token == "sessao-valida" {
		return "vendedor-1", nil
	}
	return "", common.ErrNaoAutorizado
}

func (operacoesHTTP) RemoverSessao(context.Context, string) error {
	return nil
}

func (operacoesHTTP) Ping(context.Context) error {
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
	return time.Date(2030, 6, 10, 12, 0, 0, 0, time.UTC)
}

func novoHandler() nethttp.Handler {
	operacoes := operacoesHTTP{}
	cadastrosCU := casosdeuso.NovoControladorCadastro(
		operacoes, operacoes, idHTTP{}, senhaHTTP{}, relogioHTTP{},
	)
	anunciosCU := casosdeuso.NovoControladorAnuncio(
		operacoes, operacoes, idHTTP{}, relogioHTTP{},
		hostBlobTeste,
	)
	comprasCU := casosdeuso.NovoControladorCarrinho(
		operacoes, operacoes, idHTTP{}, relogioHTTP{},
	)
	uploadsCU := casosdeuso.NovoControladorUpload(operacoes, idHTTP{}, relogioHTTP{})
	checkoutCU := casosdeuso.NovoControladorCheckout(
		operacoes, operacoes, operacoes, operacoes, operacoes, pagamentos.NovoSimulado(),
		nil, idHTTP{}, relogioHTTP{},
		compras.PoliticaCobranca{TaxaServicoPercentual: 10, FretePorPedidoCentavos: 1990},
	)
	notificacoesCU := casosdeuso.NovoControladorNotificacoes(operacoes, relogioHTTP{})
	pedidosCU := casosdeuso.NovoControladorPedidos(operacoes, operacoes, idHTTP{}, relogioHTTP{})
	vendedoresCU := casosdeuso.NovoControladorVendedor(
		operacoes, pagamentos.NovoSimulado(), relogioHTTP{}, cadastros.TaxaReativacaoCentavos,
	)
	conversasCU := casosdeuso.NovoControladorConversas(operacoes, operacoes, idHTTP{}, relogioHTTP{})
	cepCU := casosdeuso.NovoControladorCEP(consultorCEPHTTP{})
	limitador := transporte.NovoLimitadorLogin(transporte.NovoRegistroMemoria())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// confiarProxy = true: os testes simulam HTTPS via X-Forwarded-Proto, como atrás de um proxy.
	paginasHTML, err := web.NovoAdaptadorPaginas(cadastrosCU, anunciosCU, comprasCU, checkoutCU, pedidosCU, vendedoresCU, notificacoesCU, conversasCU, limitador, true, "", logger)
	if err != nil {
		panic(err)
	}
	return httptransport.NovaAPI(
		cadastrosCU, anunciosCU, comprasCU, uploadsCU, checkoutCU, pedidosCU, vendedoresCU, notificacoesCU, conversasCU, cepCU, operacoes, logger, hostBlobTeste, limitador, true, nil, "", paginasHTML,
	)
}

type consultorCEPHTTP struct{}

func (consultorCEPHTTP) ConsultarCEP(context.Context, string) (cadastros.Endereco, error) {
	return cadastros.Endereco{}, common.ErrNaoEncontrado
}
