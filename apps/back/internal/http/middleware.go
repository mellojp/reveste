package http

import (
	"fmt"
	nethttp "net/http"
	"strings"
)

func (a *API) comJSON(proximo nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") ||
			strings.HasPrefix(r.URL.Path, "/saude") ||
			strings.HasPrefix(r.URL.Path, "/tarefas/") {
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
		origemImagens := "'self' blob: data:"
		if a.hostBlob != "" {
			origemImagens += fmt.Sprintf(" https://%s", a.hostBlob)
		}
		scriptSrc := "'self'"
		styleSrc := "'self'"
		connectSrc := "'self' https://vercel.com"
		frameSrc := ""
		// Apenas na tela de cartao liberamos o SDK e os iframes/recursos do Mercado Pago (Card
		// Payment Brick). O restante do site mantem a CSP estrita.
		if r.URL.Path == "/checkout/cartao" {
			// O Card Payment Brick injeta scripts inline, busca recursos (i18n) do http2.mlstatic.com
			// e roda fingerprint antifraude contra os dominios do Mercado Pago/Mercado Livre
			// (connect + img). Liberamos amplamente esses dominios so nesta rota.
			mp := "https://*.mercadopago.com https://*.mercadolibre.com https://*.mercadolivre.com https://http2.mlstatic.com"
			scriptSrc += " https://sdk.mercadopago.com https://http2.mlstatic.com 'unsafe-inline'"
			styleSrc += " 'unsafe-inline' https://http2.mlstatic.com"
			connectSrc += " " + mp
			frameSrc = "frame-src https://*.mercadopago.com https://*.mercadolibre.com https://*.mercadolivre.com; "
			origemImagens += " " + mp
		}
		csp := fmt.Sprintf(
			"default-src 'none'; script-src %s; style-src %s; font-src 'self'; img-src %s; connect-src %s; %sobject-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; media-src 'none'; worker-src 'self' blob:; manifest-src 'none'",
			scriptSrc, styleSrc, origemImagens, connectSrc, frameSrc,
		)
		if a.requisicaoHTTPS(r) {
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
		if a.requisicaoHTTPS(r) {
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
