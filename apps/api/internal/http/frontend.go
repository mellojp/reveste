package http

import (
	nethttp "net/http"
	"os"
)

func (a *API) registrarRotasFrontend(mux *nethttp.ServeMux, paginasHTML nethttp.Handler) {
	for _, diretorio := range []string{"apps/front", "../../../front"} {
		if _, err := os.Stat(diretorio + "/styles.css"); err == nil {
			arquivos := nethttp.FileServer(nethttp.Dir(diretorio))
			publico := nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				w.Header().Set("Cache-Control", "no-cache")
				arquivos.ServeHTTP(w, r)
			})
			mux.Handle("GET /styles.css", publico)
			mux.Handle("GET /css/", publico)
			mux.Handle("GET /assets/", publico)
			mux.Handle("GET /js/", publico)
			break
		}
	}
	mux.Handle("/", paginasHTML)
}
