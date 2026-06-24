package http

import (
	"errors"
	nethttp "net/http"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/common"
)

// WebhookPagamento interpreta uma notificacao de webhook de um provedor de pagamento.
// Valida a autenticidade da requisicao (assinatura) e traduz o evento no desfecho da
// cobranca: a chave de idempotencia da compra, o provedor, o identificador externo e o
// status atual. Implementado pelo adaptador do provedor (ex.: Mercado Pago).
type WebhookPagamento interface {
	Interpretar(r *nethttp.Request) (chave, provedor, idExterno string, status casosdeuso.StatusCobranca, err error)
}

func (a *API) registrarRotasWebhooks(mux *nethttp.ServeMux) {
	// A rota so existe quando ha um provedor real configurado; com o provedor simulado
	// (sincrono) nao ha webhook a receber.
	if a.webhookPagamento == nil {
		return
	}
	mux.HandleFunc("POST /webhooks/pagamento", a.receberWebhookPagamento)
}

// receberWebhookPagamento e o ponto de entrada do webhook do provedor financeiro. Nao usa
// sessao nem protecao CSRF (e uma chamada servidor-a-servidor): a autenticidade vem da
// verificacao de assinatura dentro de Interpretar. Responde 2xx para desfechos conhecidos
// (para o provedor parar de reenviar) e 5xx apenas em falha transitoria, que pede reentrega.
func (a *API) receberWebhookPagamento(w nethttp.ResponseWriter, r *nethttp.Request) {
	chave, provedor, idExterno, status, err := a.webhookPagamento.Interpretar(r)
	if err != nil {
		a.logger.Warn("webhook de pagamento rejeitado", "erro", err)
		w.WriteHeader(nethttp.StatusBadRequest)
		return
	}

	var confirmacaoErr error
	switch status {
	case casosdeuso.CobrancaAprovada:
		_, confirmacaoErr = a.checkout.ConfirmarPagamentoExterno(r.Context(), chave, provedor, idExterno, true)
	case casosdeuso.CobrancaRecusada:
		_, confirmacaoErr = a.checkout.ConfirmarPagamentoExterno(r.Context(), chave, provedor, idExterno, false)
	default:
		// Pendente ou estado sem acao: apenas confirma o recebimento do evento.
		w.WriteHeader(nethttp.StatusOK)
		return
	}

	switch {
	case confirmacaoErr == nil,
		errors.Is(confirmacaoErr, common.ErrPagamentoRecusado),
		errors.Is(confirmacaoErr, common.ErrTransicaoInvalida),
		errors.Is(confirmacaoErr, common.ErrNaoEncontrado):
		// Desfecho conhecido (inclusive recusa aplicada, reentrega idempotente ou intencao
		// inexistente): nao adianta o provedor reenviar.
		w.WriteHeader(nethttp.StatusOK)
	default:
		a.logger.Error("falha ao confirmar pagamento por webhook", "erro", confirmacaoErr)
		w.WriteHeader(nethttp.StatusInternalServerError)
	}
}
