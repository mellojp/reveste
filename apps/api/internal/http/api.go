package http

import (
	"log/slog"
	nethttp "net/http"

	casosdeusoanuncios "reveste/apps/api/internal/casosdeuso/anuncios"
	casosdeusocadastros "reveste/apps/api/internal/casosdeuso/cadastros"
	casosdeusocompras "reveste/apps/api/internal/casosdeuso/compras"
)

type API struct {
	cadastros *casosdeusocadastros.FluxoCadastro
	anuncios  *casosdeusoanuncios.FluxoAnuncio
	compras   *casosdeusocompras.FluxoCarrinho
	logger    *slog.Logger
}

func NovaAPI(
	cadastros *casosdeusocadastros.FluxoCadastro,
	anuncios *casosdeusoanuncios.FluxoAnuncio,
	compras *casosdeusocompras.FluxoCarrinho,
	logger *slog.Logger,
) nethttp.Handler {
	api := &API{
		cadastros: cadastros,
		anuncios:  anuncios,
		compras:   compras,
		logger:    logger,
	}
	mux := nethttp.NewServeMux()

	api.registrarRotasSaude(mux)
	api.registrarRotasCadastros(mux)
	api.registrarRotasAnuncios(mux)
	api.registrarRotasCarrinho(mux)

	return api.comRecuperacao(api.comJSON(mux))
}
