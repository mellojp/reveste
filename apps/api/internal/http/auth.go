package http

import (
	nethttp "net/http"
	"strings"
	"time"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/transporte"
)

type manipuladorAutenticado func(nethttp.ResponseWriter, *nethttp.Request, string, string)

func (a *API) autenticado(proximo manipuladorAutenticado) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		token, porCookie := tokenDaRequisicao(r)
		idUsuario, err := a.cadastros.IdentificarUsuario(r.Context(), token)
		if err != nil {
			if porCookie {
				a.removerCookieSessao(w, r)
			}
			a.escreverErro(w, common.ErrNaoAutorizado)
			return
		}
		proximo(w, r, idUsuario, token)
	}
}

func tokenDaRequisicao(r *nethttp.Request) (string, bool) {
	if token := extrairToken(r.Header.Get("Authorization")); token != "" {
		return token, false
	}
	if token := transporte.TokenSessaoDoCookie(r); token != "" {
		return token, true
	}
	return "", false
}

func extrairToken(cabecalho string) string {
	partes := strings.SplitN(cabecalho, " ", 2)
	if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(partes[1])
}

func (a *API) definirCookieSessao(w nethttp.ResponseWriter, r *nethttp.Request, token string, expiraEm time.Time) {
	transporte.DefinirCookieSessao(w, r, token, expiraEm, a.confiarProxy)
}

func (a *API) removerCookieSessao(w nethttp.ResponseWriter, r *nethttp.Request) {
	transporte.RemoverCookieSessao(w, r, a.confiarProxy)
}

func (a *API) requisicaoHTTPS(r *nethttp.Request) bool {
	return transporte.HTTPS(r, a.confiarProxy)
}

func (a *API) comProtecaoCSRF(proximo nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if metodoSeguro(r.Method) {
			proximo.ServeHTTP(w, r)
			return
		}
		loginDoNavegador := r.Method == nethttp.MethodPost &&
			(r.URL.Path == "/v1/sessoes" || r.URL.Path == "/entrar") &&
			!strings.EqualFold(r.Header.Get("X-Reveste-Session-Transport"), "bearer")
		formularioWeb := !strings.HasPrefix(r.URL.Path, "/v1/") &&
			!strings.HasPrefix(r.URL.Path, "/saude")
		_, autenticacaoPorCookie := tokenDaRequisicao(r)
		if !autenticacaoPorCookie && !loginDoNavegador && !formularioWeb {
			proximo.ServeHTTP(w, r)
			return
		}
		if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") ||
			!transporte.OrigemPermitida(r, a.confiarProxy) {
			escreverJSON(w, nethttp.StatusForbidden, erroResposta{
				Codigo:   "ORIGEM_NAO_PERMITIDA",
				Mensagem: "A origem da requisicao nao foi permitida.",
				Campos:   map[string]string{},
			})
			return
		}
		proximo.ServeHTTP(w, r)
	})
}

func metodoSeguro(metodo string) bool {
	return metodo == nethttp.MethodGet || metodo == nethttp.MethodHead ||
		metodo == nethttp.MethodOptions || metodo == nethttp.MethodTrace
}
