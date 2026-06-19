package http

import (
	nethttp "net/http"
)

func (a *API) registrarRotasNotificacoes(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /v1/me/notificacoes", a.autenticado(a.listarNotificacoes))
	mux.HandleFunc("POST /v1/me/notificacoes/lidas", a.autenticado(a.marcarNotificacoesLidas))
	mux.HandleFunc("DELETE /v1/me/notificacoes/{idNotificacao}", a.autenticado(a.removerNotificacao))
	mux.HandleFunc("DELETE /v1/me/notificacoes", a.autenticado(a.limparNotificacoes))
}

func (a *API) removerNotificacao(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	if err := a.notificacoes.Remover(r.Context(), idUsuario, r.PathValue("idNotificacao")); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (a *API) limparNotificacoes(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	if err := a.notificacoes.Limpar(r.Context(), idUsuario); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (a *API) listarNotificacoes(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	notificacoes, err := a.notificacoes.Listar(r.Context(), idUsuario)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	naoLidas, err := a.notificacoes.ContarNaoLidas(r.Context(), idUsuario)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, map[string]any{
		"notificacoes": notificacoes,
		"nao_lidas":    naoLidas,
	})
}

func (a *API) marcarNotificacoesLidas(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	if err := a.notificacoes.MarcarTodasLidas(r.Context(), idUsuario); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}
