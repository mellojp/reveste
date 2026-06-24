package casosdeuso_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/anuncios"
	"reveste/apps/back/internal/dominio/cadastros"
)

func novoCadastroEnderecos(store *Store) *casosdeuso.ControladorCadastro {
	return casosdeuso.NovoControladorCadastro(
		store, store, &geradorSequencial{},
		common.ProcessadorPBKDF2{Iteracoes: 100_000},
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
}

func enderecoValido() cadastros.Endereco {
	return cadastros.Endereco{
		CEP: "01310200", Logradouro: "Avenida Paulista", Numero: "1000",
		Bairro: "Bela Vista", Cidade: "São Paulo", Estado: "SP",
	}
}

func TestAdicionarEnderecoValidaCampos(t *testing.T) {
	controlador := novoCadastroEnderecos(newTestStore())
	_, err := controlador.AdicionarEndereco(context.Background(), "usuario-1", cadastros.Endereco{CEP: "1"})
	var validacao common.ErroValidacao
	if !errors.As(err, &validacao) || validacao.Campos["cep"] == "" {
		t.Fatalf("erro = %v; esperada validação de endereço", err)
	}
}

func TestAdicionarEnderecoGeraIDeNaoEhPrincipal(t *testing.T) {
	store := newTestStore()
	controlador := novoCadastroEnderecos(store)
	endereco, err := controlador.AdicionarEndereco(context.Background(), "usuario-1", enderecoValido())
	if err != nil {
		t.Fatalf("AdicionarEndereco() erro = %v", err)
	}
	if endereco.ID == "" || endereco.Principal {
		t.Fatalf("endereço deveria ter ID e não ser principal: %+v", endereco)
	}
}

func TestRemoverEnderecoPrincipalEhBloqueado(t *testing.T) {
	store := newTestStore()
	controlador := novoCadastroEnderecos(store)
	ctx := context.Background()

	endereco, _ := controlador.AdicionarEndereco(ctx, "usuario-1", enderecoValido())
	if err := controlador.DefinirEnderecoPrincipal(ctx, "usuario-1", endereco.ID); err != nil {
		t.Fatalf("DefinirEnderecoPrincipal() erro = %v", err)
	}
	err := controlador.RemoverEndereco(ctx, "usuario-1", endereco.ID)
	if !errors.Is(err, common.ErrNaoPermitido) {
		t.Fatalf("erro = %v; esperado ErrNaoPermitido ao remover o principal", err)
	}
}

func TestRemoverEnderecoNaoPrincipalFunciona(t *testing.T) {
	store := newTestStore()
	controlador := novoCadastroEnderecos(store)
	ctx := context.Background()

	endereco, _ := controlador.AdicionarEndereco(ctx, "usuario-1", enderecoValido())
	if err := controlador.RemoverEndereco(ctx, "usuario-1", endereco.ID); err != nil {
		t.Fatalf("RemoverEndereco() erro = %v", err)
	}
	enderecos, _ := controlador.ListarEnderecos(ctx, "usuario-1")
	if len(enderecos) != 0 {
		t.Fatalf("endereço não foi removido: %+v", enderecos)
	}
}

func TestCheckoutUsaEnderecoEscolhido(t *testing.T) {
	store := newTestStore()
	semearCheckout(store, "comprador-1", "a1", "vendedor-1", 10_000, anuncios.StatusAnuncioDisponivel)
	store.enderecosPorUsuario["comprador-1"] = []cadastros.Endereco{{
		ID: "endereco-sp", CEP: "01310200", Logradouro: "Avenida Paulista", Numero: "1000",
		Bairro: "Bela Vista", Cidade: "São Paulo", Estado: "SP",
	}}
	checkout := novoCheckout(store, pagamentoFake{aprovar: true})

	compra, err := checkout.FinalizarCompra(context.Background(), "comprador-1", "endereco-sp")
	if err != nil {
		t.Fatalf("FinalizarCompra() erro = %v", err)
	}
	entrega := compra.Pedidos[0].EnderecoEntrega
	if entrega.Cidade != "São Paulo" || entrega.Estado != "SP" {
		t.Fatalf("pedido não usou o endereço escolhido: %+v", entrega)
	}
}
