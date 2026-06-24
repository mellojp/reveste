package web

import (
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/anuncios"
	"reveste/apps/back/internal/dominio/cadastros"
	"reveste/apps/back/internal/dominio/compras"
	"reveste/apps/back/internal/dominio/interacao"
)

func funcoesApresentacaoTemplates() template.FuncMap {
	return template.FuncMap{
		"formatarDinheiro":   formatarDinheiro,
		"formatarData":       formatarData,
		"linhaDoTempoPedido": linhaDoTempoPedido,
		"linhaDeEstrelas":    linhaDeEstrelas,
		"extrairAno":         func(valor time.Time) int { return valor.Year() },
		"formatarRotulo":     formatarRotulo,
		"classeStatus":       classeStatus,
		"taxaReativacao":     func() int64 { return cadastros.TaxaReativacaoCentavos },
		"linkNotificacao":    linkNotificacao,
		"iconeNotificacao":   iconeNotificacao,
		"iniciais":           iniciais,
		"primeiroNome":       primeiroNome,
		"incrementar":        func(valor int) int { return valor + 1 },
		"anuncioNoCarrinho":  carrinhoContemAnuncio,
		"contarDisponiveis":  contarAnunciosDisponiveis,
		"contarIndisponiveis": func(itens []anuncios.Anuncio) int {
			return len(itens) - contarAnunciosDisponiveis(itens)
		},
		"fotoCapa": fotoCapa,
		"contextoCartaoAnuncio": func(item anuncios.Anuncio, contexto contextoDocumento) contextoCartaoAnuncio {
			idUsuario := ""
			if contexto.UsuarioAutenticado != nil {
				idUsuario = contexto.UsuarioAutenticado.ID
			}
			return contextoCartaoAnuncio{
				Anuncio: item, IDUsuarioAutenticado: idUsuario, URLRetorno: contexto.URLRetorno,
				EstaNoCarrinho: carrinhoContemAnuncio(contexto.CarrinhoAutenticado, item.ID),
			}
		},
		"contextoErroCampo": func(erros map[string]string, nomeCampo string) contextoMensagemCampo {
			return contextoMensagemCampo{
				ErrosValidacao: erros,
				NomeCampo:      nomeCampo,
				IDMensagem:     "erro-" + strings.ReplaceAll(nomeCampo, ".", "-"),
			}
		},
		"temErroCampo": func(erros map[string]string, nome string) bool {
			return mensagemErroCampo(erros, nome) != ""
		},
		"formatarMesAno":    formatarMesAno,
		"mensagemErroCampo": mensagemErroCampo,
		"valorFormulario": func(valores map[string]string, nome string) string {
			return valores[nome]
		},
	}
}

func iconeNotificacao(tipo string) string {
	switch tipo {
	case interacao.NotificacaoVendaRealizada:
		return "✓"
	case interacao.NotificacaoPedidoEnviado:
		return "↗"
	case interacao.NotificacaoPedidoRecebido:
		return "⌂"
	case interacao.NotificacaoAvaliacaoRecebida:
		return "★"
	case interacao.NotificacaoMensagemRecebida:
		return "✦"
	default:
		return "R"
	}
}

func mensagemErroCampo(campos map[string]string, nome string) string {
	if mensagem := campos[nome]; mensagem != "" {
		return mensagem
	}
	return campos["endereco."+nome]
}

func apresentarErroCasoUso(err error) (string, map[string]string) {
	var validacao common.ErroValidacao
	var conflito common.ErroConflitoCampo
	switch {
	case errors.As(err, &validacao):
		return "Revise os campos destacados.", validacao.Campos
	case errors.As(err, &conflito):
		return "Já existe uma conta com os campos destacados.", conflito.Campos
	case errors.Is(err, common.ErrNaoAutorizado):
		return "E-mail, CPF ou senha inválidos.", map[string]string{}
	case errors.Is(err, common.ErrNaoPermitido):
		return "Você não pode realizar esta operação.", map[string]string{}
	case errors.Is(err, common.ErrAnuncioIndisponivel):
		return "Este anúncio não está mais disponível.", map[string]string{}
	default:
		return "Não foi possível concluir a operação.", map[string]string{}
	}
}

func formatarDinheiro(centavos int64) string {
	reais := float64(centavos) / 100
	texto := strconv.FormatFloat(reais, 'f', 2, 64)
	partes := strings.Split(texto, ".")
	return "R$ " + partes[0] + "," + partes[1]
}

// formatarData aceita time.Time ou *time.Time (campos opcionais como enviado_em/postado_em),
// devolvendo vazio para ponteiro nulo ou data zerada.
func formatarData(valor any) string {
	instante, ok := comoTempo(valor)
	if !ok || instante.IsZero() {
		return ""
	}
	return instante.Format("02/01/2006")
}

