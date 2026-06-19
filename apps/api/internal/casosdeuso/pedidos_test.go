package casosdeuso_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/compras"
	"reveste/apps/api/internal/dominio/interacao"
)

type pedidosFake struct {
	pedido          compras.Pedido
	pedidoErr       error
	enviado         bool
	enviadoRastreio string
	avaliacao       interacao.Avaliacao
	avaliacaoErr    error
}

func (f *pedidosFake) ListarPedidosDoVendedor(context.Context, string) ([]compras.Pedido, error) {
	return nil, nil
}

func (f *pedidosFake) BuscarPedidoDoComprador(context.Context, string, string) (compras.Pedido, error) {
	return f.pedido, f.pedidoErr
}

func (f *pedidosFake) BuscarPedidoDoVendedor(context.Context, string, string) (compras.Pedido, error) {
	return f.pedido, f.pedidoErr
}

func (f *pedidosFake) BuscarAvaliacaoDoPedido(context.Context, string) (interacao.Avaliacao, error) {
	if f.avaliacao.ID == "" {
		return interacao.Avaliacao{}, common.ErrNaoEncontrado
	}
	return f.avaliacao, nil
}

func (f *pedidosFake) MarcarPedidoEnviado(_ context.Context, _, _, _, rastreio string, _ time.Time) error {
	f.enviado = true
	f.enviadoRastreio = rastreio
	return nil
}

func (f *pedidosFake) ConfirmarRecebimentoPedido(context.Context, string, string, time.Time) error {
	return nil
}

func (f *pedidosFake) RegistrarAvaliacao(_ context.Context, avaliacao interacao.Avaliacao) error {
	f.avaliacao = avaliacao
	return f.avaliacaoErr
}

func (f *pedidosFake) ProcessarItensVencidos(context.Context, time.Time, int) (int, error) {
	return 0, nil
}

func (f *pedidosFake) MediaAvaliacoesVendedor(context.Context, string) (casosdeuso.MediaAvaliacoes, error) {
	return casosdeuso.MediaAvaliacoes{}, nil
}

func novoPedidos(fake *pedidosFake) *casosdeuso.ControladorPedidos {
	return casosdeuso.NovoControladorPedidos(
		fake, nil, &geradorSequencial{},
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
}

func novoPedidosComNotificacoes(fake *pedidosFake, registro casosdeuso.RegistroNotificacoes) *casosdeuso.ControladorPedidos {
	return casosdeuso.NovoControladorPedidos(
		fake, registro, &geradorSequencial{},
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
}

func TestMarcarEnviadoExigeRastreio(t *testing.T) {
	fake := &pedidosFake{}
	err := novoPedidos(fake).MarcarEnviado(context.Background(), "vendedor-1", "pedido-1", casosdeuso.EntradaEnvio{})
	var validacao common.ErroValidacao
	if !errors.As(err, &validacao) || validacao.Campos["codigo_rastreio"] == "" {
		t.Fatalf("erro = %v; esperada validação de codigo_rastreio", err)
	}
	if fake.enviado {
		t.Fatal("não deveria ter chamado o storage sem código de rastreio")
	}
}

func TestMarcarEnviadoRepassaRastreio(t *testing.T) {
	fake := &pedidosFake{}
	err := novoPedidos(fake).MarcarEnviado(context.Background(), "vendedor-1", "pedido-1", casosdeuso.EntradaEnvio{
		CodigoRastreio: "BR123",
	})
	if err != nil {
		t.Fatalf("MarcarEnviado() erro = %v", err)
	}
	if !fake.enviado || fake.enviadoRastreio != "BR123" {
		t.Fatalf("rastreio não repassado: %+v", fake)
	}
}

func TestAvaliacaoDoPedidoIndicaSeJaFoiAvaliado(t *testing.T) {
	semAvaliacao := &pedidosFake{}
	if _, existe, err := novoPedidos(semAvaliacao).AvaliacaoDoPedido(context.Background(), "pedido-1"); err != nil || existe {
		t.Fatalf("sem avaliação: existe=%v err=%v; esperado existe=false sem erro", existe, err)
	}

	comAvaliacao := &pedidosFake{avaliacao: interacao.Avaliacao{ID: "av-1", Nota: 5}}
	avaliacao, existe, err := novoPedidos(comAvaliacao).AvaliacaoDoPedido(context.Background(), "pedido-1")
	if err != nil || !existe || avaliacao.Nota != 5 {
		t.Fatalf("com avaliação: avaliacao=%+v existe=%v err=%v", avaliacao, existe, err)
	}
}

func TestAvaliarExigePedidoFinalizado(t *testing.T) {
	fake := &pedidosFake{pedido: compras.Pedido{
		ID: "pedido-1", IDComprador: "comprador-1", IDVendedor: "vendedor-1",
		Status: compras.StatusPedidoAguardandoEntrega,
	}}
	err := novoPedidos(fake).Avaliar(context.Background(), "comprador-1", "pedido-1", 5, "ótimo")
	if !errors.Is(err, common.ErrTransicaoInvalida) {
		t.Fatalf("erro = %v; esperado ErrTransicaoInvalida", err)
	}
}

func TestAvaliarRejeitaNotaForaDoIntervalo(t *testing.T) {
	fake := &pedidosFake{pedido: compras.Pedido{
		ID: "pedido-1", IDComprador: "comprador-1", IDVendedor: "vendedor-1",
		Status: compras.StatusPedidoFinalizado,
	}}
	err := novoPedidos(fake).Avaliar(context.Background(), "comprador-1", "pedido-1", 6, "")
	var validacao common.ErroValidacao
	if !errors.As(err, &validacao) || validacao.Campos["nota"] == "" {
		t.Fatalf("erro = %v; esperada validação de nota", err)
	}
}

func TestAvaliarRegistraComAutorEAvaliadoCorretos(t *testing.T) {
	fake := &pedidosFake{pedido: compras.Pedido{
		ID: "pedido-1", IDComprador: "comprador-1", IDVendedor: "vendedor-1",
		Status: compras.StatusPedidoFinalizado,
	}}
	if err := novoPedidos(fake).Avaliar(context.Background(), "comprador-1", "pedido-1", 5, "  recomendo  "); err != nil {
		t.Fatalf("Avaliar() erro = %v", err)
	}
	if fake.avaliacao.IDUsuarioAutor != "comprador-1" || fake.avaliacao.IDUsuarioAvaliado != "vendedor-1" {
		t.Fatalf("avaliação com autor/avaliado errados: %+v", fake.avaliacao)
	}
	if fake.avaliacao.Nota != 5 || fake.avaliacao.Comentario != "recomendo" {
		t.Fatalf("avaliação não normalizada: %+v", fake.avaliacao)
	}
}
