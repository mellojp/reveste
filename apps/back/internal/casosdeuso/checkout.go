package casosdeuso

import (
	"context"
	"fmt"
	"time"

	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/anuncios"
	"reveste/apps/back/internal/dominio/compras"
	"reveste/apps/back/internal/dominio/interacao"
)

const (
	validadeCompra = 30 * time.Minute
	prazoEnvioItem = 5 * 24 * time.Hour
)

// ControladorCheckout coordena a finalizacao da compra em fases: reserva e persiste a
// intencao, processa o pagamento e confirma ou desfaz a intencao.
type ControladorCheckout struct {
	usuarios     OperacoesUsuarios
	anuncios     OperacoesAnuncios
	carrinhos    OperacoesCarrinhos
	checkout     OperacoesCheckout
	notificacoes RegistroNotificacoes
	pagamentos   ProcessadorPagamento
	frete        CotadorFrete
	ids          GeradorID
	relogio      Relogio
	politica     compras.PoliticaCobranca
}

func NovoControladorCheckout(
	usuarios OperacoesUsuarios,
	anuncios OperacoesAnuncios,
	carrinhos OperacoesCarrinhos,
	checkout OperacoesCheckout,
	notificacoes RegistroNotificacoes,
	pagamentos ProcessadorPagamento,
	frete CotadorFrete,
	ids GeradorID,
	relogio Relogio,
	politica compras.PoliticaCobranca,
) *ControladorCheckout {
	return &ControladorCheckout{
		usuarios: usuarios, anuncios: anuncios, carrinhos: carrinhos,
		checkout: checkout, notificacoes: notificacoes, pagamentos: pagamentos,
		frete: frete, ids: ids, relogio: relogio,
		politica: politica,
	}
}

// ResultadoCheckout e o retorno de FinalizarCompra. Embute a compra (aprovada, quando o
// provedor confirma de forma sincrona, ou pendente, quando aguarda confirmacao por webhook)
// e, no caso pendente, as instrucoes para o comprador concluir o pagamento (PIX/redirect).
type ResultadoCheckout struct {
	compras.Compra
	Instrucoes InstrucoesPagamento `json:"instrucoes_pagamento,omitempty"`
}

// FinalizarCompra reserva os itens antes de chamar o provedor financeiro. Assim, uma
// disputa concorrente e resolvida no PostgreSQL antes de qualquer cobranca.
//
// Com um provedor sincrono (o simulado do MVP) a venda e confirmada na hora. Com um provedor
// real, a cobranca nasce pendente: os itens seguem reservados, FinalizarCompra devolve as
// instrucoes de pagamento e a venda so se confirma quando o webhook chama
// ConfirmarPagamentoExterno (ou ate a intencao expirar e liberar a reserva).
func (c *ControladorCheckout) FinalizarCompra(
	ctx context.Context,
	idComprador, idEndereco string,
) (ResultadoCheckout, error) {
	compra, chave, emailComprador, err := c.montarCompraDoCarrinho(ctx, idComprador, idEndereco)
	if err != nil {
		return ResultadoCheckout{}, err
	}

	pagamento := compras.Pagamento{
		ID:                c.ids.Novo(),
		IDCompra:          compra.ID,
		Provedor:          "pendente",
		Status:            compras.StatusPagamentoPendente,
		ValorCentavos:     compra.ValorFinalPagoCentavos,
		ChaveIdempotencia: chave,
	}
	intencao, criada, err := c.checkout.IniciarCompra(ctx, compra, pagamento, idComprador)
	if err != nil {
		return ResultadoCheckout{}, err
	}
	if !criada {
		switch intencao.Status {
		case compras.StatusCompraAprovada:
			return ResultadoCheckout{Compra: intencao}, nil
		case compras.StatusCompraRecusada:
			return ResultadoCheckout{}, common.ErrPagamentoRecusado
		case compras.StatusCompraExpirada, compras.StatusCompraCancelada:
			return ResultadoCheckout{}, common.ErrTransicaoInvalida
		}
		// Intencao ainda pendente (aguardando_pagamento): recria a cobranca (idempotente no
		// provedor) para recuperar as instrucoes de pagamento.
	}

	cobranca, err := c.pagamentos.CriarCobranca(ctx, SolicitacaoPagamento{
		IDCompra:          intencao.ID,
		ValorCentavos:     intencao.ValorFinalPagoCentavos,
		ChaveIdempotencia: chave,
		EmailPagador:      emailComprador,
	})
	if err != nil {
		_ = c.checkout.RecusarCompra(ctx, chave, "", "", c.relogio.Agora())
		return ResultadoCheckout{}, err
	}

	switch cobranca.Status {
	case CobrancaRecusada:
		if err := c.checkout.RecusarCompra(ctx, chave, cobranca.Provedor, cobranca.IdentificadorExterno, c.relogio.Agora()); err != nil {
			return ResultadoCheckout{}, err
		}
		return ResultadoCheckout{}, common.ErrPagamentoRecusado
	case CobrancaPendente:
		return ResultadoCheckout{Compra: intencao, Instrucoes: cobranca.Instrucoes}, nil
	default: // CobrancaAprovada (provedor sincrono)
		confirmada, err := c.confirmarVenda(ctx, chave, cobranca.Provedor, cobranca.IdentificadorExterno)
		if err != nil {
			return ResultadoCheckout{}, err
		}
		return ResultadoCheckout{Compra: confirmada}, nil
	}
}

