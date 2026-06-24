package casosdeuso

import (
	"context"
	"strings"

	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/interacao"
)

const tamanhoMaximoMensagem = 4000

// ConversaDetalhada projeta a conversa de um pedido para a apresentacao: alem das mensagens,
// expoe os participantes para que a camada web rotule remetentes e resolva o interlocutor.
type ConversaDetalhada struct {
	IDPedido    string               `json:"id_pedido"`
	IDConversa  string               `json:"id_conversa"`
	IDComprador string               `json:"id_comprador"`
	IDVendedor  string               `json:"id_vendedor"`
	Mensagens   []interacao.Mensagem `json:"mensagens"`
}

// Interlocutor devolve o id do outro participante em relacao ao usuario informado.
func (c ConversaDetalhada) Interlocutor(idUsuario string) string {
	if idUsuario == c.IDComprador {
		return c.IDVendedor
	}
	return c.IDComprador
}

// ControladorConversas coordena o chat por pedido entre comprador e vendedor.
type ControladorConversas struct {
	conversas    OperacoesConversas
	notificacoes RegistroNotificacoes
	ids          GeradorID
	relogio      Relogio
}

func NovoControladorConversas(
	conversas OperacoesConversas,
	notificacoes RegistroNotificacoes,
	ids GeradorID,
	relogio Relogio,
) *ControladorConversas {
	return &ControladorConversas{conversas: conversas, notificacoes: notificacoes, ids: ids, relogio: relogio}
}

// autorizar garante que o usuario participa do pedido e devolve os participantes.
func (c *ControladorConversas) autorizar(
	ctx context.Context,
	idUsuario, idPedido string,
) (idComprador, idVendedor string, err error) {
	idComprador, idVendedor, err = c.conversas.BuscarParticipantesPedido(ctx, idPedido)
	if err != nil {
		return "", "", err
	}
	if idUsuario != idComprador && idUsuario != idVendedor {
		return "", "", common.ErrNaoPermitido
	}
	return idComprador, idVendedor, nil
}

// Abrir devolve (criando se necessario) a conversa do pedido para um participante.
func (c *ControladorConversas) Abrir(
	ctx context.Context,
	idUsuario, idPedido string,
) (ConversaDetalhada, error) {
	idComprador, idVendedor, err := c.autorizar(ctx, idUsuario, idPedido)
	if err != nil {
		return ConversaDetalhada{}, err
	}
	idConversa, err := c.conversas.ObterOuCriarConversa(ctx, c.ids.Novo(), idPedido, c.relogio.Agora())
	if err != nil {
		return ConversaDetalhada{}, err
	}
	mensagens, err := c.conversas.ListarMensagens(ctx, idConversa)
	if err != nil {
		return ConversaDetalhada{}, err
	}
	if mensagens == nil {
		mensagens = []interacao.Mensagem{}
	}
	return ConversaDetalhada{
		IDPedido: idPedido, IDConversa: idConversa,
		IDComprador: idComprador, IDVendedor: idVendedor, Mensagens: mensagens,
	}, nil
}

// Enviar valida e registra uma mensagem do participante e notifica o interlocutor.
func (c *ControladorConversas) Enviar(
	ctx context.Context,
	idUsuario, idPedido, conteudo string,
) error {
	idComprador, idVendedor, err := c.autorizar(ctx, idUsuario, idPedido)
	if err != nil {
		return err
	}
	conteudo = strings.TrimSpace(conteudo)
	if conteudo == "" || len([]rune(conteudo)) > tamanhoMaximoMensagem {
		return common.NovaValidacao(map[string]string{
			"conteudo": "Escreva uma mensagem de 1 a 4000 caracteres.",
		})
	}
	idConversa, err := c.conversas.ObterOuCriarConversa(ctx, c.ids.Novo(), idPedido, c.relogio.Agora())
	if err != nil {
		return err
	}
	if err := c.conversas.CriarMensagem(ctx, interacao.Mensagem{
		ID:                 c.ids.Novo(),
		IDConversa:         idConversa,
		IDUsuarioRemetente: idUsuario,
		Conteudo:           conteudo,
		CriadaEm:           c.relogio.Agora(),
	}); err != nil {
		return err
	}

	destinatario := idVendedor
	if idUsuario == idVendedor {
		destinatario = idComprador
	}
	if c.notificacoes != nil {
		_ = c.notificacoes.CriarNotificacao(ctx, interacao.Notificacao{
			ID:        c.ids.Novo(),
			IDUsuario: destinatario,
			Tipo:      interacao.NotificacaoMensagemRecebida,
			Conteudo:  "Você recebeu uma nova mensagem sobre um pedido.",
			IDPedido:  idPedido,
			CriadaEm:  c.relogio.Agora(),
		})
	}
	return nil
}
