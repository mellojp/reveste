package http

import (
	nethttp "net/http"
)

func (a *API) registrarRotasVendedores(mux *nethttp.ServeMux) {
	mux.HandleFunc("POST /v1/me/reativacao", a.autenticado(a.reativarVendedor))
}

func (a *API) reativarVendedor(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	if err := a.vendedores.Reativar(r.Context(), idUsuario); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}
