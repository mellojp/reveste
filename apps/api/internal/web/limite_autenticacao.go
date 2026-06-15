package web

import (
	"net"
	nethttp "net/http"
	"time"
)

const (
	maxTentativasLogin = 5
	janelaLogin        = time.Minute
)

type registroTentativasLogin struct {
	inicio     time.Time
	quantidade int
}

func (a *AdaptadorPaginas) autenticacaoPermitida(r *nethttp.Request) bool {
	chave := enderecoRemoto(r)
	agora := time.Now()
	a.tentativasMu.Lock()
	defer a.tentativasMu.Unlock()
	tentativas, existe := a.tentativasLogin[chave]
	if !existe || agora.Sub(tentativas.inicio) >= janelaLogin {
		delete(a.tentativasLogin, chave)
		return true
	}
	return tentativas.quantidade < maxTentativasLogin
}

func (a *AdaptadorPaginas) registrarFalhaAutenticacao(r *nethttp.Request) {
	chave := enderecoRemoto(r)
	agora := time.Now()
	a.tentativasMu.Lock()
	defer a.tentativasMu.Unlock()
	if len(a.tentativasLogin) >= 1_000 {
		for chave, tentativa := range a.tentativasLogin {
			if agora.Sub(tentativa.inicio) >= janelaLogin {
				delete(a.tentativasLogin, chave)
			}
		}
	}
	tentativas, existe := a.tentativasLogin[chave]
	if !existe || agora.Sub(tentativas.inicio) >= janelaLogin {
		tentativas = registroTentativasLogin{inicio: agora}
	}
	tentativas.quantidade++
	a.tentativasLogin[chave] = tentativas
}

func (a *AdaptadorPaginas) limparFalhasAutenticacao(r *nethttp.Request) {
	a.tentativasMu.Lock()
	delete(a.tentativasLogin, enderecoRemoto(r))
	a.tentativasMu.Unlock()
}

func enderecoRemoto(r *nethttp.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
