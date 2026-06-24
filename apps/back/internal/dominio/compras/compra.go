package compras

import (
	"time"

	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/anuncios"
	"reveste/apps/back/internal/dominio/cadastros"
)

type StatusCompra string

const (
	StatusCompraAguardandoPagamento StatusCompra = "aguardando_pagamento"
	StatusCompraAprovada            StatusCompra = "aprovada"
	StatusCompraRecusada            StatusCompra = "recusada"
	StatusCompraExpirada            StatusCompra = "expirada"
	StatusCompraCancelada           StatusCompra = "cancelada"
)

type Compra struct {
	ID                       string       `json:"id"`
	IDComprador              string       `json:"id_comprador"`
	Status                   StatusCompra `json:"status"`
	ValorTotalItensCentavos  int64        `json:"valor_total_itens_centavos"`
	ValorTotalFretesCentavos int64        `json:"valor_total_fretes_centavos"`
	ValorTaxaServicoCentavos int64        `json:"valor_taxa_servico_centavos"`
	ValorFinalPagoCentavos   int64        `json:"valor_final_pago_centavos"`
	ChaveIdempotencia        string       `json:"chave_idempotencia"`
	ExpiraEm                 time.Time    `json:"expira_em"`
	Pedidos                  []Pedido     `json:"pedidos"`
	CriadaEm                 time.Time    `json:"criada_em"`
}

func (c Compra) CalcularTotal() int64 {
	return c.ValorTotalItensCentavos + c.ValorTotalFretesCentavos + c.ValorTaxaServicoCentavos
}

func (c *Compra) AlterarStatus(status StatusCompra) error {
	if c.Status != StatusCompraAguardandoPagamento ||
		!statusCompraFinal(status) {
		return common.ErrTransicaoInvalida
	}
	c.Status = status
	return nil
}

func statusCompraFinal(status StatusCompra) bool {
	switch status {
	case StatusCompraAprovada, StatusCompraRecusada, StatusCompraExpirada, StatusCompraCancelada:
		return true
	default:
		return false
	}
}

type StatusPedido string

const (
	StatusPedidoCriado              StatusPedido = "criado"
	StatusPedidoAguardandoPagamento StatusPedido = "aguardando_pagamento"
	StatusPedidoCancelado           StatusPedido = "cancelado"
	StatusPedidoExpirado            StatusPedido = "expirado"
	StatusPedidoAguardandoEnvio     StatusPedido = "aguardando_envio"
	StatusPedidoAguardandoEntrega   StatusPedido = "aguardando_entrega"
	StatusPedidoFinalizado          StatusPedido = "finalizado"
)

type Pedido struct {
	ID                           string             `json:"id"`
	IDCompra                     string             `json:"id_compra"`
	IDComprador                  string             `json:"id_comprador"`
	IDVendedor                   string             `json:"id_vendedor"`
	Status                       StatusPedido       `json:"status"`
	ValorTotalItensCentavos      int64              `json:"valor_total_itens_centavos"`
	ValorFreteCentavos           int64              `json:"valor_frete_centavos"`
	TaxaServicoCentavos          int64              `json:"taxa_servico_centavos"`
	ValorLiquidoVendedorCentavos int64              `json:"valor_liquido_vendedor_centavos"`
	NomeDestinatario             string             `json:"nome_destinatario"`
	EnderecoEntrega              cadastros.Endereco `json:"endereco_entrega"`
	Itens                        []ItemPedido       `json:"itens"`
	Entrega                      *Entrega           `json:"entrega,omitempty"`
	CriadoEm                     time.Time          `json:"criado_em"`
	FinalizadoEm                 *time.Time         `json:"finalizado_em,omitempty"`
}

// TotalCompradorCentavos e o quanto o comprador paga por este pedido: itens mais frete.
func (p Pedido) TotalCompradorCentavos() int64 {
	return p.ValorTotalItensCentavos + p.ValorFreteCentavos
}

func (p *Pedido) AlterarStatus(status StatusPedido) error {
	if !transicaoPedidoValida(p.Status, status) {
		return common.ErrTransicaoInvalida
	}
	p.Status = status
	return nil
}

func transicaoPedidoValida(atual, proximo StatusPedido) bool {
	switch atual {
	case StatusPedidoCriado:
		return proximo == StatusPedidoAguardandoPagamento ||
			proximo == StatusPedidoCancelado
	case StatusPedidoAguardandoPagamento:
		return proximo == StatusPedidoAguardandoEnvio ||
			proximo == StatusPedidoCancelado ||
			proximo == StatusPedidoExpirado
	case StatusPedidoAguardandoEnvio:
		return proximo == StatusPedidoAguardandoEntrega ||
			proximo == StatusPedidoCancelado
	case StatusPedidoAguardandoEntrega:
		return proximo == StatusPedidoFinalizado
	default:
		return false
	}
}

type StatusItemPedido string

const (
	StatusItemAguardandoEnvio StatusItemPedido = "aguardando_envio"
	StatusItemEnviado         StatusItemPedido = "enviado"
	StatusItemNaoEnviado      StatusItemPedido = "nao_enviado"
	StatusItemRecebido        StatusItemPedido = "recebido"
	StatusItemSuspenso        StatusItemPedido = "suspenso"
)

