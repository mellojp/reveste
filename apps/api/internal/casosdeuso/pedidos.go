package casosdeuso

import (
	"context"
	"errors"
	"strings"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
	"reveste/apps/api/internal/dominio/interacao"
)

// ControladorPedidos coordena o ciclo de vida do pedido apos a compra: envio pelo
// vendedor, confirmacao de recebimento e avaliacao pelo comprador, e o bloqueio do
// vendedor por itens nao enviados.
type ControladorPedidos struct {
	pedidos OperacoesPedidos
	ids     GeradorID
	relogio Relogio
}

func NovoControladorPedidos(
	pedidos OperacoesPedidos,
	ids GeradorID,
	relogio Relogio,
) *ControladorPedidos {
	return &ControladorPedidos{pedidos: pedidos, ids: ids, relogio: relogio}
}

type EntradaEnvio struct {
	Provedor       string
	CodigoRastreio string
}

func (c *ControladorPedidos) ListarVendas(
	ctx context.Context,
	idVendedor string,
) ([]compras.Pedido, error) {
	pedidos, err := c.pedidos.ListarPedidosDoVendedor(ctx, idVendedor)
	if err != nil {
		return nil, err
	}
	if pedidos == nil {
		pedidos = []compras.Pedido{}
	}
	return pedidos, nil
}

// DetalharCompra devolve um pedido especifico do comprador.
func (c *ControladorPedidos) DetalharCompra(
	ctx context.Context,
	idComprador, idPedido string,
) (compras.Pedido, error) {
	return c.pedidos.BuscarPedidoDoComprador(ctx, idComprador, idPedido)
}

// DetalharVenda devolve um pedido especifico recebido pelo vendedor.
func (c *ControladorPedidos) DetalharVenda(
	ctx context.Context,
	idVendedor, idPedido string,
) (compras.Pedido, error) {
	return c.pedidos.BuscarPedidoDoVendedor(ctx, idVendedor, idPedido)
}

// AvaliacaoDoPedido devolve a avaliacao ja registrada e se ela existe. Erros diferentes de
// ErrNaoEncontrado sao propagados; ausencia de avaliacao nao e erro.
func (c *ControladorPedidos) AvaliacaoDoPedido(
	ctx context.Context,
	idPedido string,
) (interacao.Avaliacao, bool, error) {
	avaliacao, err := c.pedidos.BuscarAvaliacaoDoPedido(ctx, idPedido)
	if errors.Is(err, common.ErrNaoEncontrado) {
		return interacao.Avaliacao{}, false, nil
	}
	if err != nil {
		return interacao.Avaliacao{}, false, err
	}
	return avaliacao, true, nil
}

// MarcarEnviado registra a postagem de um pedido pelo vendedor.
func (c *ControladorPedidos) MarcarEnviado(
	ctx context.Context,
	idVendedor, idPedido string,
	entrada EntradaEnvio,
) error {
	rastreio := strings.TrimSpace(entrada.CodigoRastreio)
	if rastreio == "" {
		return common.NovaValidacao(map[string]string{
			"codigo_rastreio": "Informe o código de rastreio do envio.",
		})
	}
	provedor := strings.TrimSpace(entrada.Provedor)
	if provedor == "" {
		provedor = "Correios"
	}
	return c.pedidos.MarcarPedidoEnviado(ctx, idPedido, idVendedor, provedor, rastreio, c.relogio.Agora())
}

// ConfirmarRecebimento finaliza o pedido a pedido do comprador.
func (c *ControladorPedidos) ConfirmarRecebimento(
	ctx context.Context,
	idComprador, idPedido string,
) error {
	return c.pedidos.ConfirmarRecebimentoPedido(ctx, idPedido, idComprador, c.relogio.Agora())
}

// Avaliar registra a avaliacao do vendedor por um pedido finalizado do comprador.
func (c *ControladorPedidos) Avaliar(
	ctx context.Context,
	idComprador, idPedido string,
	nota int,
	comentario string,
) error {
	pedido, err := c.pedidos.BuscarPedidoDoComprador(ctx, idComprador, idPedido)
	if err != nil {
		return err
	}
	if pedido.Status != compras.StatusPedidoFinalizado {
		return common.ErrTransicaoInvalida
	}
	avaliacao := interacao.Avaliacao{
		ID:                c.ids.Novo(),
		IDPedido:          idPedido,
		IDUsuarioAutor:    idComprador,
		IDUsuarioAvaliado: pedido.IDVendedor,
		Nota:              nota,
		Comentario:        strings.TrimSpace(comentario),
		CriadaEm:          c.relogio.Agora(),
	}
	if !avaliacao.Valida() {
		return common.NovaValidacao(map[string]string{
			"nota": "A nota deve ser um número de 1 a 5.",
		})
	}
	return c.pedidos.RegistrarAvaliacao(ctx, avaliacao)
}

func (c *ControladorPedidos) MediaVendedor(
	ctx context.Context,
	idVendedor string,
) (MediaAvaliacoes, error) {
	return c.pedidos.MediaAvaliacoesVendedor(ctx, idVendedor)
}

// ProcessarPrazosEnvio marca itens vencidos como nao enviados e bloqueia vendedores que
// atingirem o limite. Deve ser acionado periodicamente (job/cron).
func (c *ControladorPedidos) ProcessarPrazosEnvio(ctx context.Context) (int, error) {
	return c.pedidos.ProcessarItensVencidos(ctx, c.relogio.Agora(), cadastros.LimiteItensNaoEnviados)
}
