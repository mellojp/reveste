package http

import nethttp "net/http"

func (a *API) registrarRotasCheckout(mux *nethttp.ServeMux) {
	mux.HandleFunc("POST /v1/checkout", a.autenticado(a.finalizarCompra))
	mux.HandleFunc("GET /v1/me/pedidos", a.autenticado(a.listarPedidos))
}

func (a *API) finalizarCompra(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	var entrada struct {
		IDEndereco string `json:"id_endereco"`
	}
	// O corpo e opcional: sem id_endereco, usa-se o endereco principal do comprador.
	if r.ContentLength != 0 {
		if !decodificarJSON(w, r, &entrada) {
			return
		}
	}
	compra, err := a.checkout.FinalizarCompra(r.Context(), idUsuario, entrada.IDEndereco)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusCreated, compra)
}

func (a *API) listarPedidos(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	pedidos, err := a.checkout.ListarPedidos(r.Context(), idUsuario)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, map[string]any{"pedidos": pedidos})
}
