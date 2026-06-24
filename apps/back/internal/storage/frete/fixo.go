// Package frete contem adaptadores para provedores de cotacao de frete.
package frete

import (
	"context"

	"reveste/apps/back/internal/casosdeuso"
)

const provedorFixo = "fixo"

// Fixo e um CotadorFrete que devolve sempre o mesmo valor, independente de origem, destino
// ou dimensoes. Serve de contingencia quando nao ha integracao real configurada (sem token
// do Melhor Envio), mantendo o checkout funcional. A porta CotadorFrete permite substitui-lo.
type Fixo struct {
	valorCentavos int64
}

func NovoFixo(valorCentavos int64) *Fixo {
	return &Fixo{valorCentavos: valorCentavos}
}

func (f *Fixo) Cotar(
	_ context.Context,
	_, _ string,
	_ []casosdeuso.ItemFrete,
) (casosdeuso.CotacaoFrete, error) {
	return casosdeuso.CotacaoFrete{
		ValorCentavos: f.valorCentavos,
		Provedor:      provedorFixo,
		Servico:       "Frete padrão",
	}, nil
}
