package http

import (
	nethttp "net/http"
	"os"
)

func (a *API) registrarRotasFrontend(mux *nethttp.ServeMux, paginasHTML nethttp.Handler) {
	for _, diretorio := range []string{"apps/front", "../../../front"} {
		if _, err := os.Stat(diretorio + "/styles.css"); err == nil {
			arquivos := nethttp.FileServer(nethttp.Dir(diretorio))
			comCache := func(politica string) nethttp.Handler {
				return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
					w.Header().Set("Cache-Control", politica)
					arquivos.ServeHTTP(w, r)
				})
			}
			// Sem etapa de build, CSS e JS mudam "a quente". no-store garante que o navegador
			// sempre baixe a versao atual, evitando estilos defasados servidos do cache (inclusive
			// arquivos referenciados por @import, que alguns navegadores nao revalidam com no-cache).
			semArmazenamento := comCache("no-store")
			revalidar := comCache("no-cache")
			mux.Handle("GET /styles.css", semArmazenamento)
			mux.Handle("GET /css/", semArmazenamento)
			mux.Handle("GET /js/", semArmazenamento)
			mux.Handle("GET /assets/", revalidar)
			break
		}
	}
	mux.Handle("/", paginasHTML)
}
