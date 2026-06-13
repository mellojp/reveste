package http

import (
	"net"
	nethttp "net/http"
	"time"
)

const (
	maxTentativasLogin = 5
	janelaLogin        = time.Minute
)

type tentativasLogin struct {
	inicio     time.Time
	quantidade int
}

func (a *API) loginPermitido(r *nethttp.Request) bool {
	chave := enderecoRemoto(r)
	agora := time.Now()

	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	tentativas, existe := a.logins[chave]
	if !existe || agora.Sub(tentativas.inicio) >= janelaLogin {
		delete(a.logins, chave)
		return true
	}
	return tentativas.quantidade < maxTentativasLogin
}

func (a *API) registrarFalhaLogin(r *nethttp.Request) {
	chave := enderecoRemoto(r)
	agora := time.Now()

	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	if len(a.logins) >= 1_000 {
		for chave, tentativa := range a.logins {
			if agora.Sub(tentativa.inicio) >= janelaLogin {
				delete(a.logins, chave)
			}
		}
	}
	tentativas, existe := a.logins[chave]
	if !existe || agora.Sub(tentativas.inicio) >= janelaLogin {
		tentativas = tentativasLogin{inicio: agora}
	}
	tentativas.quantidade++
	a.logins[chave] = tentativas
}

func (a *API) limparFalhasLogin(r *nethttp.Request) {
	a.loginMu.Lock()
	delete(a.logins, enderecoRemoto(r))
	a.loginMu.Unlock()
}

func enderecoRemoto(r *nethttp.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
