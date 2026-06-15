package web

import nethttp "net/http"

func (a *AdaptadorPaginas) responderDocumentoHTML(
	w nethttp.ResponseWriter,
	status int,
	contexto contextoDocumento,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := a.documentosHTML.ExecuteTemplate(w, "documento", contexto); err != nil {
		a.logger.Error("renderizar documento HTML", "conteudo", contexto.Conteudo, "erro", err)
	}
}

func (a *AdaptadorPaginas) responderFragmentoHTML(
	w nethttp.ResponseWriter,
	nome string,
	contexto contextoDocumento,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.documentosHTML.ExecuteTemplate(w, nome, contexto); err != nil {
		a.logger.Error("renderizar fragmento HTML", "fragmento", nome, "erro", err)
	}
}

func (a *AdaptadorPaginas) responderRedirecionamento(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	destino string,
) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", destino)
		w.WriteHeader(nethttp.StatusNoContent)
		return
	}
	nethttp.Redirect(w, r, destino, nethttp.StatusSeeOther)
}
