package web

import (
	nethttp "net/http"
	"net/url"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/transporte"
)

type sessaoNavegador struct {
	IDUsuario string
	Token     string
}

type manipuladorComSessao func(nethttp.ResponseWriter, *nethttp.Request, sessaoNavegador)

func (a *AdaptadorPaginas) exigirSessao(proximo manipuladorComSessao) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		token := transporte.TokenSessaoDoCookie(r)
		idUsuario, err := a.controladorCadastro.IdentificarUsuario(r.Context(), token)
		if err != nil {
			a.removerCookieSessao(w, r)
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

func (a *AdaptadorPaginas) definirCookieSessao(w nethttp.ResponseWriter, r *nethttp.Request, sessao casosdeuso.Sessao) {
	transporte.DefinirCookieSessao(w, r, sessao.Token, sessao.ExpiraEm, a.confiarProxy)
}

func (a *AdaptadorPaginas) removerCookieSessao(w nethttp.ResponseWriter, r *nethttp.Request) {
	transporte.RemoverCookieSessao(w, r, a.confiarProxy)
}
