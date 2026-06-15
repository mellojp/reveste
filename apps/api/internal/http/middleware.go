package http

import (
	"fmt"
	nethttp "net/http"
	"strings"
)

func (a *API) comJSON(proximo nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") || strings.HasPrefix(r.URL.Path, "/saude") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
		}
		proximo.ServeHTTP(w, r)
	})
}

func (a *API) comSeguranca(proximo nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		origemImagens := "'self' blob:"
		if a.hostBlob != "" {
			origemImagens += fmt.Sprintf(" https://%s", a.hostBlob)
		}
		csp := fmt.Sprintf(
			"default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; img-src %s; connect-src 'self' https://vercel.com; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; media-src 'none'; worker-src 'none'; manifest-src 'none'",
			origemImagens,
		)
		if requisicaoHTTPS(r) {
			csp += "; upgrade-insecure-requests"
		}
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("Origin-Agent-Cluster", "?1")
		w.Header().Set("X-DNS-Prefetch-Control", "off")
		if requisicaoHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
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