// ConfirmarPagamentoExterno aplica o desfecho de uma cobranca assincrona. E o ponto de
// entrada do webhook do provedor financeiro (e, futuramente, da reconciliacao): aprovado=true
// confirma a venda; false libera a reserva e registra a recusa. E idempotente — uma intencao
// ja concluida e devolvida sem reprocessar, tolerando reentregas do webhook.
func (c *ControladorCheckout) ConfirmarPagamentoExterno(
	ctx context.Context,
	chave, provedor, identificadorExterno string,
	aprovado bool,
) (compras.Compra, error) {
	intencao, err := c.checkout.BuscarCompraPorChave(ctx, chave)
	if err != nil {
		return compras.Compra{}, err
	}
	switch intencao.Status {
	case compras.StatusCompraAprovada:
		return intencao, nil
	case compras.StatusCompraRecusada, compras.StatusCompraExpirada, compras.StatusCompraCancelada:
		return compras.Compra{}, common.ErrTransicaoInvalida
	}
	if !aprovado {
		if err := c.checkout.RecusarCompra(ctx, chave, provedor, identificadorExterno, c.relogio.Agora()); err != nil {
			return compras.Compra{}, err
		}
		return compras.Compra{}, common.ErrPagamentoRecusado
	}
	return c.confirmarVenda(ctx, chave, provedor, identificadorExterno)
}

// confirmarVenda conclui uma intencao aprovada: vende os itens, aprova compra e pagamento,
// avanca os pedidos e limpa o carrinho; em seguida notifica os vendedores.
func (c *ControladorCheckout) confirmarVenda(
	ctx context.Context,
	chave, provedor, identificadorExterno string,
) (compras.Compra, error) {
	confirmada, err := c.checkout.ConfirmarCompraAprovada(
		ctx, chave, provedor, identificadorExterno, c.relogio.Agora(),
	)
	if err != nil {
		return compras.Compra{}, err
	}
	c.notificarVendedores(ctx, confirmada)
	return confirmada, nil
}

func (c *ControladorCheckout) notificarVendedores(ctx context.Context, compra compras.Compra) {
	if c.notificacoes == nil {
		return
	}
	for _, pedido := range compra.Pedidos {
		quantidade := len(pedido.Itens)
		pecas := "peças"
		if quantidade == 1 {
			pecas = "peça"
		}
		_ = c.notificacoes.CriarNotificacao(ctx, interacao.Notificacao{
			ID:        c.ids.Novo(),
			IDUsuario: pedido.IDVendedor,
			Tipo:      interacao.NotificacaoVendaRealizada,
			Conteudo:  fmt.Sprintf("Nova venda! Você vendeu %d %s. Prepare o envio dentro do prazo.", quantidade, pecas),
			IDPedido:  pedido.ID,
			CriadaEm:  c.relogio.Agora(),
		})
	}
}

