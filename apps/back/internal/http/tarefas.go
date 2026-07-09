package http

import (
	"crypto/subtle"
	nethttp "net/http"
	"strings"
)

func (a *API) registrarRotasTarefas(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /tarefas/processar", a.processarTarefasAgendadas)
}

func (a *API) processarTarefasAgendadas(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !a.autorizarTarefaAgendada(r) {
		escreverJSON(w, nethttp.StatusUnauthorized, erroResposta{
			Codigo: "NAO_AUTORIZADO", Mensagem: "Autenticacao obrigatoria ou invalida.", Campos: map[string]string{},
		})
		return
	}

	expiradas, err := a.checkout.ProcessarExpiracoes(r.Context())
	if err != nil {
		a.logger.Error("falha ao expirar compras pendentes", "erro", err)
		a.escreverErro(w, err)
		return
	}

	naoEnviados, err := a.pedidos.ProcessarPrazosEnvio(r.Context())
	if err != nil {
		a.logger.Error("falha ao processar prazos de envio", "erro", err)
		a.escreverErro(w, err)
		return
	}

	escreverJSON(w, nethttp.StatusOK, map[string]int{
		"compras_expiradas":  expiradas,
		"itens_nao_enviados": naoEnviados,
	})
}

func (a *API) autorizarTarefaAgendada(r *nethttp.Request) bool {
	cabecalho := strings.TrimSpace(r.Header.Get("Authorization"))
	token, ok := strings.CutPrefix(cabecalho, "Bearer ")
	if ok && a.cronSecret != "" {
		return subtle.ConstantTimeCompare([]byte(token), []byte(a.cronSecret)) == 1
	}

	return strings.EqualFold(strings.TrimSpace(r.UserAgent()), "vercel-cron/1.0") &&
		strings.TrimSpace(r.Header.Get("x-vercel-cron-schedule")) != ""
}