func comoTempo(valor any) (time.Time, bool) {
	switch v := valor.(type) {
	case time.Time:
		return v, true
	case *time.Time:
		if v == nil {
			return time.Time{}, false
		}
		return *v, true
	default:
		return time.Time{}, false
	}
}

// passoPedido descreve um marco da linha do tempo do pedido para o comprador.
type passoPedido struct {
	Rotulo    string
	Concluido bool
	Atual     bool
}

// linhaDoTempoPedido projeta o status do pedido em marcos visuais (Pago -> Enviado -> Recebido).
func linhaDoTempoPedido(status compras.StatusPedido) []passoPedido {
	nivel := 1
	switch status {
	case compras.StatusPedidoAguardandoEntrega:
		nivel = 2
	case compras.StatusPedidoFinalizado:
		nivel = 3
	}
	rotulos := []string{"Pago", "Enviado", "Recebido"}
	passos := make([]passoPedido, len(rotulos))
	for indice, rotulo := range rotulos {
		passos[indice] = passoPedido{
			Rotulo:    rotulo,
			Concluido: indice+1 <= nivel,
			Atual:     indice+1 == nivel,
		}
	}
	return passos
}

// linhaDeEstrelas devolve cinco posicoes (true = preenchida) para renderizar a nota visualmente.
func linhaDeEstrelas(nota int) []bool {
	estrelas := make([]bool, 5)
	for indice := range estrelas {
		estrelas[indice] = indice < nota
	}
	return estrelas
}

func formatarMesAno(valor time.Time) string {
	meses := [...]string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}
	if valor.IsZero() {
		return ""
	}
	return meses[valor.Month()-1] + " de " + strconv.Itoa(valor.Year())
}

func formatarRotulo(valor any) string {
	texto := strings.ReplaceAll(fmt.Sprint(valor), "_", " ")
	if texto == "" {
		return ""
	}
	return strings.ToUpper(texto[:1]) + texto[1:]
}

// classeStatus projeta um status de anuncio ou pedido em um tom visual do badge
// (sucesso, em andamento, encerrado negativo ou neutro), para que o estado seja
// lido de relance sem depender apenas do texto.
func classeStatus(valor any) string {
	switch fmt.Sprint(valor) {
	case "disponivel", "finalizado", "aprovada", "aprovado", "recebido", "entregue", "processado",
		interacao.NotificacaoVendaRealizada, interacao.NotificacaoPedidoRecebido, interacao.NotificacaoAvaliacaoRecebida:
		return "is-success"
	case "aguardando_pagamento", "aguardando_envio", "aguardando_entrega", "aguardando_postagem",
		"criado", "reservado", "pendente", "enviado", "postado", "em_transito",
		interacao.NotificacaoPedidoEnviado, interacao.NotificacaoMensagemRecebida:
		return "is-progress"
	case "cancelado", "cancelada", "recusada", "recusado", "expirado", "expirada",
		"suspenso", "excluido", "falhou", "nao_enviado":
		return "is-danger"
	default:
		return "is-neutral"
	}
}

// linkNotificacao resolve o destino de uma notificacao a partir do tipo do evento e do
// pedido associado. O lado (comprador ou vendedor) e inferido pelo tipo, ja que o
// destinatario da notificacao corresponde ao papel esperado em cada evento.
func linkNotificacao(n interacao.Notificacao) string {
	if n.IDPedido == "" {
		return ""
	}
	switch n.Tipo {
	case interacao.NotificacaoVendaRealizada:
		return "/minhas-vendas/" + n.IDPedido
	case interacao.NotificacaoPedidoEnviado:
		return "/meus-pedidos/" + n.IDPedido
	case interacao.NotificacaoPedidoRecebido, interacao.NotificacaoAvaliacaoRecebida:
		return "/minhas-vendas/" + n.IDPedido
	case interacao.NotificacaoMensagemRecebida:
		return "/pedidos/" + n.IDPedido + "/conversa"
	default:
		return ""
	}
}

func iniciais(nome string) string {
	partes := strings.Fields(nome)
	resultado := ""
	for i := 0; i < len(partes) && i < 2; i++ {
		resultado += strings.ToUpper(string([]rune(partes[i])[0]))
	}
	return resultado
}

func primeiroNome(nome string) string {
	partes := strings.Fields(nome)
	if len(partes) == 0 {
		return ""
	}
	return partes[0]
}

func carrinhoContemAnuncio(carrinho casosdeuso.CarrinhoDetalhado, id string) bool {
	for _, item := range carrinho.Anuncios {
		if item.ID == id {
			return true
		}
	}
	return false
}

func contarAnunciosDisponiveis(itens []anuncios.Anuncio) int {
	total := 0
	for _, item := range itens {
		if item.Status == anuncios.StatusAnuncioDisponivel {
			total++
		}
	}
	return total
}

func fotoCapa(item anuncios.Anuncio) string {
	if len(item.Fotos) == 0 {
		return ""
	}
	return item.Fotos[0].URL
}
