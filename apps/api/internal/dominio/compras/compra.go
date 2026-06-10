package compras

import (
	"time"

	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
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
	EnderecoEntrega              cadastros.Endereco `json:"endereco_entrega"`
	Itens                        []ItemPedido       `json:"itens"`
	Entrega                      *Entrega           `json:"entrega,omitempty"`
	CriadoEm                     time.Time          `json:"criado_em"`
	FinalizadoEm                 *time.Time         `json:"finalizado_em,omitempty"`
}

func (p *Pedido) AlterarStatus(status StatusPedido) {
	p.Status = status
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
	ProcessadoEm         *time.Time      `json:"processado_em,omitempty"`
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
