package http

import nethttp "net/http"

func (a *API) registrarRotasSaude(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /saude", a.saude)
}

func (a *API) saude(w nethttp.ResponseWriter, _ *nethttp.Request) {
	escreverJSON(w, nethttp.StatusOK, map[string]string{"status": "ok"})
}