type ItemPedido struct {
	ID                    string                     `json:"id"`
	IDPedido              string                     `json:"id_pedido"`
	IDAnuncio             string                     `json:"id_anuncio"`
	Status                StatusItemPedido           `json:"status"`
	Titulo                string                     `json:"titulo"`
	Categoria             string                     `json:"categoria"`
	Tamanho               string                     `json:"tamanho"`
	Cor                   string                     `json:"cor"`
	EstadoConservacao     anuncios.EstadoConservacao `json:"estado_conservacao"`
	ValorUnitarioCentavos int64                      `json:"valor_unitario_centavos"`
	TaxaServicoCentavos   int64                      `json:"taxa_servico_centavos"`
	PrazoEnvioEm          time.Time                  `json:"prazo_envio_em"`
	EnviadoEm             *time.Time                 `json:"enviado_em,omitempty"`
	RecebidoEm            *time.Time                 `json:"recebido_em,omitempty"`
}

func (i ItemPedido) CalcularTotal() int64 {
	return i.ValorUnitarioCentavos + i.TaxaServicoCentavos
}

func (i *ItemPedido) AlterarStatus(status StatusItemPedido) error {
	valida := (i.Status == StatusItemAguardandoEnvio &&
		(status == StatusItemEnviado || status == StatusItemNaoEnviado)) ||
		(i.Status == StatusItemEnviado && status == StatusItemRecebido) ||
		(i.Status == StatusItemNaoEnviado && status == StatusItemSuspenso)
	if !valida {
		return common.ErrTransicaoInvalida
	}
	i.Status = status
	return nil
}

type StatusPagamento string

const (
	StatusPagamentoPendente           StatusPagamento = "pendente"
	StatusPagamentoAprovado           StatusPagamento = "aprovado"
	StatusPagamentoRecusado           StatusPagamento = "recusado"
	StatusPagamentoReembolsadoParcial StatusPagamento = "reembolsado_parcial"
	StatusPagamentoReembolsado        StatusPagamento = "reembolsado"
)

type Pagamento struct {
	ID                   string          `json:"id"`
	IDCompra             string          `json:"id_compra"`
	Provedor             string          `json:"provedor"`
	IdentificadorExterno string          `json:"identificador_externo,omitempty"`
	Status               StatusPagamento `json:"status"`
	ValorCentavos        int64           `json:"valor_centavos"`
	ChaveIdempotencia    string          `json:"chave_idempotencia"`
	PagoEm               *time.Time      `json:"pago_em,omitempty"`
}

func (p *Pagamento) AlterarStatus(status StatusPagamento) error {
	valida := (p.Status == StatusPagamentoPendente &&
		(status == StatusPagamentoAprovado || status == StatusPagamentoRecusado)) ||
		(p.Status == StatusPagamentoAprovado &&
			(status == StatusPagamentoReembolsadoParcial || status == StatusPagamentoReembolsado)) ||
		(p.Status == StatusPagamentoReembolsadoParcial && status == StatusPagamentoReembolsado)
	if !valida {
		return common.ErrTransicaoInvalida
	}
	p.Status = status
	return nil
}

type StatusReembolso string

const (
	StatusReembolsoPendente   StatusReembolso = "pendente"
	StatusReembolsoProcessado StatusReembolso = "processado"
	StatusReembolsoFalhou     StatusReembolso = "falhou"
)

type Reembolso struct {
	ID                   string          `json:"id"`
	IDPagamento          string          `json:"id_pagamento"`
	IDItemPedido         string          `json:"id_item_pedido"`
	IdentificadorExterno string          `json:"identificador_externo,omitempty"`
	Status               StatusReembolso `json:"status"`
	ValorCentavos        int64           `json:"valor_centavos"`
	Motivo               string          `json:"motivo"`
	ChaveIdempotencia    string          `json:"chave_idempotencia"`
	ProcessadoEm         *time.Time      `json:"processado_em,omitempty"`
}

func (r *Reembolso) AlterarStatus(status StatusReembolso) error {
	if r.Status != StatusReembolsoPendente ||
		(status != StatusReembolsoProcessado && status != StatusReembolsoFalhou) {
		return common.ErrTransicaoInvalida
	}
	r.Status = status
	return nil
}

type StatusEntrega string

const (
	StatusEntregaAguardandoPostagem StatusEntrega = "aguardando_postagem"
	StatusEntregaPostado            StatusEntrega = "postado"
	StatusEntregaEmTransito         StatusEntrega = "em_transito"
	StatusEntregaEntregue           StatusEntrega = "entregue"
	StatusEntregaFalhou             StatusEntrega = "falhou"
)

type Entrega struct {
	ID                 string        `json:"id"`
	IDPedido           string        `json:"id_pedido"`
	Provedor           string        `json:"provedor,omitempty"`
	CodigoRastreio     string        `json:"codigo_rastreio,omitempty"`
	Status             StatusEntrega `json:"status"`
	ValorFreteCentavos int64         `json:"valor_frete_centavos"`
	PostadoEm          *time.Time    `json:"postado_em,omitempty"`
	EntregueEm         *time.Time    `json:"entregue_em,omitempty"`
}

func (e *Entrega) AlterarStatus(status StatusEntrega) error {
	valida := (e.Status == StatusEntregaAguardandoPostagem && status == StatusEntregaPostado) ||
		(e.Status == StatusEntregaPostado &&
			(status == StatusEntregaEmTransito || status == StatusEntregaEntregue || status == StatusEntregaFalhou)) ||
		(e.Status == StatusEntregaEmTransito &&
			(status == StatusEntregaEntregue || status == StatusEntregaFalhou))
	if !valida {
		return common.ErrTransicaoInvalida
	}
	e.Status = status
	return nil
}
