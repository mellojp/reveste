package casosdeuso_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/anuncios"
	"reveste/apps/back/internal/dominio/cadastros"
	"reveste/apps/back/internal/dominio/compras"
	"reveste/apps/back/internal/dominio/interacao"
)

type pagamentoFake struct {
	aprovar  bool
	chamadas *atomic.Int32
}

func (p pagamentoFake) Processar(
	_ context.Context,
	solicitacao casosdeuso.SolicitacaoPagamento,
) (casosdeuso.ResultadoPagamento, error) {
	if p.chamadas != nil {
		p.chamadas.Add(1)
	}
	return casosdeuso.ResultadoPagamento{
		Aprovado:             p.aprovar,
		Provedor:             "fake",
		IdentificadorExterno: "ext-" + solicitacao.ChaveIdempotencia,
	}, nil
}

type freteFake struct {
	valorCentavos int64
	err           error
	origemVista   string
	destinoVisto  string
}

func (f *freteFake) Cotar(
	_ context.Context,
	origemCEP, destinoCEP string,
	_ []casosdeuso.ItemFrete,
) (casosdeuso.CotacaoFrete, error) {
	f.origemVista = origemCEP
	f.destinoVisto = destinoCEP
	if f.err != nil {
		return casosdeuso.CotacaoFrete{}, f.err
	}
	return casosdeuso.CotacaoFrete{ValorCentavos: f.valorCentavos, Provedor: "fake"}, nil
}

func novoCheckout(store *Store, pagamento casosdeuso.ProcessadorPagamento) *casosdeuso.ControladorCheckout {
	return novoCheckoutComFrete(store, pagamento, nil)
}

func novoCheckoutComFrete(
	store *Store,
	pagamento casosdeuso.ProcessadorPagamento,
	cotador casosdeuso.CotadorFrete,
) *casosdeuso.ControladorCheckout {
	return casosdeuso.NovoControladorCheckout(
		store, store, store, store, store, pagamento, cotador,
		&geradorSequencial{},
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
		compras.PoliticaCobranca{TaxaServicoPercentual: 10, FretePorPedidoCentavos: 1990},
	)
}

func semearCheckout(store *Store, idComprador, idAnuncio, idVendedor string, preco int64, status anuncios.StatusAnuncio) {
	store.usuarios[idComprador] = cadastros.Usuario{
		ID: idComprador, Nome: "Comprador Teste",
		EnderecoPrincipal: cadastros.Endereco{
			CEP: "49000000", Logradouro: "Rua A", Numero: "1",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "SE",
		},
	}
	store.anuncios[idAnuncio] = anuncios.Anuncio{
		ID: idAnuncio, IDVendedor: idVendedor, Titulo: "Casaco",
		Categoria: anuncios.CategoriaCasacos, Tamanho: "M", Cor: "azul",
		EstadoConservacao: anuncios.EstadoSeminovo, PrecoCentavos: preco, Status: status,
	}
	store.carrinhoPorUsuario[idComprador] = compras.Carrinho{
		ID: "carrinho-" + idComprador, IDUsuario: idComprador,
		IDsAnuncios: []string{idAnuncio},
	}
}

