package casosdeuso

import (
	"context"
	"fmt"

	"reveste/apps/api/internal/common"
)

// ControladorVendedor coordena o ciclo de vida da conta de vendedor que extrapola o
// cadastro basico. Hoje cobre a reativacao apos o bloqueio por itens nao enviados.
type ControladorVendedor struct {
	vendedores OperacoesReativacao
	pagamentos ProcessadorPagamento
	relogio    Relogio
	taxa       int64
}

func NovoControladorVendedor(
	vendedores OperacoesReativacao,
	pagamentos ProcessadorPagamento,
	relogio Relogio,
	taxaReativacaoCentavos int64,
) *ControladorVendedor {
	return &ControladorVendedor{
		vendedores: vendedores,
		pagamentos: pagamentos,
		relogio:    relogio,
		taxa:       taxaReativacaoCentavos,
	}
}

// TaxaReativacaoCentavos devolve o valor cobrado para reativar a conta.
func (c *ControladorVendedor) TaxaReativacaoCentavos() int64 {
	return c.taxa
}

// Reativar cobra a taxa de reativacao (provedor simulado no MVP) e, com a aprovacao,
// desbloqueia o vendedor e zera o contador de itens nao enviados.
//
// A cobranca antecede o desbloqueio. A chave de idempotencia inclui o contador de itens
// nao enviados do episodio atual: como a reativacao zera esse contador, uma repeticao da
// mesma intencao (ex.: falha apos cobrar, antes de desbloquear) reaproveita a cobranca no
// provedor sem duplicar o valor, enquanto um bloqueio futuro gera uma chave diferente.
func (c *ControladorVendedor) Reativar(ctx context.Context, idVendedor string) error {
	usuario, err := c.vendedores.BuscarUsuarioPorID(ctx, idVendedor)
	if err != nil {
		return err
	}
	if !usuario.BloqueadoParaVendas {
		return common.ErrTransicaoInvalida
	}

	chave := fmt.Sprintf("reativacao-%s-%d", idVendedor, usuario.ItensNaoEnviados)
	resultado, err := c.pagamentos.Processar(ctx, SolicitacaoPagamento{
		IDCompra:          chave,
		ValorCentavos:     c.taxa,
		ChaveIdempotencia: chave,
	})
	if err != nil {
		return err
	}
	if !resultado.Aprovado {
		return common.ErrPagamentoRecusado
	}

	if _, err := c.vendedores.ReativarVendedor(ctx, idVendedor, c.relogio.Agora()); err != nil {
		return err
	}
	return nil
}
