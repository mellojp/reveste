package http

import (
	nethttp "net/http"

	"reveste/apps/back/internal/casosdeuso"
)

func (a *API) registrarRotasPedidos(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /v1/me/vendas", a.autenticado(a.listarVendas))
	mux.HandleFunc("POST /v1/me/vendas/{idPedido}/envio", a.autenticado(a.marcarPedidoEnviado))
	mux.HandleFunc("POST /v1/me/pedidos/{idPedido}/recebimento", a.autenticado(a.confirmarRecebimento))
	mux.HandleFunc("POST /v1/me/pedidos/{idPedido}/avaliacao", a.autenticado(a.avaliarPedido))
}

func (a *API) listarVendas(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	vendas, err := a.pedidos.ListarVendas(r.Context(), idUsuario)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, map[string]any{"pedidos": vendas})
}

func (a *API) marcarPedidoEnviado(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	var entrada struct {
		Provedor       string `json:"provedor"`
		CodigoRastreio string `json:"codigo_rastreio"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	err := a.pedidos.MarcarEnviado(r.Context(), idUsuario, r.PathValue("idPedido"), casosdeuso.EntradaEnvio{
		Provedor:       entrada.Provedor,
		CodigoRastreio: entrada.CodigoRastreio,
	})
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (a *API) confirmarRecebimento(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	if err := a.pedidos.ConfirmarRecebimento(r.Context(), idUsuario, r.PathValue("idPedido")); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (a *API) avaliarPedido(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	var entrada struct {
		Nota       int    `json:"nota"`
		Comentario string `json:"comentario"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	if err := a.pedidos.Avaliar(r.Context(), idUsuario, r.PathValue("idPedido"), entrada.Nota, entrada.Comentario); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}
