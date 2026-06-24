package compras

import (
	"errors"
	"testing"

	"reveste/apps/back/internal/common"
)

func TestPedidoRejeitaTransicaoInvalida(t *testing.T) {
	pedido := Pedido{Status: StatusPedidoCriado}

	err := pedido.AlterarStatus(StatusPedidoFinalizado)

	if !errors.Is(err, common.ErrTransicaoInvalida) {
		t.Fatalf("erro = %v; esperado ErrTransicaoInvalida", err)
	}
	if pedido.Status != StatusPedidoCriado {
		t.Fatalf("status foi alterado para %q", pedido.Status)
	}
}

func TestPedidoAceitaFluxoPrincipal(t *testing.T) {
	pedido := Pedido{Status: StatusPedidoCriado}
	fluxo := []StatusPedido{
		StatusPedidoAguardandoPagamento,
		StatusPedidoAguardandoEnvio,
		StatusPedidoAguardandoEntrega,
		StatusPedidoFinalizado,
	}

	for _, status := range fluxo {
		if err := pedido.AlterarStatus(status); err != nil {
			t.Fatalf("AlterarStatus(%q) erro = %v", status, err)
		}
	}
}

func TestPagamentoNaoVoltaParaPendente(t *testing.T) {
	pagamento := Pagamento{Status: StatusPagamentoAprovado}

	err := pagamento.AlterarStatus(StatusPagamentoPendente)

	if !errors.Is(err, common.ErrTransicaoInvalida) {
		t.Fatalf("erro = %v; esperado ErrTransicaoInvalida", err)
	}
}
