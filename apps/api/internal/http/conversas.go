package http

import (
	nethttp "net/http"
)

func (a *API) registrarRotasConversas(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /v1/me/pedidos/{idPedido}/conversa", a.autenticado(a.abrirConversa))
	mux.HandleFunc("POST /v1/me/pedidos/{idPedido}/mensagens", a.autenticado(a.enviarMensagem))
}

func (a *API) abrirConversa(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	conversa, err := a.conversas.Abrir(r.Context(), idUsuario, r.PathValue("idPedido"))
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, conversa)
}

func (a *API) enviarMensagem(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	var entrada struct {
		Conteudo string `json:"conteudo"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	if err := a.conversas.Enviar(r.Context(), idUsuario, r.PathValue("idPedido"), entrada.Conteudo); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}
