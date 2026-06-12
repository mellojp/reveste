package http

import (
	nethttp "net/http"
	"os"
)

func (a *API) registrarRotasFrontend(mux *nethttp.ServeMux) {
	for _, diretorio := range []string{"apps/front", "../../../front"} {
		if _, err := os.Stat(diretorio + "/index.html"); err == nil {
			mux.Handle("GET /", nethttp.FileServerFS(os.DirFS(diretorio)))
			return
		}
	}

	a.logger.Warn("frontend nao encontrado")
}
