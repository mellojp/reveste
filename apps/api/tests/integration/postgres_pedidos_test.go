package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/compras"
	"reveste/apps/api/internal/dominio/interacao"
	"reveste/apps/api/internal/storage/pagamentos"
	"reveste/apps/api/internal/storage/postgres"
)

func checkoutDeTeste(store *postgres.Store) *casosdeuso.ControladorCheckout {
	return casosdeuso.NovoControladorCheckout(
		store, store, store, store, store, pagamentos.NovoSimulado(),
		common.GeradorIDCriptografico{}, relogioHTTP{},
		compras.PoliticaCobranca{TaxaServicoPercentual: 10, FretePorPedidoCentavos: 1990},
	)
}

func TestIntegracaoCicloDePedidoCompleto(t *testing.T) {
	store := abrirStorePostgres(t)
	ctx := context.Background()
	agora := time.Date(2030, 6, 10, 12, 0, 0, 0, time.UTC)

	const (
		vendedorID  = "00000000-0000-4000-8000-0000000000e1"
		compradorID = "00000000-0000-4000-8000-0000000000f1"
		anuncioID   = "00000000-0000-4000-8000-000000000a01"
	)
	semearUsuarioIntegracao(t, store, vendedorID, "52998224725", "vendedor-ciclo@teste.local", agora)
	semearUsuarioIntegracao(t, store, compradorID, "11144477735", "comprador-ciclo@teste.local", agora)
	if err := store.CriarAnuncio(ctx, novoAnuncioIntegracao(anuncioID, vendedorID, agora)); err != nil {
		t.Fatalf("CriarAnuncio() erro = %v", err)
	}
	if _, err := store.AdicionarAnuncioAoCarrinho(ctx, "00000000-0000-4000-8000-000000000a99", compradorID, anuncioID, agora); err != nil {
		t.Fatalf("AdicionarAnuncioAoCarrinho() erro = %v", err)
	}

	compra, err := checkoutDeTeste(store).FinalizarCompra(ctx, compradorID, "")
	if err != nil {
		t.Fatalf("FinalizarCompra() erro = %v", err)
	}
	pedidoID := compra.Pedidos[0].ID

	// Um usuário que não é o vendedor não pode marcar como enviado.
	if err := store.MarcarPedidoEnviado(ctx, pedidoID, compradorID, "Correios", "BR1", agora); !errors.Is(err, common.ErrNaoPermitido) {
		t.Fatalf("envio por não-vendedor: erro = %v; esperado ErrNaoPermitido", err)
	}
	if err := store.MarcarPedidoEnviado(ctx, pedidoID, vendedorID, "Correios", "BR123456789", agora); err != nil {
		t.Fatalf("MarcarPedidoEnviado() erro = %v", err)
	}

	// Um usuário que não é o comprador não pode confirmar o recebimento.
	if err := store.ConfirmarRecebimentoPedido(ctx, pedidoID, vendedorID, agora); !errors.Is(err, common.ErrNaoPermitido) {
		t.Fatalf("recebimento por não-comprador: erro = %v; esperado ErrNaoPermitido", err)
	}
	if err := store.ConfirmarRecebimentoPedido(ctx, pedidoID, compradorID, agora); err != nil {
		t.Fatalf("ConfirmarRecebimentoPedido() erro = %v", err)
	}

	pedidos, err := store.ListarPedidosDoComprador(ctx, compradorID)
	if err != nil {
		t.Fatalf("ListarPedidosDoComprador() erro = %v", err)
	}
	if pedidos[0].Status != compras.StatusPedidoFinalizado {
		t.Fatalf("pedido não finalizado: %v", pedidos[0].Status)
	}
	if pedidos[0].Itens[0].Status != compras.StatusItemRecebido {
		t.Fatalf("item não recebido: %v", pedidos[0].Itens[0].Status)
	}

	avaliacao := interacao.Avaliacao{
		ID: "00000000-0000-4000-8000-000000000a02", IDPedido: pedidoID,
		IDUsuarioAutor: compradorID, IDUsuarioAvaliado: vendedorID, Nota: 5, CriadaEm: agora,
	}
	if err := store.RegistrarAvaliacao(ctx, avaliacao); err != nil {
		t.Fatalf("RegistrarAvaliacao() erro = %v", err)
	}
	avaliacao.ID = "00000000-0000-4000-8000-000000000a03"
	if err := store.RegistrarAvaliacao(ctx, avaliacao); !errors.Is(err, common.ErrConflito) {
		t.Fatalf("avaliação duplicada: erro = %v; esperado ErrConflito", err)
	}

	media, err := store.MediaAvaliacoesVendedor(ctx, vendedorID)
	if err != nil {
		t.Fatalf("MediaAvaliacoesVendedor() erro = %v", err)
	}
	if media.Quantidade != 1 || media.Media != 5 {
		t.Fatalf("média inesperada: %+v", media)
	}
}

