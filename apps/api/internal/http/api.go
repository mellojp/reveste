package http

import (
	"context"
	"log/slog"
	nethttp "net/http"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/transporte"
)

type verificadorProntidao interface {
	Ping(context.Context) error
}

type API struct {
	cadastros    *casosdeuso.ControladorCadastro
	anuncios     *casosdeuso.ControladorAnuncio
	compras      *casosdeuso.ControladorCarrinho
	checkout     *casosdeuso.ControladorCheckout
	pedidos      *casosdeuso.ControladorPedidos
	vendedores   *casosdeuso.ControladorVendedor
	notificacoes *casosdeuso.ControladorNotificacoes
	conversas    *casosdeuso.ControladorConversas
	uploads      *casosdeuso.ControladorUpload
	prontidao    verificadorProntidao
	logger       *slog.Logger
	hostBlob     string
	limitador    *transporte.LimitadorLogin
	confiarProxy bool
}

func NovaAPI(
	cadastros *casosdeuso.ControladorCadastro,
	anuncios *casosdeuso.ControladorAnuncio,
	compras *casosdeuso.ControladorCarrinho,
	uploads *casosdeuso.ControladorUpload,
	checkout *casosdeuso.ControladorCheckout,
	pedidos *casosdeuso.ControladorPedidos,
	vendedores *casosdeuso.ControladorVendedor,
	notificacoes *casosdeuso.ControladorNotificacoes,
	conversas *casosdeuso.ControladorConversas,
	prontidao verificadorProntidao,
	logger *slog.Logger,
	hostBlob string,
	limitador *transporte.LimitadorLogin,
	confiarProxy bool,
	paginasHTML nethttp.Handler,
) nethttp.Handler {
	api := &API{
		cadastros:    cadastros,
		anuncios:     anuncios,
		compras:      compras,
		checkout:     checkout,
		pedidos:      pedidos,
		vendedores:   vendedores,
		notificacoes: notificacoes,
		conversas:    conversas,
		uploads:      uploads,
		prontidao:    prontidao,
		logger:       logger,
		hostBlob:     hostBlob,
		limitador:    limitador,
		confiarProxy: confiarProxy,
	}
	mux := nethttp.NewServeMux()

	api.registrarRotasSaude(mux)
	api.registrarRotasCadastros(mux)
	api.registrarRotasAnuncios(mux)
	api.registrarRotasCarrinho(mux)
	api.registrarRotasCheckout(mux)
	api.registrarRotasEnderecos(mux)
	api.registrarRotasPedidos(mux)
	api.registrarRotasVendedores(mux)
	api.registrarRotasNotificacoes(mux)
	api.registrarRotasConversas(mux)
	api.registrarRotasUploads(mux)
	api.registrarRotasFrontend(mux, paginasHTML)

	return api.comRecuperacao(api.comSeguranca(api.comJSON(api.comProtecaoCSRF(mux))))
}
