// Package transporte reune utilitarios de borda HTTP compartilhados entre o adaptador
// JSON (internal/http) e o adaptador de paginas SSR (internal/web): cookie de sessao,
// deteccao de HTTPS, IP do cliente, verificacao de origem (CSRF) e limite de login.
package transporte

import (
	"net"
	nethttp "net/http"
	"net/url"
	"strings"
	"time"
)

// NomeCookieSessao e o nome do cookie HttpOnly que carrega o token de sessao.
const NomeCookieSessao = "reveste_session"

// HTTPS indica se a requisicao chegou por TLS. O cabecalho X-Forwarded-Proto so e
// considerado quando confiarProxy e verdadeiro (atras de um proxy reverso conhecido),
// pois um cliente direto poderia forja-lo.
func HTTPS(r *nethttp.Request, confiarProxy bool) bool {
	if r.TLS != nil {
		return true
	}
	return confiarProxy && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// EnderecoCliente devolve o IP do cliente para fins de limite de tentativas. Atras de um
// proxy confiavel usa o primeiro IP de X-Forwarded-For; caso contrario usa RemoteAddr,
// pois o cabecalho seria spoofavel sem um proxy a frente.
func EnderecoCliente(r *nethttp.Request, confiarProxy bool) string {
	if confiarProxy {
		if encaminhado := r.Header.Get("X-Forwarded-For"); encaminhado != "" {
			if primeiro := strings.TrimSpace(strings.Split(encaminhado, ",")[0]); primeiro != "" {
				return primeiro
			}
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// DefinirCookieSessao grava o cookie de sessao HttpOnly/SameSite=Lax (Secure em HTTPS).
func DefinirCookieSessao(w nethttp.ResponseWriter, r *nethttp.Request, token string, expiraEm time.Time, confiarProxy bool) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name: NomeCookieSessao, Value: token, Path: "/", HttpOnly: true,
		Secure: HTTPS(r, confiarProxy), SameSite: nethttp.SameSiteLaxMode,
		Expires: expiraEm, MaxAge: max(1, int(time.Until(expiraEm).Seconds())),
	})
}

// RemoverCookieSessao expira o cookie de sessao.
func RemoverCookieSessao(w nethttp.ResponseWriter, r *nethttp.Request, confiarProxy bool) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name: NomeCookieSessao, Value: "", Path: "/", HttpOnly: true,
		Secure: HTTPS(r, confiarProxy), SameSite: nethttp.SameSiteLaxMode,
		Expires: time.Unix(1, 0), MaxAge: -1,
	})
}

// TokenSessaoDoCookie devolve o token de sessao presente no cookie, se houver.
func TokenSessaoDoCookie(r *nethttp.Request) string {
	cookie, err := r.Cookie(NomeCookieSessao)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

// OrigemPermitida verifica se o cabecalho Origin corresponde a propria aplicacao,
// mitigando CSRF em requisicoes mutaveis autenticadas por cookie.
func OrigemPermitida(r *nethttp.Request, confiarProxy bool) bool {
	origem := strings.TrimSpace(r.Header.Get("Origin"))
	if origem == "" {
		return false
	}
	endereco, err := url.Parse(origem)
	esquemaEsperado := "http"
	if HTTPS(r, confiarProxy) {
		esquemaEsperado = "https"
	}
	return err == nil && endereco.Host == r.Host && endereco.Scheme == esquemaEsperado
}
