package http

import nethttp "net/http"

func (a *API) comJSON(proximo nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