func TestCheckoutFinalizaCompraEReservaItem(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	checkout := novoCheckout(store, pagamentoFake{aprovar: true})

	compra, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "")
	if err != nil {
		t.Fatalf("FinalizarCompra() erro = %v", err)
	}
	if compra.Status != compras.StatusCompraAprovada {
		t.Fatalf("status da compra = %v; esperado aprovada", compra.Status)
	}
	if len(compra.Pedidos) != 1 {
		t.Fatalf("pedidos = %d; esperado 1", len(compra.Pedidos))
	}
	pedido := compra.Pedidos[0]
	if pedido.Status != compras.StatusPedidoAguardandoEnvio {
		t.Fatalf("status do pedido = %v; esperado aguardando_envio", pedido.Status)
	}
	if pedido.ValorTotalItensCentavos != 10_000 || pedido.ValorFreteCentavos != 1990 {
		t.Fatalf("valores do pedido inesperados: %+v", pedido)
	}
	if pedido.TaxaServicoCentavos != 1_000 {
		t.Fatalf("taxa de servico = %d; esperado 1000 (10%%)", pedido.TaxaServicoCentavos)
	}
	if pedido.ValorLiquidoVendedorCentavos != 10_990 { // 10000 + 1990 - 1000
		t.Fatalf("liquido do vendedor = %d; esperado 10990", pedido.ValorLiquidoVendedorCentavos)
	}
	if compra.ValorFinalPagoCentavos != 11_990 { // itens + frete
		t.Fatalf("total pago = %d; esperado 11990", compra.ValorFinalPagoCentavos)
	}
	if store.anuncios["a1"].Status != anuncios.StatusAnuncioVendido {
		t.Fatalf("anuncio nao foi vendido: %v", store.anuncios["a1"].Status)
	}
	if itens := store.carrinhoPorUsuario["comprador-1"].IDsAnuncios; len(itens) != 0 {
		t.Fatalf("carrinho deveria estar vazio: %+v", itens)
	}
	notificacoes, err := store.ListarNotificacoes(context.Background(), "vendedor-1", 10)
	if err != nil || len(notificacoes) != 1 {
		t.Fatalf("notificações do vendedor = %d, erro %v; esperada nova venda", len(notificacoes), err)
	}
	if notificacoes[0].Tipo != interacao.NotificacaoVendaRealizada || notificacoes[0].IDPedido != pedido.ID {
		t.Fatalf("notificação de venda inesperada: %+v", notificacoes[0])
	}
	pedidos, err := checkout.ListarPedidos(context.Background(), "comprador-1")
	if err != nil || len(pedidos) != 1 {
		t.Fatalf("ListarPedidos() = %d pedidos, erro %v; esperado 1", len(pedidos), err)
	}
}

func TestCheckoutUsaFreteCotado(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	// o vendedor precisa de endereco principal para servir de origem da cotacao.
	store.usuarios["vendedor-1"] = cadastros.Usuario{
		ID: "vendedor-1", Nome: "Vendedor Teste",
		EnderecoPrincipal: cadastros.Endereco{CEP: "01310100", Cidade: "São Paulo", Estado: "SP"},
	}
	cotador := &freteFake{valorCentavos: 3_500}
	checkout := novoCheckoutComFrete(store, pagamentoFake{aprovar: true}, cotador)

	compra, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "")
	if err != nil {
		t.Fatalf("FinalizarCompra() erro = %v", err)
	}
	pedido := compra.Pedidos[0]
	if pedido.ValorFreteCentavos != 3_500 {
		t.Fatalf("frete = %d; esperado 3500 (cotado)", pedido.ValorFreteCentavos)
	}
	if compra.ValorFinalPagoCentavos != 13_500 { // 10000 itens + 3500 frete
		t.Fatalf("total pago = %d; esperado 13500", compra.ValorFinalPagoCentavos)
	}
	if cotador.origemVista != "01310100" || cotador.destinoVisto != "49000000" {
		t.Fatalf("CEPs de cotacao inesperados: origem=%q destino=%q", cotador.origemVista, cotador.destinoVisto)
	}
}

func TestCheckoutCaiNoFreteContingenciaQuandoCotacaoFalha(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	store.usuarios["vendedor-1"] = cadastros.Usuario{
		ID: "vendedor-1", EnderecoPrincipal: cadastros.Endereco{CEP: "01310100"},
	}
	cotador := &freteFake{err: common.ErrCotacaoFreteIndisponivel}
	checkout := novoCheckoutComFrete(store, pagamentoFake{aprovar: true}, cotador)

	compra, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "")
	if err != nil {
		t.Fatalf("FinalizarCompra() erro = %v", err)
	}
	if compra.Pedidos[0].ValorFreteCentavos != 1990 {
		t.Fatalf("frete = %d; esperado 1990 (contingência)", compra.Pedidos[0].ValorFreteCentavos)
	}
}

