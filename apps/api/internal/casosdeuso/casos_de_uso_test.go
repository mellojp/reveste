package casosdeuso_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
)

type geradorSequencial struct {
	mu      sync.Mutex
	proximo int
}

func (g *geradorSequencial) Novo() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.proximo++
	return "id-" + time.Unix(int64(g.proximo), 0).UTC().Format("150405")
}

type relogioFixo struct {
	agora time.Time
}

func (r relogioFixo) Agora() time.Time {
	return r.agora
}

func TestControladoresCadastroAnuncioCarrinho(t *testing.T) {
	store := newTestStore()
	ids := &geradorSequencial{}
	cadastrosCU := casosdeuso.NovoControladorCadastro(
		store,
		store,
		ids,
		common.ProcessadorPBKDF2{Iteracoes: 100_000},
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
	anunciosCU := casosdeuso.NovoControladorAnuncio(
		store, store, ids,
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
	comprasCU := casosdeuso.NovoControladorCarrinho(
		store, store, ids,
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
	ctx := context.Background()

	vendedor := cadastrarUsuario(t, ctx, cadastrosCU, "Vendedor Teste", "52998224725", "vendedor@teste.local")
	comprador := cadastrarUsuario(t, ctx, cadastrosCU, "Comprador Teste", "11144477735", "comprador@teste.local")

	anuncio, err := anunciosCU.CriarAnuncio(ctx, vendedor.ID, casosdeuso.EntradaAnuncio{
		Titulo: "Jaqueta jeans", Descricao: "Jaqueta jeans em excelente estado",
		Categoria: "Jaqueta", Tamanho: "m", Cor: "Azul",
		EstadoConservacao: anuncios.EstadoSeminovo, PrecoCentavos: 12_000,
		URLsFotos: []string{"https://exemplo.test/1.jpg", "https://exemplo.test/2.jpg"},
	})
	if err != nil {
		t.Fatalf("CriarAnuncio() erro = %v", err)
	}

	carrinho, err := comprasCU.AdicionarAoCarrinho(ctx, comprador.ID, anuncio.ID)
	if err != nil {
		t.Fatalf("AdicionarAoCarrinho() erro = %v", err)
	}
	if len(carrinho.Anuncios) != 1 || carrinho.TotalCentavos != 12_000 {
		t.Fatalf("carrinho inesperado: %+v", carrinho)
	}

	carrinho, err = comprasCU.AdicionarAoCarrinho(ctx, comprador.ID, anuncio.ID)
	if err != nil {
		t.Fatalf("segunda adicao retornou erro = %v", err)
	}
	if len(carrinho.Anuncios) != 1 {
		t.Fatalf("anuncio duplicado no carrinho: %+v", carrinho)
	}
}

func TestAdicoesConcorrentesAoCarrinhoSaoPreservadas(t *testing.T) {
	store := newTestStore()
	agora := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()
	anunciosDoTeste := []anuncios.Anuncio{
		{ID: "anuncio-1", IDVendedor: "vendedor-1", Status: anuncios.StatusAnuncioDisponivel},
		{ID: "anuncio-2", IDVendedor: "vendedor-2", Status: anuncios.StatusAnuncioDisponivel},
	}
	for _, anuncio := range anunciosDoTeste {
		store.anuncios[anuncio.ID] = anuncio
	}
	controlador := casosdeuso.NovoControladorCarrinho(
		store, store, &geradorSequencial{}, relogioFixo{agora: agora},
	)

	var grupo sync.WaitGroup
	erros := make(chan error, len(anunciosDoTeste))
	for _, anuncio := range anunciosDoTeste {
		grupo.Add(1)
		go func(idAnuncio string) {
			defer grupo.Done()
			_, err := controlador.AdicionarAoCarrinho(ctx, "comprador-1", idAnuncio)
			erros <- err
		}(anuncio.ID)
	}
	grupo.Wait()
	close(erros)
	for err := range erros {
		if err != nil {
			t.Fatalf("AdicionarAoCarrinho() erro = %v", err)
		}
	}

	carrinho, err := controlador.ObterCarrinho(ctx, "comprador-1")
	if err != nil {
		t.Fatalf("ObterCarrinho() erro = %v", err)
	}
	if len(carrinho.Anuncios) != len(anunciosDoTeste) {
		t.Fatalf("quantidade de anuncios = %d; esperado %d", len(carrinho.Anuncios), len(anunciosDoTeste))
	}
}

type usuariosInexistentes struct{}

func (usuariosInexistentes) CriarUsuario(context.Context, cadastros.Usuario) error {
	return nil
}

func (usuariosInexistentes) BuscarUsuarioPorID(context.Context, string) (cadastros.Usuario, error) {
	return cadastros.Usuario{}, common.ErrNaoEncontrado
}

func (usuariosInexistentes) BuscarUsuarioPorEmailOuCPF(context.Context, string) (cadastros.Usuario, error) {
	return cadastros.Usuario{}, common.ErrNaoEncontrado
}

type comparadorContador struct {
	chamadas int
}

func (c *comparadorContador) Gerar(string) (string, error) {
	return "", nil
}

func (c *comparadorContador) Comparar(string, string) bool {
	c.chamadas++
	return false
}

func TestAutenticacaoComUsuarioInexistenteAindaComparaSenha(t *testing.T) {
	comparador := &comparadorContador{}
	controlador := casosdeuso.NovoControladorCadastro(
		usuariosInexistentes{}, newTestStore(), &geradorSequencial{}, comparador,
		relogioFixo{agora: time.Now()},
	)

	_, _ = controlador.Autenticar(context.Background(), "inexistente@teste.local", "senha")

	if comparador.chamadas != 1 {
		t.Fatalf("Comparar() chamadas = %d; esperado 1", comparador.chamadas)
	}
}

func cadastrarUsuario(
	t *testing.T,
	ctx context.Context,
	controlador *casosdeuso.ControladorCadastro,
	nome, cpf, email string,
) cadastros.Usuario {
	t.Helper()
	usuario, err := controlador.CadastrarUsuario(ctx, casosdeuso.EntradaCadastro{
		Nome: nome, CPF: cpf, Email: email, Senha: "senha-segura",
		Endereco: cadastros.Endereco{
			CEP: "49000000", Logradouro: "Rua de Teste", Numero: "10",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "SE",
		},
	})
	if err != nil {
		t.Fatalf("CadastrarUsuario(%q) erro = %v", email, err)
	}
	return usuario
}
