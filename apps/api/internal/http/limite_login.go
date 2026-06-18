package http

import (
	nethttp "net/http"

	"reveste/apps/api/internal/transporte"
)

func (a *API) loginPermitido(r *nethttp.Request) bool {
	return a.limitador.Permitido(r.Context(), transporte.EnderecoCliente(r, a.confiarProxy))
}

func (a *API) registrarFalhaLogin(r *nethttp.Request) {
	a.limitador.RegistrarFalha(r.Context(), transporte.EnderecoCliente(r, a.confiarProxy))
}

func (a *API) limparFalhasLogin(r *nethttp.Request) {
	a.limitador.Limpar(r.Context(), transporte.EnderecoCliente(r, a.confiarProxy))
}
