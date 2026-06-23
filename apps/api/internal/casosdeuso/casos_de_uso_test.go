package casosdeuso_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
)

const hostBlobTeste = "reveste-test.public.blob.vercel-storage.com"

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
		hostBlobTeste,
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
		Categoria: anuncios.CategoriaCasacos, Tamanho: "m", Cor: "Azul",
		EstadoConservacao: anuncios.EstadoSeminovo, PrecoCentavos: 12_000,
		PesoGramas: 800, AlturaCm: 5, LarguraCm: 30, ComprimentoCm: 40,
		URLsFotos: []string{"https://reveste-test.public.blob.vercel-storage.com/1.jpg", "https://reveste-test.public.blob.vercel-storage.com/2.jpg"},
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

func TestCarrinhoMantemItemIndisponivelSemSomarAoTotal(t *testing.T) {
	store := newTestStore()
	store.anuncios["anuncio-indisponivel"] = anuncios.Anuncio{
		ID: "anuncio-indisponivel", IDVendedor: "vendedor-1",
		Status: anuncios.StatusAnuncioVendido, PrecoCentavos: 9_000,
	}
	store.carrinhoPorUsuario["comprador-1"] = compras.Carrinho{
		ID: "carrinho-1", IDUsuario: "comprador-1",
		IDsAnuncios: []string{"anuncio-indisponivel"},
	}
	controlador := casosdeuso.NovoControladorCarrinho(
		store, store, &geradorSequencial{}, relogioFixo{agora: time.Now()},
	)

	carrinho, err := controlador.ObterCarrinho(context.Background(), "comprador-1")
	if err != nil {
		t.Fatalf("ObterCarrinho() erro = %v", err)
	}
	if len(carrinho.Anuncios) != 1 {
		t.Fatalf("anuncios = %d; esperado 1", len(carrinho.Anuncios))
	}
	if carrinho.TotalCentavos != 0 {
		t.Fatalf("total = %d; esperado 0", carrinho.TotalCentavos)
	}
}

func TestCatalogoExcluiAnunciosPropriosEPerfilOsInclui(t *testing.T) {
	store := newTestStore()
	controlador := casosdeuso.NovoControladorAnuncio(
		store, store, &geradorSequencial{}, relogioFixo{agora: time.Now()},
		hostBlobTeste,
	)
	store.anuncios["proprio"] = anuncios.Anuncio{
		ID: "proprio", IDVendedor: "usuario-1", Status: anuncios.StatusAnuncioDisponivel,
	}
	store.anuncios["outro"] = anuncios.Anuncio{
		ID: "outro", IDVendedor: "usuario-2", Status: anuncios.StatusAnuncioDisponivel,
	}

	catalogo, err := controlador.ListarAnuncios(context.Background(), casosdeuso.FiltroAnuncios{
		ExcluirVendedor: "usuario-1",
		Limite:          20,
	})
	if err != nil {
		t.Fatalf("ListarAnuncios() erro = %v", err)
	}
	if len(catalogo) != 1 || catalogo[0].ID != "outro" {
		t.Fatalf("catalogo inesperado: %+v", catalogo)
	}

	meusAnuncios, err := controlador.ListarAnunciosDoVendedor(context.Background(), "usuario-1")
	if err != nil {
		t.Fatalf("ListarAnunciosDoVendedor() erro = %v", err)
	}
	if len(meusAnuncios) != 1 || meusAnuncios[0].ID != "proprio" {
		t.Fatalf("anuncios do perfil inesperados: %+v", meusAnuncios)
	}
}

func TestVendedorBloqueadoNaoPodeCriarAnuncio(t *testing.T) {
	store := newTestStore()
	store.usuarios["vendedor-bloqueado"] = cadastros.Usuario{
		ID: "vendedor-bloqueado", BloqueadoParaVendas: true,
	}
	controlador := casosdeuso.NovoControladorAnuncio(
		store, store, &geradorSequencial{}, relogioFixo{agora: time.Now()},
		hostBlobTeste,
	)

	_, err := controlador.CriarAnuncio(
		context.Background(),
		"vendedor-bloqueado",
		casosdeuso.EntradaAnuncio{
			Titulo: "Peça válida", Descricao: "Descrição suficientemente detalhada",
			Categoria: anuncios.CategoriaCasacos, Tamanho: "M", Cor: "verde",
			EstadoConservacao: anuncios.EstadoSeminovo, PrecoCentavos: 10_000,
			URLsFotos: []string{
				"https://reveste-test.public.blob.vercel-storage.com/1.jpg",
				"https://reveste-test.public.blob.vercel-storage.com/2.jpg",
			},
		},
	)

	if err != common.ErrVendedorBloqueado {
		t.Fatalf("CriarAnuncio() erro = %v; esperado %v", err, common.ErrVendedorBloqueado)
	}
}

