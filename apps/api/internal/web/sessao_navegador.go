package web

import (
	nethttp "net/http"
	"net/url"
	"strings"
	"time"

	"reveste/apps/api/internal/casosdeuso"
)

const nomeCookieSessao = "reveste_session"

type sessaoNavegador struct {
	IDUsuario string
	Token     string
}

type manipuladorComSessao func(nethttp.ResponseWriter, *nethttp.Request, sessaoNavegador)

func (a *AdaptadorPaginas) exigirSessao(proximo manipuladorComSessao) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		token := tokenSessaoDoCookie(r)
		idUsuario, err := a.controladorCadastro.IdentificarUsuario(r.Context(), token)
		if err != nil {
			removerCookieSessao(w, r)
			a.responderRedirecionamento(w, r, "/entrar?retorno="+url.QueryEscape(retornoRequisicao(r)))
			return
		}
		proximo(w, r, sessaoNavegador{IDUsuario: idUsuario, Token: token})
	}
}

func retornoRequisicao(r *nethttp.Request) string {
	if r.Method == nethttp.MethodGet {
		return normalizarRotaRetorno(r.URL.RequestURI())
	}
	referencia, err := url.Parse(r.Referer())
	if err == nil && referencia.Host == r.Host {
		return normalizarRotaRetorno(referencia.RequestURI())
	}
	return "/catalogo"
}

func tokenSessaoDoCookie(r *nethttp.Request) string {
	cookie, err := r.Cookie(nomeCookieSessao)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func definirCookieSessao(w nethttp.ResponseWriter, r *nethttp.Request, sessao casosdeuso.Sessao) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name: nomeCookieSessao, Value: sessao.Token, Path: "/", HttpOnly: true,
		Secure: requisicaoHTTPS(r), SameSite: nethttp.SameSiteLaxMode,
		Expires: sessao.ExpiraEm, MaxAge: max(1, int(time.Until(sessao.ExpiraEm).Seconds())),
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
