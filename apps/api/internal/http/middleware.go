package http

import (
	nethttp "net/http"
	"strings"
)

func (a *API) comJSON(proximo nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") || strings.HasPrefix(r.URL.Path, "/saude") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		proximo.ServeHTTP(w, r)
	})
}

func (a *API) comSeguranca(proximo nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: blob: https:; connect-src 'self' https://vercel.com; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		proximo.ServeHTTP(w, r)
	})
}

func (a *API) comRecuperacao(proximo nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		defer func() {
			if recuperado := recover(); recuperado != nil {
				a.logger.Error("panico recuperado", "valor", recuperado)
				escreverJSON(w, nethttp.StatusInternalServerError, erroResposta{
					Codigo: "ERRO_INTERNO", Mensagem: "Ocorreu um erro interno.", Campos: map[string]string{},
				})
			}
		}()
		proximo.ServeHTTP(w, r)
	})
}
