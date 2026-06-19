package casosdeuso_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/interacao"
)

type conversasFake struct {
	comprador, vendedor string
	pedidoErr           error
	idConversa          string
	mensagens           []interacao.Mensagem
}

func (f *conversasFake) BuscarParticipantesPedido(context.Context, string) (string, string, error) {
	return f.comprador, f.vendedor, f.pedidoErr
}

func (f *conversasFake) ObterOuCriarConversa(context.Context, string, string, time.Time) (string, error) {
	if f.idConversa == "" {
		f.idConversa = "conv-1"
	}
	return f.idConversa, nil
}

func (f *conversasFake) ListarMensagens(context.Context, string) ([]interacao.Mensagem, error) {
	return f.mensagens, nil
}

func (f *conversasFake) CriarMensagem(_ context.Context, m interacao.Mensagem) error {
	f.mensagens = append(f.mensagens, m)
	return nil
}

func novoConversas(fake *conversasFake, registro casosdeuso.RegistroNotificacoes) *casosdeuso.ControladorConversas {
	return casosdeuso.NovoControladorConversas(
		fake, registro, &geradorSequencial{},
		relogioFixo{agora: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)},
	)
}

func TestAbrirConversaRejeitaNaoParticipante(t *testing.T) {
	fake := &conversasFake{comprador: "c1", vendedor: "v1"}
	_, err := novoConversas(fake, nil).Abrir(context.Background(), "estranho", "pedido-1")
	if !errors.Is(err, common.ErrNaoPermitido) {
		t.Fatalf("erro = %v; esperado ErrNaoPermitido", err)
	}
}

func TestAbrirConversaParticipanteDevolveParticipantes(t *testing.T) {
	fake := &conversasFake{comprador: "c1", vendedor: "v1"}
	conversa, err := novoConversas(fake, nil).Abrir(context.Background(), "c1", "pedido-1")
	if err != nil {
		t.Fatalf("Abrir() erro = %v", err)
	}
	if conversa.IDComprador != "c1" || conversa.IDVendedor != "v1" {
		t.Fatalf("participantes incorretos: %+v", conversa)
	}
	if conversa.Interlocutor("c1") != "v1" {
		t.Fatalf("interlocutor de c1 deveria ser v1, veio %q", conversa.Interlocutor("c1"))
	}
}

func TestEnviarMensagemRejeitaConteudoVazio(t *testing.T) {
	fake := &conversasFake{comprador: "c1", vendedor: "v1"}
	err := novoConversas(fake, nil).Enviar(context.Background(), "c1", "pedido-1", "   ")
	var validacao common.ErroValidacao
	if !errors.As(err, &validacao) || validacao.Campos["conteudo"] == "" {
		t.Fatalf("erro = %v; esperada validação de conteudo", err)
	}
	if len(fake.mensagens) != 0 {
		t.Fatal("não deveria ter registrado mensagem vazia")
	}
}

func TestEnviarMensagemRegistraENotificaInterlocutor(t *testing.T) {
	fake := &conversasFake{comprador: "c1", vendedor: "v1"}
	store := newTestStore()
	if err := novoConversas(fake, store).Enviar(context.Background(), "c1", "pedido-1", "  olá, tudo bem?  "); err != nil {
		t.Fatalf("Enviar() erro = %v", err)
	}
	if len(fake.mensagens) != 1 || fake.mensagens[0].Conteudo != "olá, tudo bem?" {
		t.Fatalf("mensagem não registrada/normalizada: %+v", fake.mensagens)
	}
	notificacoes, _ := store.ListarNotificacoes(context.Background(), "v1", 10)
	if len(notificacoes) != 1 || notificacoes[0].Tipo != interacao.NotificacaoMensagemRecebida {
		t.Fatalf("interlocutor v1 deveria receber notificação de mensagem: %+v", notificacoes)
	}
	if notificacoes[0].IDPedido != "pedido-1" {
		t.Fatalf("notificação sem deep-link ao pedido: %+v", notificacoes[0])
	}
}
