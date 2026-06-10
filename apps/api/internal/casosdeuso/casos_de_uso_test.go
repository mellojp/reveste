package casosdeuso_test

import (
	"context"
	"testing"
	"time"

	casosdeusoanuncios "reveste/apps/api/internal/casosdeuso/anuncios"
	casosdeusocadastros "reveste/apps/api/internal/casosdeuso/cadastros"
	casosdeusocompras "reveste/apps/api/internal/casosdeuso/compras"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
)

type geradorSequencial struct {
	proximo int
}

func (g *geradorSequencial) Novo() string {
	g.proximo++
	return "id-" + time.Unix(int64(g.proximo), 0).UTC().Format("150405")
}

type relogioFixo struct {
	agora time.Time
}

func (r relogioFixo) Agora() time.Time {
	return r.agora
}

func TestFluxoCadastroAnuncioCarrinho(t *testing.T) {
	store := newTestStore()
	ids := &geradorSequencial{}
	cadastrosCU := casosdeusocadastros.NovoFluxoCadastro(
		store,
		store,
		ids,
		common.ProcessadorPBKDF2{Iteracoes: 100_000},
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
	anunciosCU := casosdeusoanuncios.NovoFluxoAnuncio(
		store, store, ids,
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
	comprasCU := casosdeusocompras.NovoFluxoCarrinho(
		store, store, ids,
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
	ctx := context.Background()

	vendedor := cadastrarUsuario(t, ctx, cadastrosCU, "Vendedor Teste", "52998224725", "vendedor@teste.local")
	comprador := cadastrarUsuario(t, ctx, cadastrosCU, "Comprador Teste", "11144477735", "comprador@teste.local")

	anuncio, err := anunciosCU.CriarAnuncio(ctx, vendedor.ID, casosdeusoanuncios.EntradaAnuncio{
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

func cadastrarUsuario(
	t *testing.T,
	ctx context.Context,
	fluxos *casosdeusocadastros.FluxoCadastro,
	nome, cpf, email string,
) cadastros.Usuario {
	t.Helper()
	usuario, err := fluxos.CadastrarUsuario(ctx, casosdeusocadastros.EntradaCadastro{
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
