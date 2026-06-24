package http

import (
	"context"
	"log/slog"
	nethttp "net/http"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/transporte"
)

type verificadorProntidao interface {
	Ping(context.Context) error
}

type API struct {
	cadastros        *casosdeuso.ControladorCadastro
	anuncios         *casosdeuso.ControladorAnuncio
	compras          *casosdeuso.ControladorCarrinho
	checkout         *casosdeuso.ControladorCheckout
	pedidos          *casosdeuso.ControladorPedidos
	vendedores       *casosdeuso.ControladorVendedor
	notificacoes     *casosdeuso.ControladorNotificacoes
	conversas        *casosdeuso.ControladorConversas
	uploads          *casosdeuso.ControladorUpload
	cep              *casosdeuso.ControladorCEP
	prontidao        verificadorProntidao
	logger           *slog.Logger
	hostBlob         string
	limitador        *transporte.LimitadorLogin
	confiarProxy     bool
	webhookPagamento WebhookPagamento
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
	cep *casosdeuso.ControladorCEP,
	prontidao verificadorProntidao,
	logger *slog.Logger,
	hostBlob string,
	limitador *transporte.LimitadorLogin,
	confiarProxy bool,
	webhookPagamento WebhookPagamento,
	paginasHTML nethttp.Handler,
) nethttp.Handler {
	api := &API{
		cadastros:        cadastros,
		anuncios:         anuncios,
		compras:          compras,
		checkout:         checkout,
		pedidos:          pedidos,
		vendedores:       vendedores,
		notificacoes:     notificacoes,
		conversas:        conversas,
		uploads:          uploads,
		cep:              cep,
		prontidao:        prontidao,
		logger:           logger,
		hostBlob:         hostBlob,
		limitador:        limitador,
		confiarProxy:     confiarProxy,
		webhookPagamento: webhookPagamento,
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
	api.registrarRotasCEP(mux)
	api.registrarRotasWebhooks(mux)
	api.registrarRotasFrontend(mux, paginasHTML)

	return api.comRecuperacao(api.comSeguranca(api.comJSON(api.comProtecaoCSRF(mux))))
}
