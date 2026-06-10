package http

import (
	nethttp "net/http"
	"strings"

	"reveste/apps/api/internal/common"
)

type manipuladorAutenticado func(nethttp.ResponseWriter, *nethttp.Request, string, string)

func (a *API) autenticado(proximo manipuladorAutenticado) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		token := extrairToken(r.Header.Get("Authorization"))
		idUsuario, err := a.cadastros.IdentificarUsuario(r.Context(), token)
		if err != nil {
			a.escreverErro(w, common.ErrNaoAutorizado)
			return
		}
		proximo(w, r, idUsuario, token)
	}
}

func extrairToken(cabecalho string) string {
	partes := strings.SplitN(cabecalho, " ", 2)
	if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(partes[1])
}
