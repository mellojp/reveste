package casosdeuso_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/cadastros"
)

func novoVendedor(store *Store, pagamento casosdeuso.ProcessadorPagamento) *casosdeuso.ControladorVendedor {
	return casosdeuso.NovoControladorVendedor(
		store, pagamento,
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
		cadastros.TaxaReativacaoCentavos,
	)
}

func TestReativarDesbloqueiaVendedorComPagamentoAprovado(t *testing.T) {
	store := newTestStore()
	store.usuarios["vendedor-1"] = cadastros.Usuario{
		ID: "vendedor-1", BloqueadoParaVendas: true, ItensNaoEnviados: 3,
	}

	if err := novoVendedor(store, pagamentoFake{aprovar: true}).Reativar(context.Background(), "vendedor-1"); err != nil {
		t.Fatalf("Reativar() erro = %v", err)
	}
	usuario := store.usuarios["vendedor-1"]
	if usuario.BloqueadoParaVendas || usuario.ItensNaoEnviados != 0 {
		t.Fatalf("vendedor não reativado: %+v", usuario)
	}
}

func TestReativarRejeitaPagamentoRecusado(t *testing.T) {
	store := newTestStore()
	store.usuarios["vendedor-1"] = cadastros.Usuario{
		ID: "vendedor-1", BloqueadoParaVendas: true, ItensNaoEnviados: 3,
	}

	err := novoVendedor(store, pagamentoFake{aprovar: false}).Reativar(context.Background(), "vendedor-1")
	if !errors.Is(err, common.ErrPagamentoRecusado) {
		t.Fatalf("erro = %v; esperado ErrPagamentoRecusado", err)
	}
	if usuario := store.usuarios["vendedor-1"]; !usuario.BloqueadoParaVendas {
		t.Fatal("vendedor não deveria ter sido reativado com pagamento recusado")
	}
}

func TestReativarRejeitaVendedorNaoBloqueado(t *testing.T) {
	store := newTestStore()
	store.usuarios["vendedor-1"] = cadastros.Usuario{ID: "vendedor-1"}

	err := novoVendedor(store, pagamentoFake{aprovar: true}).Reativar(context.Background(), "vendedor-1")
	if !errors.Is(err, common.ErrTransicaoInvalida) {
		t.Fatalf("erro = %v; esperado ErrTransicaoInvalida", err)
	}
}
