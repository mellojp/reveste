package integration_test

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/anuncios"
	"reveste/apps/back/internal/dominio/cadastros"
	"reveste/apps/back/internal/dominio/compras"
	"reveste/apps/back/internal/adaptadores/pagamentos"
	"reveste/apps/back/internal/adaptadores/postgres"
)

type pagamentoContado struct {
	chamadas atomic.Int32
}

func (p *pagamentoContado) CriarCobranca(
	_ context.Context,
	solicitacao casosdeuso.SolicitacaoPagamento,
) (casosdeuso.Cobranca, error) {
	p.chamadas.Add(1)
	return casosdeuso.Cobranca{
		Status:               casosdeuso.CobrancaAprovada,
		Provedor:             "contado",
		IdentificadorExterno: "contado-" + solicitacao.ChaveIdempotencia,
	}, nil
}

// abrirStorePostgres cria um schema isolado, aplica as migracoes e devolve um Store real.
// O schema e descartado ao final do teste.
func abrirStorePostgres(t *testing.T) *postgres.Store {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL nao definida")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("conectar ao PostgreSQL: %v", err)
	}
	t.Cleanup(admin.Close)

	schema := "reveste_test_" + time.Now().UTC().Format("20060102150405_000000000")
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("criar schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	})

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("configurar PostgreSQL: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("criar pool de teste: %v", err)
	}
	t.Cleanup(pool.Close)

	arquivosMigracao, err := filepath.Glob(
		filepath.Join("..", "..", "..", "..", "db", "migrations", "*.up.sql"),
	)
	if err != nil {
		t.Fatalf("listar migracoes: %v", err)
	}
	for _, arquivoMigracao := range arquivosMigracao {
		migracao, err := os.ReadFile(arquivoMigracao)
		if err != nil {
			t.Fatalf("ler migracao %s: %v", arquivoMigracao, err)
		}
		if _, err := pool.Exec(ctx, string(migracao)); err != nil {
			t.Fatalf("aplicar migracao %s: %v", arquivoMigracao, err)
		}
	}

	urlStore, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("interpretar TEST_DATABASE_URL: %v", err)
	}
	consulta := urlStore.Query()
	consulta.Set("search_path", schema)
	urlStore.RawQuery = consulta.Encode()
	repositorio, err := postgres.Open(ctx, urlStore.String())
	if err != nil {
		t.Fatalf("abrir store PostgreSQL: %v", err)
	}
	t.Cleanup(repositorio.Close)
	return repositorio
}

func semearUsuarioIntegracao(t *testing.T, store *postgres.Store, id, cpf, email string, agora time.Time) {
	t.Helper()
	usuario := cadastros.Usuario{
		ID: id, Nome: "Usuario Teste", CPF: cpf, Email: email, HashSenha: "hash",
		EnderecoPrincipal: cadastros.Endereco{
			CEP: "49000000", Logradouro: "Rua Teste", Numero: "10",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "SE",
		},
		CriadoEm: agora, AtualizadoEm: agora,
	}
	if err := store.CriarUsuario(context.Background(), usuario); err != nil {
		t.Fatalf("CriarUsuario(%s) erro = %v", email, err)
	}
}