func TestIntegracaoProcessarItensVencidosBloqueiaVendedor(t *testing.T) {
	store := abrirStorePostgres(t)
	ctx := context.Background()
	agora := time.Date(2030, 6, 10, 12, 0, 0, 0, time.UTC)

	const (
		vendedorID  = "00000000-0000-4000-8000-0000000000e2"
		compradorID = "00000000-0000-4000-8000-0000000000f2"
	)
	semearUsuarioIntegracao(t, store, vendedorID, "52998224725", "vendedor-prazo@teste.local", agora)
	semearUsuarioIntegracao(t, store, compradorID, "11144477735", "comprador-prazo@teste.local", agora)

	// IDs diferem antes do ultimo caractere para nao colidir nos IDs de foto derivados
	// por novoAnuncioIntegracao (que remove o ultimo char do ID do anuncio).
	idsAnuncios := []string{
		"00000000-0000-4000-8000-0000000b0010",
		"00000000-0000-4000-8000-0000000b0020",
		"00000000-0000-4000-8000-0000000b0030",
	}
	for i, id := range idsAnuncios {
		if err := store.CriarAnuncio(ctx, novoAnuncioIntegracao(id, vendedorID, agora)); err != nil {
			t.Fatalf("CriarAnuncio(%d) erro = %v", i, err)
		}
		if _, err := store.AdicionarAnuncioAoCarrinho(ctx, "00000000-0000-4000-8000-0000000ca001", compradorID, id, agora); err != nil {
			t.Fatalf("AdicionarAnuncioAoCarrinho(%d) erro = %v", i, err)
		}
	}
	if _, err := checkoutDeTeste(store).FinalizarCompra(ctx, compradorID, ""); err != nil {
		t.Fatalf("FinalizarCompra() erro = %v", err)
	}

	// Avança o relógio para além do prazo de envio (checkout + 5 dias).
	depoisDoPrazo := agora.Add(6 * 24 * time.Hour)
	afetados, err := store.ProcessarItensVencidos(ctx, depoisDoPrazo, 3)
	if err != nil {
		t.Fatalf("ProcessarItensVencidos() erro = %v", err)
	}
	if afetados != 3 {
		t.Fatalf("itens afetados = %d; esperado 3", afetados)
	}

	vendedor, err := store.BuscarUsuarioPorID(ctx, vendedorID)
	if err != nil {
		t.Fatalf("BuscarUsuarioPorID() erro = %v", err)
	}
	if !vendedor.BloqueadoParaVendas {
		t.Fatalf("vendedor deveria estar bloqueado após 3 itens não enviados")
	}
	if vendedor.ItensNaoEnviados != 3 {
		t.Fatalf("itens não enviados = %d; esperado 3", vendedor.ItensNaoEnviados)
	}
	pedidos, err := store.ListarPedidosDoComprador(ctx, compradorID)
	if err != nil || len(pedidos) != 1 {
		t.Fatalf("ListarPedidosDoComprador() = %d, %v; esperado um pedido", len(pedidos), err)
	}
	if pedidos[0].Status != compras.StatusPedidoCancelado {
		t.Fatalf("pedido vencido permaneceu em estado inconsistente: %v", pedidos[0].Status)
	}
	if pedidos[0].Entrega == nil || pedidos[0].Entrega.Status != compras.StatusEntregaFalhou {
		t.Fatalf("entrega vencida nao foi marcada como falha: %+v", pedidos[0].Entrega)
	}
	for _, item := range pedidos[0].Itens {
		if item.Status != compras.StatusItemNaoEnviado {
			t.Fatalf("item vencido com status %v; esperado nao_enviado", item.Status)
		}
	}
	if err := store.MarcarPedidoEnviado(
		ctx, pedidos[0].ID, vendedorID, "Correios", "TARDIO", depoisDoPrazo,
	); !errors.Is(err, common.ErrNaoPermitido) {
		t.Fatalf("pedido cancelado aceitou envio tardio: %v", err)
	}
}
