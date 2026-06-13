package http

import (
	nethttp "net/http"
	"os"
	"path/filepath"
	"strings"
)

func (a *API) registrarRotasFrontend(mux *nethttp.ServeMux) {
	for _, diretorio := range []string{"apps/front", "../../../front"} {
		if _, err := os.Stat(diretorio + "/index.html"); err == nil {
			arquivos := nethttp.FileServer(nethttp.Dir(diretorio))
			mux.HandleFunc("GET /", func(w nethttp.ResponseWriter, r *nethttp.Request) {
				caminhoRelativo := strings.TrimPrefix(filepath.Clean(r.URL.Path), string(filepath.Separator))
				caminhoLocal := filepath.Join(diretorio, filepath.FromSlash(caminhoRelativo))
				if info, err := os.Stat(caminhoLocal); err == nil && !info.IsDir() {
					w.Header().Set("Cache-Control", "public, max-age=3600")
					arquivos.ServeHTTP(w, r)
					return
				}
				w.Header().Set("Cache-Control", "no-cache")
				nethttp.ServeFile(w, r, filepath.Join(diretorio, "index.html"))
			})
			return
		}
	}

	a.logger.Warn("frontend nao encontrado")
}