func TestIntegracaoCheckoutPersisteCompraEReservaItem(t *testing.T) {
	store := abrirStorePostgres(t)
	ctx := context.Background()
	agora := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	const (
		vendedorID  = "00000000-0000-4000-8000-0000000000a1"
		compradorID = "00000000-0000-4000-8000-0000000000b1"
		anuncioID   = "00000000-0000-4000-8000-0000000000c1"
		cartID      = "00000000-0000-4000-8000-0000000000d1"
	)
	semearUsuarioIntegracao(t, store, vendedorID, "52998224725", "vendedor@teste.local", agora)
	semearUsuarioIntegracao(t, store, compradorID, "11144477735", "comprador@teste.local", agora)

	if err := store.CriarAnuncio(ctx, novoAnuncioIntegracao(anuncioID, vendedorID, agora)); err != nil {
		t.Fatalf("CriarAnuncio() erro = %v", err)
	}
	if _, err := store.AdicionarAnuncioAoCarrinho(ctx, cartID, compradorID, anuncioID, agora); err != nil {
		t.Fatalf("AdicionarAnuncioAoCarrinho() erro = %v", err)
	}

	checkout := casosdeuso.NovoControladorCheckout(
		store, store, store, store, store, pagamentos.NovoSimulado(), nil,
		common.GeradorIDCriptografico{}, relogioHTTP{},
		compras.PoliticaCobranca{TaxaServicoPercentual: 10, FretePorPedidoCentavos: 1990},
	)

	compra, err := checkout.FinalizarCompra(ctx, compradorID, "")
	if err != nil {
		t.Fatalf("FinalizarCompra() erro = %v", err)
	}
	if compra.Status != compras.StatusCompraAprovada || len(compra.Pedidos) != 1 {
		t.Fatalf("compra inesperada: status=%v pedidos=%d", compra.Status, len(compra.Pedidos))
	}

	anuncio, err := store.BuscarAnuncioPorID(ctx, anuncioID)
	if err != nil {
		t.Fatalf("BuscarAnuncioPorID() erro = %v", err)
	}
	if anuncio.Status != anuncios.StatusAnuncioVendido {
		t.Fatalf("anuncio nao foi vendido: %v", anuncio.Status)
	}

	pedidos, err := store.ListarPedidosDoComprador(ctx, compradorID)
	if err != nil {
		t.Fatalf("ListarPedidosDoComprador() erro = %v", err)
	}
	if len(pedidos) != 1 || len(pedidos[0].Itens) != 1 {
		t.Fatalf("pedidos persistidos inesperados: %+v", pedidos)
	}
	if pedidos[0].Itens[0].IDAnuncio != anuncioID {
		t.Fatalf("snapshot do item nao referencia o anuncio: %+v", pedidos[0].Itens[0])
	}

	carrinho, err := store.ObterOuCriarCarrinho(ctx, cartID, compradorID, agora)
	if err != nil {
		t.Fatalf("ObterOuCriarCarrinho() erro = %v", err)
	}
	if len(carrinho.IDsAnuncios) != 0 {
		t.Fatalf("carrinho deveria estar vazio apos a compra: %+v", carrinho.IDsAnuncios)
	}
}

func TestIntegracaoCheckoutConcorrenteVendeItemUmaVez(t *testing.T) {
	store := abrirStorePostgres(t)
	ctx := context.Background()
	agora := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	const (
		vendedorID = "00000000-0000-4000-8000-0000000000a2"
		anuncioID  = "00000000-0000-4000-8000-0000000000c2"
		comprador1 = "00000000-0000-4000-8000-0000000000b2"
		comprador2 = "00000000-0000-4000-8000-0000000000b3"
	)
	semearUsuarioIntegracao(t, store, vendedorID, "52998224725", "vendedor2@teste.local", agora)
	semearUsuarioIntegracao(t, store, comprador1, "11144477735", "comprador2@teste.local", agora)
	semearUsuarioIntegracao(t, store, comprador2, "15350946056", "comprador3@teste.local", agora)

	if err := store.CriarAnuncio(ctx, novoAnuncioIntegracao(anuncioID, vendedorID, agora)); err != nil {
		t.Fatalf("CriarAnuncio() erro = %v", err)
	}
	// Os dois compradores colocam o MESMO item unico na sacola.
	if _, err := store.AdicionarAnuncioAoCarrinho(ctx, "00000000-0000-4000-8000-0000000000d2", comprador1, anuncioID, agora); err != nil {
		t.Fatalf("carrinho comprador1: %v", err)
	}
	if _, err := store.AdicionarAnuncioAoCarrinho(ctx, "00000000-0000-4000-8000-0000000000d3", comprador2, anuncioID, agora); err != nil {
		t.Fatalf("carrinho comprador2: %v", err)
	}

	provedor := &pagamentoContado{}
	checkout := casosdeuso.NovoControladorCheckout(
		store, store, store, store, store, provedor, nil,
		common.GeradorIDCriptografico{}, relogioHTTP{},
		compras.PoliticaCobranca{TaxaServicoPercentual: 10, FretePorPedidoCentavos: 1990},
	)

	var grupo sync.WaitGroup
	resultados := make(chan error, 2)
	for _, comprador := range []string{comprador1, comprador2} {
		grupo.Add(1)
		go func(idComprador string) {
			defer grupo.Done()
			_, err := checkout.FinalizarCompra(ctx, idComprador, "")
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
		case errors.Is(err, common.ErrAnuncioIndisponivel) || errors.Is(err, common.ErrSemItensDisponiveis):
			perdas++
		default:
			t.Fatalf("erro inesperado = %v", err)
		}
	}
	if sucessos != 1 || perdas != 1 {
		t.Fatalf("sucessos = %d, perdas = %d; esperado 1 e 1", sucessos, perdas)
	}
	if provedor.chamadas.Load() != 1 {
		t.Fatalf("provedor chamado %d vezes; o perdedor nao poderia ser cobrado", provedor.chamadas.Load())
	}

	anuncio, err := store.BuscarAnuncioPorID(ctx, anuncioID)
	if err != nil {
		t.Fatalf("BuscarAnuncioPorID() erro = %v", err)
	}
	if anuncio.Status != anuncios.StatusAnuncioVendido {
		t.Fatalf("anuncio deveria estar vendido uma vez: %v", anuncio.Status)
	}
}