// ListarPedidos devolve os pedidos do comprador, dos mais recentes para os mais antigos.
func (c *ControladorCheckout) ListarPedidos(
	ctx context.Context,
	idComprador string,
) ([]compras.Pedido, error) {
	pedidos, err := c.checkout.ListarPedidosDoComprador(ctx, idComprador)
	if err != nil {
		return nil, err
	}
	if pedidos == nil {
		pedidos = []compras.Pedido{}
	}
	return pedidos, nil
}

// ProcessarExpiracoes libera reservas de intencoes cujo pagamento nao foi concluido.
// Deve ser executado periodicamente pelo processo de jobs.
func (c *ControladorCheckout) ProcessarExpiracoes(ctx context.Context) (int, error) {
	return c.checkout.ExpirarComprasPendentes(ctx, c.relogio.Agora())
}

// ResumoCheckout projeta a compra a partir do carrinho sem cobrar nem persistir nada.
// Serve para a tela de revisao do checkout: itens agrupados por vendedor, fretes e totais.
// A confirmacao real ocorre em FinalizarCompra, que e idempotente por carrinho.
func (c *ControladorCheckout) ResumoCheckout(
	ctx context.Context,
	idComprador, idEndereco string,
) (compras.Compra, error) {
	compra, _, _, err := c.montarCompraDoCarrinho(ctx, idComprador, idEndereco)
	return compra, err
}

// montarCompraDoCarrinho le o carrinho do comprador, projeta os itens disponiveis e monta a
// compra (um pedido por vendedor, com totais), devolvendo tambem a chave de idempotencia.
// Nao processa pagamento nem persiste: e o trecho comum entre ResumoCheckout e FinalizarCompra.
// O endereco de entrega e o escolhido (idEndereco) ou, se vazio, o principal do comprador.
func (c *ControladorCheckout) montarCompraDoCarrinho(
	ctx context.Context,
	idComprador, idEndereco string,
) (compras.Compra, string, string, error) {
	comprador, err := c.usuarios.BuscarUsuarioPorID(ctx, idComprador)
	if err != nil {
		return compras.Compra{}, "", "", err
	}

	enderecoEntrega := comprador.EnderecoPrincipal
	if idEndereco != "" {
		escolhido, err := c.usuarios.BuscarEndereco(ctx, idComprador, idEndereco)
		if err != nil {
			return compras.Compra{}, "", "", err
		}
		enderecoEntrega = escolhido
	}

	carrinho, err := c.carrinhos.ObterOuCriarCarrinho(ctx, c.ids.Novo(), idComprador, c.relogio.Agora())
	if err != nil {
		return compras.Compra{}, "", "", err
	}
	if len(carrinho.IDsAnuncios) == 0 {
		return compras.Compra{}, "", "", common.ErrCarrinhoVazio
	}

	itens, err := c.itensCompraveis(ctx, carrinho.IDsAnuncios, idComprador)
	if err != nil {
		return compras.Compra{}, "", "", err
	}
	if len(itens) == 0 {
		return compras.Compra{}, "", "", common.ErrSemItensDisponiveis
	}

	idsAnuncios := make([]string, 0, len(itens))
	for _, item := range itens {
		idsAnuncios = append(idsAnuncios, item.IDAnuncio)
	}
	versaoCarrinho := carrinho.AtualizadoEm.UTC().Format(time.RFC3339Nano)
	chave := compras.ChaveIdempotenciaCarrinho(
		idComprador,
		append([]string{carrinho.ID, versaoCarrinho}, idsAnuncios...),
	)

	fretePorVendedor := c.cotarFretesPorVendedor(ctx, itens, enderecoEntrega.CEP)

	agora := c.relogio.Agora()
	compra, err := compras.MontarCompra(compras.ParametrosCompra{
		IDCompra:          c.ids.Novo(),
		IDComprador:       idComprador,
		NomeDestinatario:  comprador.Nome,
		EnderecoEntrega:   enderecoEntrega,
		Itens:             itens,
		Politica:          c.politica,
		FretePorVendedor:  fretePorVendedor,
		ChaveIdempotencia: chave,
		Agora:             agora,
		ExpiraEm:          agora.Add(validadeCompra),
		PrazoEnvio:        agora.Add(prazoEnvioItem),
		GerarID:           c.ids.Novo,
	})
	if err != nil {
		return compras.Compra{}, "", "", err
	}
	return compra, chave, comprador.Email, nil
}

