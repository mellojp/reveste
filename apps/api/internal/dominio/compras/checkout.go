package compras

import (
	"sort"
	"time"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
)

// ItemCompravel e a projecao de um anuncio disponivel pronto para virar item de pedido.
type ItemCompravel struct {
	IDAnuncio         string
	IDVendedor        string
	Titulo            string
	Categoria         string
	Tamanho           string
	Cor               string
	EstadoConservacao anuncios.EstadoConservacao
	PrecoCentavos     int64
	PesoGramas        int
	AlturaCm          int
	LarguraCm         int
	ComprimentoCm     int
}

// PoliticaCobranca define como a taxa de servico e o frete sao calculados no checkout.
// A taxa de servico e a comissao da plataforma, descontada do repasse ao vendedor.
// O frete e cobrado do comprador, um valor fixo por pedido (por vendedor).
type PoliticaCobranca struct {
	TaxaServicoPercentual  int64
	FretePorPedidoCentavos int64
}

func (p PoliticaCobranca) taxaServico(valorItens int64) int64 {
	if p.TaxaServicoPercentual <= 0 {
		return 0
	}
	return valorItens * p.TaxaServicoPercentual / 100
}

// ParametrosCompra reune tudo que e necessario para montar uma compra de forma deterministica.
// IDs e timestamps sao fornecidos pelo caso de uso, mantendo o dominio livre de efeitos colaterais.
type ParametrosCompra struct {
	IDCompra          string
	IDComprador       string
	NomeDestinatario  string
	EnderecoEntrega   cadastros.Endereco
	Itens             []ItemCompravel
	Politica          PoliticaCobranca
	FretePorVendedor  map[string]int64
	ChaveIdempotencia string
	Agora             time.Time
	ExpiraEm          time.Time
	PrazoEnvio        time.Time
	GerarID           func() string
}

// MontarCompra agrupa os itens por vendedor, gera um pedido por vendedor com snapshot das
// pecas e calcula os totais. A compra nasce aguardando pagamento; os estados avancam apos
// a confirmacao do pagamento, pelas maquinas de estado de Compra e Pedido.
func MontarCompra(p ParametrosCompra) (Compra, error) {
	if len(p.Itens) == 0 {
		return Compra{}, common.ErrSemItensDisponiveis
	}

	itensPorVendedor := make(map[string][]ItemCompravel)
	ordemVendedores := make([]string, 0)
	for _, item := range p.Itens {
		if _, visto := itensPorVendedor[item.IDVendedor]; !visto {
			ordemVendedores = append(ordemVendedores, item.IDVendedor)
		}
		itensPorVendedor[item.IDVendedor] = append(itensPorVendedor[item.IDVendedor], item)
	}
	sort.Strings(ordemVendedores)

	compra := Compra{
		ID:                p.IDCompra,
		IDComprador:       p.IDComprador,
		Status:            StatusCompraAguardandoPagamento,
		ChaveIdempotencia: p.ChaveIdempotencia,
		ExpiraEm:          p.ExpiraEm,
		CriadaEm:          p.Agora,
	}

	for _, idVendedor := range ordemVendedores {
		itensVendedor := itensPorVendedor[idVendedor]
		pedido := Pedido{
			ID:                 p.GerarID(),
			IDCompra:           p.IDCompra,
			IDComprador:        p.IDComprador,
			IDVendedor:         idVendedor,
			Status:             StatusPedidoAguardandoPagamento,
			ValorFreteCentavos: p.FretePorVendedor[idVendedor],
			NomeDestinatario:   p.NomeDestinatario,
			EnderecoEntrega:    p.EnderecoEntrega,
			CriadoEm:           p.Agora,
		}
		for _, item := range itensVendedor {
			pedido.Itens = append(pedido.Itens, ItemPedido{
				ID:                    p.GerarID(),
				IDPedido:              pedido.ID,
				IDAnuncio:             item.IDAnuncio,
				Status:                StatusItemAguardandoEnvio,
				Titulo:                item.Titulo,
				Categoria:             item.Categoria,
				Tamanho:               item.Tamanho,
				Cor:                   item.Cor,
				EstadoConservacao:     item.EstadoConservacao,
				ValorUnitarioCentavos: item.PrecoCentavos,
				PrazoEnvioEm:          p.PrazoEnvio,
			})
			pedido.ValorTotalItensCentavos += item.PrecoCentavos
		}
		pedido.TaxaServicoCentavos = p.Politica.taxaServico(pedido.ValorTotalItensCentavos)
		pedido.ValorLiquidoVendedorCentavos =
			pedido.ValorTotalItensCentavos + pedido.ValorFreteCentavos - pedido.TaxaServicoCentavos
		// distribui a taxa de servico do pedido para os itens (snapshot por item).
		distribuirTaxaPorItem(&pedido)

		compra.Pedidos = append(compra.Pedidos, pedido)
		compra.ValorTotalItensCentavos += pedido.ValorTotalItensCentavos
		compra.ValorTotalFretesCentavos += pedido.ValorFreteCentavos
	}

	// A taxa de servico nao e cobrada do comprador: ela e descontada do repasse ao vendedor.
	compra.ValorTaxaServicoCentavos = 0
	compra.ValorFinalPagoCentavos = compra.CalcularTotal()
	return compra, nil
}

// distribuirTaxaPorItem rateia a taxa de servico do pedido entre seus itens, garantindo
// que a soma dos itens reproduza exatamente a taxa do pedido (o residuo vai para o primeiro).
func distribuirTaxaPorItem(pedido *Pedido) {
	if len(pedido.Itens) == 0 || pedido.ValorTotalItensCentavos == 0 {
		return
	}
	acumulado := int64(0)
	for indice := range pedido.Itens {
		parcela := pedido.TaxaServicoCentavos * pedido.Itens[indice].ValorUnitarioCentavos /
			pedido.ValorTotalItensCentavos
		pedido.Itens[indice].TaxaServicoCentavos = parcela
		acumulado += parcela
	}
	pedido.Itens[0].TaxaServicoCentavos += pedido.TaxaServicoCentavos - acumulado
}

// IDsAnuncios devolve, em ordem estavel, os anuncios envolvidos na compra.
func (c Compra) IDsAnuncios() []string {
	ids := make([]string, 0)
	for _, pedido := range c.Pedidos {
		for _, item := range pedido.Itens {
			ids = append(ids, item.IDAnuncio)
		}
	}
	return ids
}

// ChaveIdempotenciaCarrinho monta uma chave estavel a partir do comprador e dos anuncios.
// Reenviar o mesmo carrinho produz a mesma chave, permitindo um checkout idempotente.
func ChaveIdempotenciaCarrinho(idComprador string, idsAnuncios []string) string {
	ordenados := append([]string(nil), idsAnuncios...)
	sort.Strings(ordenados)
	return common.HashEstavel(append([]string{idComprador}, ordenados...)...)
}