func TestIntegracaoExpiracaoLiberaReservaPendente(t *testing.T) {
	store := abrirStorePostgres(t)
	ctx := context.Background()
	agora := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	const (
		vendedorID  = "00000000-0000-4000-8000-0000000000a4"
		compradorID = "00000000-0000-4000-8000-0000000000b4"
		anuncioID   = "00000000-0000-4000-8000-0000000000c4"
		carrinhoID  = "00000000-0000-4000-8000-0000000000d4"
		pagamentoID = "00000000-0000-4000-8000-0000000000e4"
	)
	semearUsuarioIntegracao(t, store, vendedorID, "52998224725", "vendedor-expira@teste.local", agora)
	semearUsuarioIntegracao(t, store, compradorID, "11144477735", "comprador-expira@teste.local", agora)
	if err := store.CriarAnuncio(ctx, novoAnuncioIntegracao(anuncioID, vendedorID, agora)); err != nil {
		t.Fatalf("CriarAnuncio() erro = %v", err)
	}
	if _, err := store.AdicionarAnuncioAoCarrinho(ctx, carrinhoID, compradorID, anuncioID, agora); err != nil {
		t.Fatalf("AdicionarAnuncioAoCarrinho() erro = %v", err)
	}

	checkout := casosdeuso.NovoControladorCheckout(
		store, store, store, store, store, pagamentos.NovoSimulado(), nil,
		common.GeradorIDCriptografico{}, relogioHTTP{},
		compras.PoliticaCobranca{TaxaServicoPercentual: 10, FretePorPedidoCentavos: 1990},
	)
	intencao, err := checkout.ResumoCheckout(ctx, compradorID, "")
	if err != nil {
		t.Fatalf("ResumoCheckout() erro = %v", err)
	}
	pagamento := compras.Pagamento{
		ID: pagamentoID, IDCompra: intencao.ID, Provedor: "pendente",
		Status: compras.StatusPagamentoPendente, ValorCentavos: intencao.ValorFinalPagoCentavos,
		ChaveIdempotencia: intencao.ChaveIdempotencia,
	}
	if _, criada, err := store.IniciarCompra(ctx, intencao, pagamento, compradorID); err != nil || !criada {
		t.Fatalf("IniciarCompra() criada=%v erro=%v", criada, err)
	}
	anuncio, err := store.BuscarAnuncioPorID(ctx, anuncioID)
	if err != nil || anuncio.Status != anuncios.StatusAnuncioReservado {
		t.Fatalf("anuncio nao foi reservado: status=%v erro=%v", anuncio.Status, err)
	}

	afetadas, err := store.ExpirarComprasPendentes(ctx, intencao.ExpiraEm.Add(time.Second))
	if err != nil || afetadas != 1 {
		t.Fatalf("ExpirarComprasPendentes() = %d, %v; esperado 1, nil", afetadas, err)
	}
	anuncio, err = store.BuscarAnuncioPorID(ctx, anuncioID)
	if err != nil || anuncio.Status != anuncios.StatusAnuncioDisponivel {
		t.Fatalf("reserva expirada nao foi liberada: status=%v erro=%v", anuncio.Status, err)
	}
	expirada, err := store.BuscarCompraPorChave(ctx, intencao.ChaveIdempotencia)
	if err != nil || expirada.Status != compras.StatusCompraExpirada {
		t.Fatalf("compra nao foi expirada: status=%v erro=%v", expirada.Status, err)
	}
}