func TestResumoCheckoutProjetaSemReservarOuPersistir(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	checkout := novoCheckout(store, pagamentoFake{aprovar: true})

	resumo, err := checkout.ResumoCheckout(context.Background(), "comprador-1", "")
	if err != nil {
		t.Fatalf("ResumoCheckout() erro = %v", err)
	}
	if resumo.Status != compras.StatusCompraAguardandoPagamento {
		t.Fatalf("status = %v; esperado aguardando_pagamento", resumo.Status)
	}
	if len(resumo.Pedidos) != 1 || resumo.Pedidos[0].ValorFreteCentavos != 1990 {
		t.Fatalf("projecao inesperada: %+v", resumo.Pedidos)
	}
	if resumo.ValorFinalPagoCentavos != 11_990 { // itens + frete, igual ao checkout real
		t.Fatalf("total projetado = %d; esperado 11990", resumo.ValorFinalPagoCentavos)
	}
	// A revisao nao pode reservar o item nem registrar pedidos.
	if store.anuncios["a1"].Status != anuncios.StatusAnuncioDisponivel {
		t.Fatalf("anuncio nao deveria ser reservado: %v", store.anuncios["a1"].Status)
	}
	if pedidos, _ := checkout.ListarPedidos(context.Background(), "comprador-1"); len(pedidos) != 0 {
		t.Fatalf("nenhum pedido deveria existir apos a revisao: %d", len(pedidos))
	}
}

func TestCheckoutVendeItemUmaVezSobConcorrencia(t *testing.T) {
	store := newTestStore()
	// Dois compradores com o MESMO item unico na sacola, finalizando ao mesmo tempo.
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	semearCheckout(store, "comprador-2", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	var chamadasPagamento atomic.Int32
	checkout := novoCheckout(store, pagamentoFake{aprovar: true, chamadas: &chamadasPagamento})

	var grupo sync.WaitGroup
	resultados := make(chan error, 2)
	for _, comprador := range []string{"comprador-1", "comprador-2"} {
		grupo.Add(1)
		go func(idComprador string) {
			defer grupo.Done()
			_, err := checkout.FinalizarCompra(context.Background(), idComprador, "")
			resultados <- err
		}(comprador)
	}
	grupo.Wait()
	close(resultados)

	var sucessos, perdas int
	for err := range resultados {
		switch {
		case err == nil:
			sucessos++
		// O perdedor da corrida recebe um dos dois: ErrAnuncioIndisponivel (montou a compra
		// antes do vencedor concluir) ou ErrSemItensDisponiveis (ja leu o item como vendido).
		case errors.Is(err, common.ErrAnuncioIndisponivel) || errors.Is(err, common.ErrSemItensDisponiveis):
			perdas++
		default:
			t.Fatalf("erro inesperado = %v", err)
		}
	}
	if sucessos != 1 || perdas != 1 {
		t.Fatalf("sucessos = %d, perdas = %d; esperado 1 e 1", sucessos, perdas)
	}
	if store.anuncios["a1"].Status != anuncios.StatusAnuncioVendido {
		t.Fatalf("anuncio deveria estar vendido exatamente uma vez: %v", store.anuncios["a1"].Status)
	}
	if chamadasPagamento.Load() != 1 {
		t.Fatalf("pagamento foi processado %d vezes; esperado apenas pelo vencedor da reserva", chamadasPagamento.Load())
	}
}

func TestCheckoutCarrinhoVazio(t *testing.T) {
	store := newTestStore()
	store.usuarios["comprador-1"] = cadastros.Usuario{ID: "comprador-1", Nome: "Comprador"}
	checkout := novoCheckout(store, pagamentoFake{aprovar: true})

	_, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "")
	if !errors.Is(err, common.ErrCarrinhoVazio) {
		t.Fatalf("erro = %v; esperado ErrCarrinhoVazio", err)
	}
}