// itensCompraveis projeta os anuncios do carrinho que estao disponiveis e nao pertencem
// ao proprio comprador. Itens indisponiveis sao silenciosamente ignorados (o comprador
// finaliza o que ainda da para comprar).
func (c *ControladorCheckout) itensCompraveis(
	ctx context.Context,
	idsAnuncios []string,
	idComprador string,
) ([]compras.ItemCompravel, error) {
	lista, err := c.anuncios.ListarAnuncios(ctx, FiltroAnuncios{
		IDsAnuncios:        idsAnuncios,
		IncluirTodosStatus: true,
		Limite:             len(idsAnuncios),
	})
	if err != nil {
		return nil, err
	}
	itens := make([]compras.ItemCompravel, 0, len(lista))
	for _, anuncio := range lista {
		if anuncio.Status != anuncios.StatusAnuncioDisponivel || anuncio.IDVendedor == idComprador {
			continue
		}
		itens = append(itens, compras.ItemCompravel{
			IDAnuncio:         anuncio.ID,
			IDVendedor:        anuncio.IDVendedor,
			Titulo:            anuncio.Titulo,
			Categoria:         anuncio.Categoria,
			Tamanho:           anuncio.Tamanho,
			Cor:               anuncio.Cor,
			EstadoConservacao: anuncio.EstadoConservacao,
			PrecoCentavos:     anuncio.PrecoCentavos,
			PesoGramas:        anuncio.PesoGramas,
			AlturaCm:          anuncio.AlturaCm,
			LarguraCm:         anuncio.LarguraCm,
			ComprimentoCm:     anuncio.ComprimentoCm,
		})
	}
	return itens, nil
}

// cotarFretesPorVendedor cota o frete de cada vendedor (um pedido por vendedor), agrupando
// os itens e usando o CEP do vendedor como origem e o CEP de entrega como destino. Falhas de
// cotacao caem no frete de contingencia, para que a indisponibilidade do provedor de frete
// nunca bloqueie o checkout.
func (c *ControladorCheckout) cotarFretesPorVendedor(
	ctx context.Context,
	itens []compras.ItemCompravel,
	destinoCEP string,
) map[string]int64 {
	itensPorVendedor := make(map[string][]compras.ItemCompravel)
	for _, item := range itens {
		itensPorVendedor[item.IDVendedor] = append(itensPorVendedor[item.IDVendedor], item)
	}
	fretes := make(map[string]int64, len(itensPorVendedor))
	for idVendedor, itensVendedor := range itensPorVendedor {
		fretes[idVendedor] = c.cotarFreteVendedor(ctx, idVendedor, destinoCEP, itensVendedor)
	}
	return fretes
}

func (c *ControladorCheckout) cotarFreteVendedor(
	ctx context.Context,
	idVendedor, destinoCEP string,
	itensVendedor []compras.ItemCompravel,
) int64 {
	contingencia := c.politica.FretePorPedidoCentavos
	if c.frete == nil {
		return contingencia
	}
	vendedor, err := c.usuarios.BuscarUsuarioPorID(ctx, idVendedor)
	if err != nil || vendedor.EnderecoPrincipal.CEP == "" {
		return contingencia
	}
	itensFrete := make([]ItemFrete, 0, len(itensVendedor))
	for _, item := range itensVendedor {
		itensFrete = append(itensFrete, ItemFrete{
			PesoGramas:    item.PesoGramas,
			AlturaCm:      item.AlturaCm,
			LarguraCm:     item.LarguraCm,
			ComprimentoCm: item.ComprimentoCm,
			ValorCentavos: item.PrecoCentavos,
		})
	}
	cotacao, err := c.frete.Cotar(ctx, vendedor.EnderecoPrincipal.CEP, destinoCEP, itensFrete)
	if err != nil {
		return contingencia
	}
	return cotacao.ValorCentavos
}
