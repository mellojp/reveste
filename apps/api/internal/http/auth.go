package http

import (
	nethttp "net/http"
	"net/url"
	"strings"
	"time"

	"reveste/apps/api/internal/common"
)

const nomeCookieSessao = "reveste_session"

type manipuladorAutenticado func(nethttp.ResponseWriter, *nethttp.Request, string, string)

func (a *API) autenticado(proximo manipuladorAutenticado) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		token, porCookie := tokenDaRequisicao(r)
		idUsuario, err := a.cadastros.IdentificarUsuario(r.Context(), token)
		if err != nil {
			if porCookie {
				removerCookieSessao(w, r)
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
	cookie, err := r.Cookie(nomeCookieSessao)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(cookie.Value), true
}

func extrairToken(cabecalho string) string {
	partes := strings.SplitN(cabecalho, " ", 2)
	if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(partes[1])
}

func definirCookieSessao(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	token string,
	expiraEm time.Time,
) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name: nomeCookieSessao, Value: token, Path: "/", HttpOnly: true,
		Secure: requisicaoHTTPS(r), SameSite: nethttp.SameSiteLaxMode,
		Expires: expiraEm, MaxAge: max(1, int(time.Until(expiraEm).Seconds())),
	})
}

func removerCookieSessao(w nethttp.ResponseWriter, r *nethttp.Request) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name: nomeCookieSessao, Value: "", Path: "/", HttpOnly: true,
		Secure: requisicaoHTTPS(r), SameSite: nethttp.SameSiteLaxMode,
		Expires: time.Unix(1, 0), MaxAge: -1,
	})
}

func requisicaoHTTPS(r *nethttp.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
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
			!origemPermitida(r) {
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

func origemPermitida(r *nethttp.Request) bool {
	origem := strings.TrimSpace(r.Header.Get("Origin"))
	if origem == "" {
		return false
	}
	endereco, err := url.Parse(origem)
	esquemaEsperado := "http"
	if requisicaoHTTPS(r) {
		esquemaEsperado = "https"
	}
	return err == nil && endereco.Host == r.Host && endereco.Scheme == esquemaEsperado
}
