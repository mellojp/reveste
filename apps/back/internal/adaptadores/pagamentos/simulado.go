// Package pagamentos contem adaptadores para provedores financeiros.
package pagamentos

import (
	"context"

	"reveste/apps/back/internal/casosdeuso"
)

const provedorSimulado = "simulado"

// Simulado e um ProcessadorPagamento sincrono que aprova as cobrancas de forma
// deterministica. Serve ao MVP enquanto um gateway real (Mercado Pago, Pagar.me, Stripe)
// nao e integrado: a porta ProcessadorPagamento permite substitui-lo sem alterar o caso de
// uso. Por ser sincrono, devolve Aprovada (ou Recusada) na hora, sem instrucoes de pagamento.
type Simulado struct {
	// Recusar inverte a decisao para exercitar o caminho de pagamento recusado.
	Recusar bool
}

// NovoSimulado cria um provedor que aprova todas as cobrancas.
func NovoSimulado() *Simulado {
	return &Simulado{}
}

func (s *Simulado) CriarCobranca(
	_ context.Context,
	solicitacao casosdeuso.SolicitacaoPagamento,
) (casosdeuso.Cobranca, error) {
	status := casosdeuso.CobrancaAprovada
	if s.Recusar {
		status = casosdeuso.CobrancaRecusada
	}
	return casosdeuso.Cobranca{
		Status:               status,
		Provedor:             provedorSimulado,
		IdentificadorExterno: provedorSimulado + "_" + solicitacao.ChaveIdempotencia,
	}, nil
}
