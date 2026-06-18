// Package pagamentos contem adaptadores para provedores financeiros.
package pagamentos

import (
	"context"

	"reveste/apps/api/internal/casosdeuso"
)

const provedorSimulado = "simulado"

// Simulado e um ProcessadorPagamento que aprova as cobrancas de forma deterministica.
// Serve ao MVP enquanto um gateway real (Stripe, Mercado Pago) nao e integrado: a porta
// ProcessadorPagamento permite substitui-lo sem alterar o caso de uso.
type Simulado struct {
	// Recusar inverte a decisao para exercitar o caminho de pagamento recusado.
	Recusar bool
}

// NovoSimulado cria um provedor que aprova todas as cobrancas.
func NovoSimulado() *Simulado {
	return &Simulado{}
}

func (s *Simulado) Processar(
	_ context.Context,
	solicitacao casosdeuso.SolicitacaoPagamento,
) (casosdeuso.ResultadoPagamento, error) {
	return casosdeuso.ResultadoPagamento{
		Aprovado:             !s.Recusar,
		Provedor:             provedorSimulado,
		IdentificadorExterno: provedorSimulado + "_" + solicitacao.ChaveIdempotencia,
	}, nil
}
