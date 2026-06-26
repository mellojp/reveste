package web

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	nethttp "net/http"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/transporte"
)

//go:embed templates/*.html
var arquivosTemplates embed.FS

type AdaptadorPaginas struct {
	controladorCadastro     *casosdeuso.ControladorCadastro
	controladorAnuncio      *casosdeuso.ControladorAnuncio
	controladorCarrinho     *casosdeuso.ControladorCarrinho
	controladorCheckout     *casosdeuso.ControladorCheckout
	controladorPedidos      *casosdeuso.ControladorPedidos
	controladorVendedor     *casosdeuso.ControladorVendedor
	controladorNotificacoes *casosdeuso.ControladorNotificacoes
	controladorConversas    *casosdeuso.ControladorConversas
	documentosHTML          *template.Template
	logger                  *slog.Logger
	limitador               *transporte.LimitadorLogin
	confiarProxy            bool
	chavePublicaPagamento   string
}

func NovoAdaptadorPaginas(
	controladorCadastro *casosdeuso.ControladorCadastro,
	controladorAnuncio *casosdeuso.ControladorAnuncio,
	controladorCarrinho *casosdeuso.ControladorCarrinho,
	controladorCheckout *casosdeuso.ControladorCheckout,
	controladorPedidos *casosdeuso.ControladorPedidos,
	controladorVendedor *casosdeuso.ControladorVendedor,
	controladorNotificacoes *casosdeuso.ControladorNotificacoes,
	controladorConversas *casosdeuso.ControladorConversas,
	limitador *transporte.LimitadorLogin,
	confiarProxy bool,
	chavePublicaPagamento string,
	logger *slog.Logger,
) (nethttp.Handler, error) {
	tmpl, err := template.New("web").
		Funcs(funcoesApresentacaoTemplates()).
		ParseFS(arquivosTemplates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("carregar templates web: %w", err)
	}
	adaptador := &AdaptadorPaginas{
		controladorCadastro:     controladorCadastro,
		controladorAnuncio:      controladorAnuncio,
		controladorCarrinho:     controladorCarrinho,
		controladorCheckout:     controladorCheckout,
		controladorPedidos:      controladorPedidos,
		controladorVendedor:     controladorVendedor,
		controladorNotificacoes: controladorNotificacoes,
		controladorConversas:    controladorConversas,
		documentosHTML:          tmpl,
		logger:                  logger,
		limitador:               limitador,
		confiarProxy:            confiarProxy,
		chavePublicaPagamento:   chavePublicaPagamento,
	}
	mux := nethttp.NewServeMux()
	adaptador.registrarConsultasPaginas(mux)
	adaptador.registrarComandosFormularios(mux)
	return mux, nil
}

func (a *AdaptadorPaginas) registrarConsultasPaginas(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /", a.exibirPaginaInicial)
	mux.HandleFunc("GET /catalogo", a.exibirCatalogo)
	mux.HandleFunc("GET /fragmentos/catalogo", a.exibirProximoLoteCatalogo)
	mux.HandleFunc("GET /anuncios/{idAnuncio}", a.exibirDetalheAnuncio)
	mux.HandleFunc("GET /vendedores/{idVendedor}", a.exibirPerfilPublicoVendedor)
	mux.HandleFunc("GET /usuarios/{idUsuario}", a.exibirPerfilPublicoUsuario)
	mux.HandleFunc("GET /entrar", a.exibirLogin)
	mux.HandleFunc("GET /cadastro", a.exibirCadastroUsuario)
	mux.HandleFunc("GET /perfil", a.exigirSessao(a.exibirPerfilUsuario))
	mux.HandleFunc("GET /perfil/editar", a.exigirSessao(a.exibirEdicaoPerfilUsuario))
	mux.HandleFunc("GET /perfil/enderecos", a.exigirSessao(a.exibirEnderecos))
	mux.HandleFunc("GET /perfil/enderecos/{idEndereco}/editar", a.exigirSessao(a.exibirEdicaoEndereco))
	mux.HandleFunc("GET /meus-anuncios", a.exigirSessao(a.exibirAnunciosUsuario))
	mux.HandleFunc("GET /meus-anuncios/{idAnuncio}/editar", a.exigirSessao(a.exibirEdicaoAnuncio))
	mux.HandleFunc("GET /vender", a.exigirSessao(a.exibirPublicacaoAnuncio))
	mux.HandleFunc("GET /carrinho", a.exigirSessao(a.exibirCarrinhoUsuario))
	mux.HandleFunc("GET /checkout", a.exigirSessao(a.exibirCheckout))
	mux.HandleFunc("GET /checkout/cartao", a.exigirSessao(a.exibirCheckoutCartao))
	mux.HandleFunc("GET /checkout/pagamento", a.exigirSessao(a.exibirPagamento))
	mux.HandleFunc("GET /checkout/pagamento/status", a.exigirSessao(a.exibirStatusPagamento))
	mux.HandleFunc("GET /meus-pedidos", a.exigirSessao(a.exibirPedidosUsuario))
	mux.HandleFunc("GET /meus-pedidos/{idPedido}", a.exigirSessao(a.exibirDetalhePedido))
	mux.HandleFunc("GET /minhas-vendas", a.exigirSessao(a.exibirVendasUsuario))
	mux.HandleFunc("GET /minhas-vendas/{idPedido}", a.exigirSessao(a.exibirDetalheVenda))
	mux.HandleFunc("GET /notificacoes", a.exigirSessao(a.exibirNotificacoes))
	mux.HandleFunc("GET /pedidos/{idPedido}/conversa", a.exigirSessao(a.exibirConversa))
	mux.HandleFunc("GET /pedidos/{idPedido}/conversa/mensagens", a.exigirSessao(a.exibirMensagensConversa))
}

