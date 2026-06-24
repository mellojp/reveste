package http

import nethttp "net/http"

func (a *API) registrarRotasCarrinho(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /v1/carrinho", a.autenticado(a.obterCarrinho))
	mux.HandleFunc("POST /v1/carrinho/itens", a.autenticado(a.adicionarAoCarrinho))
	mux.HandleFunc("DELETE /v1/carrinho/itens/{idAnuncio}", a.autenticado(a.removerDoCarrinho))
}

func (a *API) obterCarrinho(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	carrinho, err := a.compras.ObterCarrinho(r.Context(), idUsuario)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, carrinho)
}

func (a *API) adicionarAoCarrinho(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	var entrada struct {
		IDAnuncio string `json:"id_anuncio"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	carrinho, err := a.compras.AdicionarAoCarrinho(r.Context(), idUsuario, entrada.IDAnuncio)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, carrinho)
}

func (a *API) removerDoCarrinho(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	carrinho, err := a.compras.RemoverDoCarrinho(r.Context(), idUsuario, r.PathValue("idAnuncio"))
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, carrinho)
}
