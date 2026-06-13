package http

import (
	"context"
	nethttp "net/http"
	"time"
)

func (a *API) registrarRotasSaude(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /saude", a.saude)
	mux.HandleFunc("GET /saude/prontidao", a.verificarProntidao)
}

func (a *API) saude(w nethttp.ResponseWriter, _ *nethttp.Request) {
	escreverJSON(w, nethttp.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) verificarProntidao(w nethttp.ResponseWriter, r *nethttp.Request) {
	if a.prontidao == nil {
		escreverJSON(w, nethttp.StatusServiceUnavailable, map[string]string{"status": "indisponivel"})
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancelar()
	if err := a.prontidao.Ping(ctx); err != nil {
		a.logger.Error("verificacao de prontidao falhou", "erro", err)
		escreverJSON(w, nethttp.StatusServiceUnavailable, map[string]string{"status": "indisponivel"})
		return
	}
	escreverJSON(w, nethttp.StatusOK, map[string]string{"status": "pronto"})
}
