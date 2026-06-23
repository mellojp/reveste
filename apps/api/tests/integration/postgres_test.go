package integration_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/storage/postgres"
)

func TestIntegracaoFluxoPersistencia(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL nao definida")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("conectar ao PostgreSQL: %v", err)
	}
	defer admin.Close()

	schema := "reveste_test_" + time.Now().UTC().Format("20060102150405_000000000")
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("criar schema: %v", err)
	}
	defer func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	}()

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
	defer pool.Close()

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
	defer repositorio.Close()
	agora := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	usuario := cadastros.Usuario{
		ID: "00000000-0000-4000-8000-000000000001", Nome: "Usuario Teste",
		CPF: "52998224725", Email: "usuario@teste.local", HashSenha: "hash",
		EnderecoPrincipal: cadastros.Endereco{
			CEP: "49000000", Logradouro: "Rua Teste", Numero: "10",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "SE",
		},
		CriadoEm: agora, AtualizadoEm: agora,
	}
	if err := repositorio.CriarUsuario(ctx, usuario); err != nil {
		t.Fatalf("CriarUsuario() erro = %v", err)
	}

	anunciosTeste := []anuncios.Anuncio{
		novoAnuncioIntegracao("00000000-0000-4000-8000-000000000010", usuario.ID, agora),
		novoAnuncioIntegracao("00000000-0000-4000-8000-000000000020", usuario.ID, agora),
	}
	for _, anuncio := range anunciosTeste {
		if err := repositorio.CriarAnuncio(ctx, anuncio); err != nil {
			t.Fatalf("CriarAnuncio() erro = %v", err)
		}
	}

	var grupo sync.WaitGroup
	erros := make(chan error, len(anunciosTeste))
	for indice, anuncio := range anunciosTeste {
		grupo.Add(1)
		go func(indice int, idAnuncio string) {
			defer grupo.Done()
			_, err := repositorio.AdicionarAnuncioAoCarrinho(
				ctx,
				[]string{
					"00000000-0000-4000-8000-000000000101",
					"00000000-0000-4000-8000-000000000102",
				}[indice],
				usuario.ID,
				idAnuncio,
				agora,
			)
			erros <- err
		}(indice, anuncio.ID)
	}
	grupo.Wait()
	close(erros)
	for err := range erros {
		if err != nil {
			t.Fatalf("AdicionarAnuncioAoCarrinho() erro = %v", err)
		}
	}

	carrinho, err := repositorio.ObterOuCriarCarrinho(
		ctx, "00000000-0000-4000-8000-000000000103", usuario.ID, agora,
	)
	if err != nil {
		t.Fatalf("ObterOuCriarCarrinho() erro = %v", err)
	}
	if len(carrinho.IDsAnuncios) != len(anunciosTeste) {
		t.Fatalf("itens no carrinho = %d; esperado %d", len(carrinho.IDsAnuncios), len(anunciosTeste))
	}
}

func novoAnuncioIntegracao(id, idVendedor string, agora time.Time) anuncios.Anuncio {
	return anuncios.Anuncio{
		ID: id, IDVendedor: idVendedor, Titulo: "Anuncio teste",
		Descricao: "Descricao valida para teste", Categoria: anuncios.CategoriaCamisetas,
		Tamanho: "M", Cor: "azul", EstadoConservacao: anuncios.EstadoSeminovo,
		PrecoCentavos: 10_000, Status: anuncios.StatusAnuncioDisponivel,
		PesoGramas: 400, AlturaCm: 3, LarguraCm: 25, ComprimentoCm: 35,
		Fotos: []anuncios.Foto{
			{ID: id[:len(id)-1] + "1", URL: "https://reveste-test.public.blob.vercel-storage.com/1.jpg", Ordem: 0},
			{ID: id[:len(id)-1] + "2", URL: "https://reveste-test.public.blob.vercel-storage.com/2.jpg", Ordem: 1},
		},
		CriadoEm: agora, AtualizadoEm: agora,
	}
}
