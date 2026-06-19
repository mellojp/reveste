package casosdeuso_test

import (
	"context"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/dominio/compras"
	"reveste/apps/api/internal/dominio/interacao"
)

func novoNotificacoes(store *Store) *casosdeuso.ControladorNotificacoes {
	return casosdeuso.NovoControladorNotificacoes(
		store, relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
}

func TestNotificacoesListaContaEMarcaLidas(t *testing.T) {
	store := newTestStore()
	ctx := context.Background()
	controlador := novoNotificacoes(store)

	for i, conteudo := range []string{"primeira", "segunda"} {
		_ = store.CriarNotificacao(ctx, interacao.Notificacao{
			ID: string(rune('a' + i)), IDUsuario: "u-1", Tipo: interacao.NotificacaoPedidoEnviado,
			Conteudo: conteudo, CriadaEm: time.Now(),
		})
	}
	_ = store.CriarNotificacao(ctx, interacao.Notificacao{
		ID: "x", IDUsuario: "u-2", Conteudo: "de outro", CriadaEm: time.Now(),
	})

	lista, err := controlador.Listar(ctx, "u-1")
	if err != nil || len(lista) != 2 {
		t.Fatalf("Listar() = %d itens, err=%v; esperados 2 do usuário u-1", len(lista), err)
	}
	if lista[0].Conteudo != "segunda" {
		t.Fatalf("ordem incorreta: esperava a mais recente primeiro, veio %q", lista[0].Conteudo)
	}

	if n, err := controlador.ContarNaoLidas(ctx, "u-1"); err != nil || n != 2 {
		t.Fatalf("ContarNaoLidas() = %d, err=%v; esperado 2", n, err)
	}
	if err := controlador.MarcarTodasLidas(ctx, "u-1"); err != nil {
		t.Fatalf("MarcarTodasLidas() erro = %v", err)
	}
	if n, _ := controlador.ContarNaoLidas(ctx, "u-1"); n != 0 {
		t.Fatalf("após marcar lidas, não lidas = %d; esperado 0", n)
	}
	if n, _ := controlador.ContarNaoLidas(ctx, "u-2"); n != 1 {
		t.Fatalf("notificação de outro usuário não deveria ser afetada: %d", n)
	}
}

func TestMarcarEnviadoNotificaComprador(t *testing.T) {
	store := newTestStore()
	fake := &pedidosFake{pedido: compras.Pedido{
		ID: "pedido-1", IDComprador: "comprador-1", IDVendedor: "vendedor-1",
		Status: compras.StatusPedidoAguardandoEnvio,
	}}
	controlador := novoPedidosComNotificacoes(fake, store)

	if err := controlador.MarcarEnviado(context.Background(), "vendedor-1", "pedido-1", casosdeuso.EntradaEnvio{
		CodigoRastreio: "BR123",
	}); err != nil {
		t.Fatalf("MarcarEnviado() erro = %v", err)
	}

	notificacoes, _ := store.ListarNotificacoes(context.Background(), "comprador-1", 10)
	if len(notificacoes) != 1 {
		t.Fatalf("esperava 1 notificação para o comprador, veio %d", len(notificacoes))
	}
	if notificacoes[0].Tipo != interacao.NotificacaoPedidoEnviado || notificacoes[0].IDPedido != "pedido-1" {
		t.Fatalf("notificação inesperada: %+v", notificacoes[0])
	}
}

func TestNotificacoesRemoveUmaELimpaSomenteDoUsuario(t *testing.T) {
	store := newTestStore()
	ctx := context.Background()
	controlador := novoNotificacoes(store)
	for _, n := range []interacao.Notificacao{
		{ID: "n1", IDUsuario: "u-1", Conteudo: "uma"},
		{ID: "n2", IDUsuario: "u-1", Conteudo: "duas"},
		{ID: "n3", IDUsuario: "u-2", Conteudo: "de outro"},
	} {
		if err := store.CriarNotificacao(ctx, n); err != nil {
			t.Fatal(err)
		}
	}

	if err := controlador.Remover(ctx, "u-1", "n1"); err != nil {
		t.Fatalf("Remover() erro = %v", err)
	}
	lista, _ := controlador.Listar(ctx, "u-1")
	if len(lista) != 1 || lista[0].ID != "n2" {
		t.Fatalf("lista após remover = %+v", lista)
	}
	if err := controlador.Remover(ctx, "u-2", "n2"); err != nil {
		t.Fatalf("remoção fora do proprietário deveria ser inócua: %v", err)
	}
	if err := controlador.Limpar(ctx, "u-1"); err != nil {
		t.Fatalf("Limpar() erro = %v", err)
	}
	if lista, _ = controlador.Listar(ctx, "u-1"); len(lista) != 0 {
		t.Fatalf("caixa do usuário não foi limpa: %+v", lista)
	}
	if lista, _ = controlador.Listar(ctx, "u-2"); len(lista) != 1 || lista[0].ID != "n3" {
		t.Fatalf("notificações de outro usuário foram afetadas: %+v", lista)
	}
}