func TestCriacaoRejeitaFotoDeOutroBlobStore(t *testing.T) {
	store := newTestStore()
	store.usuarios["vendedor-1"] = cadastros.Usuario{ID: "vendedor-1"}
	controlador := casosdeuso.NovoControladorAnuncio(
		store, store, &geradorSequencial{}, relogioFixo{agora: time.Now()},
		hostBlobTeste,
	)

	_, err := controlador.CriarAnuncio(context.Background(), "vendedor-1", casosdeuso.EntradaAnuncio{
		Titulo: "Peça válida", Descricao: "Descrição suficientemente detalhada",
		Categoria: anuncios.CategoriaCasacos, Tamanho: "M", Cor: "verde",
		EstadoConservacao: anuncios.EstadoSeminovo, PrecoCentavos: 10_000,
		URLsFotos: []string{
			"https://outro-store.public.blob.vercel-storage.com/1.jpg",
			"https://outro-store.public.blob.vercel-storage.com/2.jpg",
		},
	})
	var validacao common.ErroValidacao
	if !errors.As(err, &validacao) || validacao.Campos["fotos"] == "" {
		t.Fatalf("erro = %v; esperada validacao de fotos", err)
	}
}

func TestAtualizarPerfilPreservaDadosPrivadosEAtualizaEndereco(t *testing.T) {
	store := newTestStore()
	ids := &geradorSequencial{}
	controlador := casosdeuso.NovoControladorCadastro(
		store, store, ids, common.ProcessadorPBKDF2{Iteracoes: 100_000},
		relogioFixo{agora: time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)},
	)
	usuario := cadastrarUsuario(
		t, context.Background(), controlador,
		"Nome Original", "52998224725", "original@teste.local",
	)

	atualizado, err := controlador.AtualizarPerfil(
		context.Background(),
		usuario.ID,
		casosdeuso.EntradaAtualizacaoPerfil{
			Nome: "  Nome Atualizado  ", Email: "NOVO@TESTE.LOCAL", Telefone: " 79999999999 ",
			Endereco: cadastros.Endereco{
				CEP: "49010-000", Logradouro: "Rua Nova", Numero: "20",
				Complemento: "Apto 3", Bairro: "Centro", Cidade: "Aracaju", Estado: "se",
			},
		},
	)
	if err != nil {
		t.Fatalf("AtualizarPerfil() erro = %v", err)
	}
	if atualizado.Nome != "Nome Atualizado" || atualizado.Email != "novo@teste.local" {
		t.Fatalf("perfil nao normalizado: %+v", atualizado)
	}
	if atualizado.CPF != usuario.CPF || atualizado.HashSenha != usuario.HashSenha {
		t.Fatal("CPF ou hash da senha foram alterados")
	}
	if atualizado.EnderecoPrincipal.CEP != "49010000" || atualizado.EnderecoPrincipal.Estado != "SE" {
		t.Fatalf("endereco nao normalizado: %+v", atualizado.EnderecoPrincipal)
	}
}

