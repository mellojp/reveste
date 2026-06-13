package http

import (
	"context"
	"log/slog"
	nethttp "net/http"
	"sync"

	"reveste/apps/api/internal/casosdeuso"
)

type verificadorProntidao interface {
	Ping(context.Context) error
}

type API struct {
	cadastros *casosdeuso.ControladorCadastro
	anuncios  *casosdeuso.ControladorAnuncio
	compras   *casosdeuso.ControladorCarrinho
	uploads   *casosdeuso.ControladorUpload
	prontidao verificadorProntidao
	logger    *slog.Logger
	loginMu   sync.Mutex
	logins    map[string]tentativasLogin
}

func NovaAPI(
	cadastros *casosdeuso.ControladorCadastro,
	anuncios *casosdeuso.ControladorAnuncio,
	compras *casosdeuso.ControladorCarrinho,
	uploads *casosdeuso.ControladorUpload,
	prontidao verificadorProntidao,
	logger *slog.Logger,
) nethttp.Handler {
	api := &API{
		cadastros: cadastros,
		anuncios:  anuncios,
		compras:   compras,
		uploads:   uploads,
		prontidao: prontidao,
		logger:    logger,
		logins:    make(map[string]tentativasLogin),
	}
	mux := nethttp.NewServeMux()

	api.registrarRotasSaude(mux)
	api.registrarRotasCadastros(mux)
	api.registrarRotasAnuncios(mux)
	api.registrarRotasCarrinho(mux)
	api.registrarRotasUploads(mux)
	api.registrarRotasFrontend(mux)

	return api.comRecuperacao(api.comSeguranca(api.comJSON(mux)))
}
