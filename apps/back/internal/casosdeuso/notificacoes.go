package casosdeuso

import (
	"context"

	"reveste/apps/back/internal/dominio/interacao"
)

// limiteCaixaNotificacoes limita a caixa de entrada exibida; o MVP nao pagina notificacoes.
const limiteCaixaNotificacoes = 50

// ControladorNotificacoes coordena a caixa de entrada de notificacoes do usuario: leitura,
// contagem de nao lidas e marcacao de leitura. A escrita das notificacoes e feita pelos
// casos de uso de origem (pedidos, chat) pela porta RegistroNotificacoes.
type ControladorNotificacoes struct {
	notificacoes OperacoesNotificacoes
	relogio      Relogio
}

func NovoControladorNotificacoes(
	notificacoes OperacoesNotificacoes,
	relogio Relogio,
) *ControladorNotificacoes {
	return &ControladorNotificacoes{notificacoes: notificacoes, relogio: relogio}
}

// Listar devolve as notificacoes mais recentes do usuario (vazio em vez de nil).
func (c *ControladorNotificacoes) Listar(
	ctx context.Context,
	idUsuario string,
) ([]interacao.Notificacao, error) {
	notificacoes, err := c.notificacoes.ListarNotificacoes(ctx, idUsuario, limiteCaixaNotificacoes)
	if err != nil {
		return nil, err
	}
	if notificacoes == nil {
		notificacoes = []interacao.Notificacao{}
	}
	return notificacoes, nil
}

// ContarNaoLidas devolve quantas notificacoes ainda nao foram lidas, para o indicador do
// cabecalho. Erros sao propagados; a apresentacao decide como degradar.
func (c *ControladorNotificacoes) ContarNaoLidas(
	ctx context.Context,
	idUsuario string,
) (int, error) {
	return c.notificacoes.ContarNotificacoesNaoLidas(ctx, idUsuario)
}

// MarcarTodasLidas marca todas as notificacoes nao lidas do usuario como lidas.
func (c *ControladorNotificacoes) MarcarTodasLidas(
	ctx context.Context,
	idUsuario string,
) error {
	return c.notificacoes.MarcarNotificacoesLidas(ctx, idUsuario, c.relogio.Agora())
}

func (c *ControladorNotificacoes) Remover(ctx context.Context, idUsuario, idNotificacao string) error {
	return c.notificacoes.RemoverNotificacao(ctx, idUsuario, idNotificacao)
}

func (c *ControladorNotificacoes) Limpar(ctx context.Context, idUsuario string) error {
	return c.notificacoes.LimparNotificacoes(ctx, idUsuario)
}