func TestGerenciamentoDeAnuncioExigeProprietarioEDisponibilidade(t *testing.T) {
	store := newTestStore()
	store.usuarios["vendedor-1"] = cadastros.Usuario{
		ID: "vendedor-1", Nome: "Vendedor", CriadoEm: time.Now(),
		EnderecoPrincipal: cadastros.Endereco{Cidade: "Aracaju", Estado: "SE"},
	}
	store.anuncios["anuncio-1"] = anuncios.Anuncio{
		ID: "anuncio-1", IDVendedor: "vendedor-1", Titulo: "Titulo antigo",
		Descricao: "Descricao antiga valida", Categoria: anuncios.CategoriaCasacos,
		Tamanho: "M", Cor: "azul", EstadoConservacao: anuncios.EstadoSeminovo,
		PrecoCentavos: 10_000, Status: anuncios.StatusAnuncioDisponivel,
		Fotos: []anuncios.Foto{
			{ID: "foto-1", URL: "https://reveste-test.public.blob.vercel-storage.com/1.jpg"},
			{ID: "foto-2", URL: "https://reveste-test.public.blob.vercel-storage.com/2.jpg"},
		},
	}
	controlador := casosdeuso.NovoControladorAnuncio(
		store, store, &geradorSequencial{}, relogioFixo{agora: time.Now()},
		hostBlobTeste,
	)
	entrada := casosdeuso.EntradaAnuncio{
		Titulo: "Titulo atualizado", Descricao: "Descricao atualizada e valida",
		Categoria: anuncios.CategoriaCamisetas, Tamanho: "g", Cor: "Branco",
		EstadoConservacao: anuncios.EstadoUsado, PrecoCentavos: 12_000,
		PesoGramas: 400, AlturaCm: 3, LarguraCm: 25, ComprimentoCm: 35,
		URLsFotos: []string{"https://reveste-test.public.blob.vercel-storage.com/3.jpg", "https://reveste-test.public.blob.vercel-storage.com/4.jpg"},
	}

	if _, err := controlador.AtualizarAnuncio(
		context.Background(), "outro-vendedor", "anuncio-1", entrada,
	); err != common.ErrNaoPermitido {
		t.Fatalf("edicao por terceiro erro = %v; esperado ErrNaoPermitido", err)
	}
	atualizado, err := controlador.AtualizarAnuncio(
		context.Background(), "vendedor-1", "anuncio-1", entrada,
	)
	if err != nil {
		t.Fatalf("AtualizarAnuncio() erro = %v", err)
	}
	if atualizado.Titulo != "Titulo atualizado" || atualizado.Tamanho != "G" {
		t.Fatalf("anuncio nao atualizado: %+v", atualizado)
	}
	if err := controlador.ExcluirAnuncio(
		context.Background(), "vendedor-1", "anuncio-1",
	); err != nil {
		t.Fatalf("ExcluirAnuncio() erro = %v", err)
	}
	lista, err := controlador.ListarAnunciosDoVendedor(context.Background(), "vendedor-1")
	if err != nil {
		t.Fatalf("ListarAnunciosDoVendedor() erro = %v", err)
	}
	if len(lista) != 0 {
		t.Fatalf("anuncio excluido ainda listado: %+v", lista)
	}
}

func TestPerfilPublicoVendedorNaoExpoeDadosPrivados(t *testing.T) {
	store := newTestStore()
	store.usuarios["vendedor-1"] = cadastros.Usuario{
		ID: "vendedor-1", Nome: "Vendedora Teste", Email: "privado@teste.local",
		Telefone: "79999999999", CPF: "52998224725",
		EnderecoPrincipal: cadastros.Endereco{
			Logradouro: "Rua Privada", Numero: "10", Cidade: "Aracaju", Estado: "SE",
		},
		CriadoEm: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	controlador := casosdeuso.NovoControladorAnuncio(
		store, store, &geradorSequencial{}, relogioFixo{agora: time.Now()},
		hostBlobTeste,
	)

	perfil, err := controlador.ObterPerfilPublicoVendedor(context.Background(), "vendedor-1")
	if err != nil {
		t.Fatalf("ObterPerfilPublicoVendedor() erro = %v", err)
	}
	if perfil.Vendedor.Nome != "Vendedora Teste" ||
		perfil.Vendedor.Cidade != "Aracaju" ||
		perfil.Vendedor.Estado != "SE" {
		t.Fatalf("perfil publico inesperado: %+v", perfil.Vendedor)
	}
}

type usuariosInexistentes struct{}

func (usuariosInexistentes) CriarUsuario(context.Context, cadastros.Usuario) error {
	return nil
}

func (usuariosInexistentes) AtualizarUsuario(context.Context, cadastros.Usuario) error {
	return common.ErrNaoEncontrado
}

func (usuariosInexistentes) BuscarUsuarioPorID(context.Context, string) (cadastros.Usuario, error) {
	return cadastros.Usuario{}, common.ErrNaoEncontrado
}

func (usuariosInexistentes) BuscarUsuarioPorEmailOuCPF(context.Context, string) (cadastros.Usuario, error) {
	return cadastros.Usuario{}, common.ErrNaoEncontrado
}

func (usuariosInexistentes) ListarEnderecos(context.Context, string) ([]cadastros.Endereco, error) {
	return nil, common.ErrNaoEncontrado
}

func (usuariosInexistentes) BuscarEndereco(context.Context, string, string) (cadastros.Endereco, error) {
	return cadastros.Endereco{}, common.ErrNaoEncontrado
}

func (usuariosInexistentes) AdicionarEndereco(context.Context, string, cadastros.Endereco, time.Time) error {
	return common.ErrNaoEncontrado
}

func (usuariosInexistentes) AtualizarEndereco(context.Context, string, string, cadastros.Endereco, time.Time) error {
	return common.ErrNaoEncontrado
}

func (usuariosInexistentes) RemoverEndereco(context.Context, string, string, time.Time) error {
	return common.ErrNaoEncontrado
}

func (usuariosInexistentes) DefinirEnderecoPrincipal(context.Context, string, string, time.Time) error {
	return common.ErrNaoEncontrado
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