func (a *AdaptadorPaginas) registrarComandosFormularios(mux *nethttp.ServeMux) {
	mux.HandleFunc("POST /entrar", a.processarLogin)
	mux.HandleFunc("POST /cadastro", a.processarCadastroUsuario)
	mux.HandleFunc("POST /sair", a.exigirSessao(a.processarEncerramentoSessao))
	mux.HandleFunc("POST /perfil", a.exigirSessao(a.processarAtualizacaoPerfil))
	mux.HandleFunc("POST /perfil/enderecos", a.exigirSessao(a.processarInclusaoEndereco))
	mux.HandleFunc("POST /perfil/enderecos/{idEndereco}", a.exigirSessao(a.processarAtualizacaoEndereco))
	mux.HandleFunc("POST /perfil/enderecos/{idEndereco}/principal", a.exigirSessao(a.processarEnderecoPrincipal))
	mux.HandleFunc("POST /perfil/enderecos/{idEndereco}/remover", a.exigirSessao(a.processarRemocaoEndereco))
	mux.HandleFunc("POST /carrinho/itens", a.exigirSessao(a.processarInclusaoCarrinho))
	mux.HandleFunc("POST /carrinho/itens/{idAnuncio}/remover", a.exigirSessao(a.processarRemocaoCarrinho))
	mux.HandleFunc("POST /checkout", a.exigirSessao(a.processarCheckout))
	mux.HandleFunc("POST /checkout/cartao", a.exigirSessao(a.processarCheckoutCartao))
	mux.HandleFunc("POST /checkout/pagamento/cancelar", a.exigirSessao(a.processarCancelamentoPagamento))
	mux.HandleFunc("POST /notificacoes/lidas", a.exigirSessao(a.processarLeituraNotificacoes))
	mux.HandleFunc("POST /notificacoes/limpar", a.exigirSessao(a.processarLimpezaNotificacoes))
	mux.HandleFunc("POST /notificacoes/{idNotificacao}/remover", a.exigirSessao(a.processarRemocaoNotificacao))
	mux.HandleFunc("POST /pedidos/{idPedido}/mensagens", a.exigirSessao(a.processarMensagemConversa))
	mux.HandleFunc("POST /minhas-vendas/reativacao", a.exigirSessao(a.processarReativacaoVendedor))
	mux.HandleFunc("POST /minhas-vendas/{idPedido}/envio", a.exigirSessao(a.processarEnvioPedido))
	mux.HandleFunc("POST /meus-pedidos/{idPedido}/recebimento", a.exigirSessao(a.processarRecebimentoPedido))
	mux.HandleFunc("POST /meus-pedidos/{idPedido}/avaliacao", a.exigirSessao(a.processarAvaliacaoPedido))
	mux.HandleFunc("POST /meus-anuncios/{idAnuncio}/excluir", a.exigirSessao(a.processarExclusaoAnuncio))
	mux.HandleFunc("POST /anuncios", a.exigirSessao(a.processarCriacaoAnuncio))
	mux.HandleFunc("POST /meus-anuncios/{idAnuncio}", a.exigirSessao(a.processarAtualizacaoAnuncio))
}