func TestCheckoutSemItensDisponiveis(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioReservado)
	checkout := novoCheckout(store, pagamentoFake{aprovar: true})

	_, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "")
	if !errors.Is(err, common.ErrSemItensDisponiveis) {
		t.Fatalf("erro = %v; esperado ErrSemItensDisponiveis", err)
	}
}

func TestCheckoutIgnoraProprioAnuncio(t *testing.T) {
	store := newTestStore()
	// O comprador e o proprio vendedor do item: deve ser descartado.
	semearCheckout(store, "comprador-1", "a1", "comprador-1", 10_000, anuncios.StatusAnuncioDisponivel)
	checkout := novoCheckout(store, pagamentoFake{aprovar: true})

	_, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "")
	if !errors.Is(err, common.ErrSemItensDisponiveis) {
		t.Fatalf("erro = %v; esperado ErrSemItensDisponiveis", err)
	}
}

func TestCheckoutPagamentoRecusadoNaoReservaItem(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	checkout := novoCheckout(store, pagamentoFake{aprovar: false})

	_, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "")
	if !errors.Is(err, common.ErrPagamentoRecusado) {
		t.Fatalf("erro = %v; esperado ErrPagamentoRecusado", err)
	}
	if store.anuncios["a1"].Status != anuncios.StatusAnuncioDisponivel {
		t.Fatalf("anuncio nao deveria ter sido reservado: %v", store.anuncios["a1"].Status)
	}
}

func TestCheckoutIdempotenteRetornaCompraExistente(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	carrinho := store.carrinhoPorUsuario["comprador-1"]
	chave := compras.ChaveIdempotenciaCarrinho(
		"comprador-1",
		[]string{carrinho.ID, carrinho.AtualizadoEm.UTC().Format(time.RFC3339Nano), "a1"},
	)
	store.comprasPorChave[chave] = compras.Compra{
		ID: "compra-existente", Status: compras.StatusCompraAprovada, ChaveIdempotencia: chave,
	}
	checkout := novoCheckout(store, pagamentoFake{aprovar: true})

	compra, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "")
	if err != nil {
		t.Fatalf("FinalizarCompra() erro = %v", err)
	}
	if compra.ID != "compra-existente" {
		t.Fatalf("compra retornada = %q; esperado a existente", compra.ID)
	}
	if store.anuncios["a1"].Status != anuncios.StatusAnuncioDisponivel {
		t.Fatalf("compra idempotente nao deveria vender o item de novo: %v", store.anuncios["a1"].Status)
	}
}

func TestProcessarExpiracoesLiberaReserva(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioReservado)
	chave := "checkout-expirado"
	compra := compras.Compra{
		ID: "compra-1", IDComprador: "comprador-1",
		Status:            compras.StatusCompraAguardandoPagamento,
		ChaveIdempotencia: chave,
		ExpiraEm:          time.Date(2026, 6, 10, 11, 0, 0, 0, time.UTC),
		Pedidos: []compras.Pedido{{
			ID: "pedido-1", IDComprador: "comprador-1",
			Status: compras.StatusPedidoAguardandoPagamento,
			Itens:  []compras.ItemPedido{{IDAnuncio: "a1"}},
		}},
	}
	store.comprasPorChave[chave] = compra
	store.pedidosPorComprador["comprador-1"] = compra.Pedidos

	afetadas, err := novoCheckout(store, pagamentoFake{aprovar: true}).ProcessarExpiracoes(context.Background())
	if err != nil || afetadas != 1 {
		t.Fatalf("ProcessarExpiracoes() = %d, %v; esperado 1, nil", afetadas, err)
	}
	if store.anuncios["a1"].Status != anuncios.StatusAnuncioDisponivel {
		t.Fatalf("reserva expirada nao foi liberada: %v", store.anuncios["a1"].Status)
	}
	if store.comprasPorChave[chave].Status != compras.StatusCompraExpirada {
		t.Fatalf("compra nao foi expirada: %v", store.comprasPorChave[chave].Status)
	}
}
