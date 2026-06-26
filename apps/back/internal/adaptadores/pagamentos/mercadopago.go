package pagamentos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"reveste/apps/back/internal/casosdeuso"
)

const (
	provedorMercadoPago = "mercadopago"
	caminhoOrders       = "/v1/orders"
	emailPadraoPagador  = "comprador@reveste.com.br"
)

// MercadoPago implementa casosdeuso.ProcessadorPagamento e a verificacao do webhook usando a
// Orders API do Mercado Pago (https://www.mercadopago.com.br/developers, Checkout Transparente).
// No MVP cria cobrancas PIX: CriarCobranca abre uma order com pagamento PIX e devolve a cobranca
// pendente com o QR Code; a confirmacao chega depois pelo webhook do topico "order", interpretado
// por Interpretar. Valores trafegam em reais (string com 2 casas); internamente usamos centavos.
type MercadoPago struct {
	cliente        *http.Client
	urlBase        string
	accessToken    string
	webhookSecret  string
	notificacaoURL string // mantido por compatibilidade; a Orders API usa o webhook do painel
}

// NovoMercadoPago cria o adaptador. urlBase aponta para a API (producao por padrao; o sandbox
// usa a mesma base com credenciais de teste). accessToken e o token da aplicacao; webhookSecret
// e a chave de assinatura das notificacoes; notificacaoURL fica registrado mas a Orders API usa a
// URL configurada no painel do Mercado Pago (topico "Order").
func NovoMercadoPago(urlBase, accessToken, webhookSecret, notificacaoURL string) *MercadoPago {
	if strings.TrimSpace(urlBase) == "" {
		urlBase = "https://api.mercadopago.com"
	}
	return &MercadoPago{
		cliente:        &http.Client{Timeout: 10 * time.Second},
		urlBase:        strings.TrimRight(urlBase, "/"),
		accessToken:    accessToken,
		webhookSecret:  webhookSecret,
		notificacaoURL: notificacaoURL,
	}
}

type metodoPagamentoOrder struct {
	ID           string `json:"id"`   // "pix" ou bandeira do cartao ("master", "visa"...)
	Type         string `json:"type"` // "bank_transfer" (pix) ou "credit_card"
	Token        string `json:"token,omitempty"`
	Installments int    `json:"installments,omitempty"`
}

type identificacaoPagador struct {
	Type   string `json:"type"`
	Number string `json:"number"`
}

type pagamentoOrder struct {
	Amount        string               `json:"amount"`
	PaymentMethod metodoPagamentoOrder `json:"payment_method"`
}

type transacoesOrder struct {
	Payments []pagamentoOrder `json:"payments"`
}

type pagadorOrder struct {
	Email          string                `json:"email"`
	FirstName      string                `json:"first_name,omitempty"`
	LastName       string                `json:"last_name,omitempty"`
	Identification *identificacaoPagador `json:"identification,omitempty"`
}

type requisicaoOrder struct {
	Type              string          `json:"type"`            // "online"
	ProcessingMode    string          `json:"processing_mode"` // "automatic"
	TotalAmount       string          `json:"total_amount"`
	ExternalReference string          `json:"external_reference"`
	Transactions      transacoesOrder `json:"transactions"`
	Payer             pagadorOrder    `json:"payer"`
}

// respostaOrder cobre os campos da order que nos interessam: status (agregado), referencia
// externa e, no primeiro pagamento, os dados do PIX (QR Code, copia-e-cola e ticket).
type respostaOrder struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	ExternalReference string `json:"external_reference"`
	Transactions      struct {
		Payments []struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			PaymentMethod struct {
				ID           string `json:"id"`
				Type         string `json:"type"`
				TicketURL    string `json:"ticket_url"`
				QRCode       string `json:"qr_code"`
				QRCodeBase64 string `json:"qr_code_base64"`
			} `json:"payment_method"`
		} `json:"payments"`
	} `json:"transactions"`
}

// CriarCobranca abre uma order PIX no Mercado Pago em modo automatico. A ChaveIdempotencia vai no
// header X-Idempotency-Key e em external_reference, garantindo que repeticoes nao gerem cobranca
// nova e que o webhook consiga reassociar a order a intencao de compra. No sandbox, o primeiro
// nome do pagador ("APRO"/"CONT"/"OTHE") define o desfecho do pagamento de teste.
func (m *MercadoPago) CriarCobranca(
	ctx context.Context,
	solicitacao casosdeuso.SolicitacaoPagamento,
) (casosdeuso.Cobranca, error) {
	email := strings.TrimSpace(solicitacao.EmailPagador)
	if email == "" {
		email = emailPadraoPagador
	}
	valor := formatarValor(solicitacao.ValorCentavos)
	primeiroNome, sobrenome := dividirNome(solicitacao.NomePagador)
	metodo := metodoPagamentoOrder{ID: "pix", Type: "bank_transfer"}
	pagador := pagadorOrder{Email: email, FirstName: primeiroNome, LastName: sobrenome}
	if cartao := solicitacao.Cartao; cartao != nil {
		parcelas := cartao.Parcelas
		if parcelas < 1 {
			parcelas = 1
		}
		metodo = metodoPagamentoOrder{
			ID:           cartao.MetodoPagamento,
			Type:         "credit_card",
			Token:        cartao.Token,
			Installments: parcelas,
		}
		if cartao.NumeroDocumento != "" {
			pagador.Identification = &identificacaoPagador{Type: cartao.TipoDocumento, Number: cartao.NumeroDocumento}
		}
	}
	corpo, err := json.Marshal(requisicaoOrder{
		Type:              "online",
		ProcessingMode:    "automatic",
		TotalAmount:       valor,
		ExternalReference: solicitacao.ChaveIdempotencia,
		Transactions: transacoesOrder{
			Payments: []pagamentoOrder{{Amount: valor, PaymentMethod: metodo}},
		},
		Payer: pagador,
	})
	if err != nil {
		return casosdeuso.Cobranca{}, fmt.Errorf("mercadopago: montar requisicao: %w", err)
	}

	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, m.urlBase+caminhoOrders, bytes.NewReader(corpo))
	if err != nil {
		return casosdeuso.Cobranca{}, fmt.Errorf("mercadopago: criar requisicao: %w", err)
	}
	requisicao.Header.Set("Content-Type", "application/json")
	requisicao.Header.Set("Accept", "application/json")
	requisicao.Header.Set("Authorization", "Bearer "+m.accessToken)
	requisicao.Header.Set("X-Idempotency-Key", solicitacao.ChaveIdempotencia)

	resposta, err := m.cliente.Do(requisicao)
	if err != nil {
		return casosdeuso.Cobranca{}, fmt.Errorf("mercadopago: chamar provedor: %w", err)
	}
	defer resposta.Body.Close()
	corpoResposta, _ := io.ReadAll(io.LimitReader(resposta.Body, 1<<20))
	if resposta.StatusCode != http.StatusOK && resposta.StatusCode != http.StatusCreated {
		return casosdeuso.Cobranca{}, fmt.Errorf("mercadopago: status %d: %s", resposta.StatusCode, string(corpoResposta))
	}

	var order respostaOrder
	if err := json.Unmarshal(corpoResposta, &order); err != nil {
		return casosdeuso.Cobranca{}, fmt.Errorf("mercadopago: decodificar resposta: %w", err)
	}

	cobranca := casosdeuso.Cobranca{
		Status:               mapearStatus(order.Status),
		Provedor:             provedorMercadoPago,
		IdentificadorExterno: order.ID,
	}
	if cobranca.Status == casosdeuso.CobrancaPendente && len(order.Transactions.Payments) > 0 {
		dados := order.Transactions.Payments[0].PaymentMethod
		cobranca.Instrucoes = casosdeuso.InstrucoesPagamento{
			Tipo:         "pix",
			PixCopiaCola: dados.QRCode,
			PixQRCode:    dados.QRCodeBase64,
			URLPagamento: dados.TicketURL,
		}
	}
	return cobranca, nil
}

// Interpretar valida a notificacao e devolve a chave de idempotencia (external_reference), o
// identificador externo e o status atual. Trata os dois topicos que a aprovacao de uma order PIX
// gera: "order" (consulta /v1/orders/{id}) e "payment" (consulta /v1/payments/{id}) — assim a
// confirmacao funciona seja qual for o topico assinado no painel. Outros topicos sao confirmados
// (200) sem acao.
func (m *MercadoPago) Interpretar(
	r *http.Request,
) (chave, provedor, idExterno string, status casosdeuso.StatusCobranca, err error) {
	idRecurso, tipo := dadosNotificacao(r)
	if idRecurso == "" {
		return "", "", "", "", fmt.Errorf("mercadopago: notificacao sem id de recurso")
	}
	ehPayment := strings.EqualFold(tipo, "payment")
	ehOrder := tipo == "" || strings.EqualFold(tipo, "order")
	if !ehPayment && !ehOrder {
		// Topico irrelevante (ex.: merchant_order): apenas confirma o recebimento.
		return "", provedorMercadoPago, idRecurso, casosdeuso.CobrancaPendente, nil
	}
	if !m.assinaturaValida(r, idRecurso) {
		return "", "", "", "", fmt.Errorf("mercadopago: assinatura do webhook invalida")
	}

	if ehPayment {
		pagamento, err := m.consultarPagamento(r.Context(), idRecurso)
		if err != nil {
			return "", "", "", "", err
		}
		return pagamento.ExternalReference, provedorMercadoPago, pagamento.ID.String(), mapearStatusPagamento(pagamento.Status), nil
	}

	order, err := m.consultarOrder(r.Context(), idRecurso)
	if err != nil {
		return "", "", "", "", err
	}
	return order.ExternalReference, provedorMercadoPago, order.ID, mapearStatus(order.Status), nil
}

type respostaPagamento struct {
	ID                json.Number `json:"id"`
	Status            string      `json:"status"`
	ExternalReference string      `json:"external_reference"`
}

func (m *MercadoPago) consultarPagamento(ctx context.Context, id string) (respostaPagamento, error) {
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodGet, m.urlBase+"/v1/payments/"+id, nil)
	if err != nil {
		return respostaPagamento{}, fmt.Errorf("mercadopago: criar consulta de pagamento: %w", err)
	}
	requisicao.Header.Set("Accept", "application/json")
	requisicao.Header.Set("Authorization", "Bearer "+m.accessToken)

	resposta, err := m.cliente.Do(requisicao)
	if err != nil {
		return respostaPagamento{}, fmt.Errorf("mercadopago: consultar pagamento: %w", err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		return respostaPagamento{}, fmt.Errorf("mercadopago: consulta de pagamento status %d", resposta.StatusCode)
	}
	var pagamento respostaPagamento
	if err := json.NewDecoder(io.LimitReader(resposta.Body, 1<<20)).Decode(&pagamento); err != nil {
		return respostaPagamento{}, fmt.Errorf("mercadopago: decodificar consulta de pagamento: %w", err)
	}
	return pagamento, nil
}

// ReconciliarCobranca consulta o provedor pela chave de idempotencia (external_reference) e devolve
// o status atual, o provedor e o id externo do pagamento. Serve de fallback quando o webhook nao
// chega: a tela de pagamento reconcilia em polling, sem depender da entrega da notificacao.
// encontrada=false quando o provedor ainda nao tem nenhum pagamento para a chave.
func (m *MercadoPago) ReconciliarCobranca(
	ctx context.Context,
	chave string,
) (status casosdeuso.StatusCobranca, provedor, identificadorExterno string, encontrada bool, err error) {
	endereco := m.urlBase + "/v1/payments/search?external_reference=" + url.QueryEscape(chave)
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return "", "", "", false, fmt.Errorf("mercadopago: criar busca: %w", err)
	}
	requisicao.Header.Set("Accept", "application/json")
	requisicao.Header.Set("Authorization", "Bearer "+m.accessToken)

	resposta, err := m.cliente.Do(requisicao)
	if err != nil {
		return "", "", "", false, fmt.Errorf("mercadopago: buscar pagamento: %w", err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		return "", "", "", false, fmt.Errorf("mercadopago: busca status %d", resposta.StatusCode)
	}
	var corpo struct {
		Results []struct {
			ID     json.Number `json:"id"`
			Status string      `json:"status"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resposta.Body, 1<<20)).Decode(&corpo); err != nil {
		return "", "", "", false, fmt.Errorf("mercadopago: decodificar busca: %w", err)
	}
	if len(corpo.Results) == 0 {
		return "", "", "", false, nil
	}
	// Pode haver mais de um pagamento para a mesma intencao (ex.: tentativa recusada + aprovada);
	// um pagamento aprovado tem prioridade.
	melhor := corpo.Results[0]
	for _, pagamento := range corpo.Results {
		if pagamento.Status == "approved" {
			melhor = pagamento
			break
		}
	}
	return mapearStatusPagamento(melhor.Status), provedorMercadoPago, melhor.ID.String(), true, nil
}

// mapearStatusPagamento traduz o status de um pagamento (topico payment) para o dominio.
func mapearStatusPagamento(status string) casosdeuso.StatusCobranca {
	switch status {
	case "approved":
		return casosdeuso.CobrancaAprovada
	case "rejected", "cancelled", "refunded", "charged_back":
		return casosdeuso.CobrancaRecusada
	default: // pending, in_process, in_mediation, authorized
		return casosdeuso.CobrancaPendente
	}
}

func (m *MercadoPago) consultarOrder(ctx context.Context, id string) (respostaOrder, error) {
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodGet, m.urlBase+caminhoOrders+"/"+id, nil)
	if err != nil {
		return respostaOrder{}, fmt.Errorf("mercadopago: criar consulta: %w", err)
	}
	requisicao.Header.Set("Accept", "application/json")
	requisicao.Header.Set("Authorization", "Bearer "+m.accessToken)

	resposta, err := m.cliente.Do(requisicao)
	if err != nil {
		return respostaOrder{}, fmt.Errorf("mercadopago: consultar order: %w", err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		return respostaOrder{}, fmt.Errorf("mercadopago: consulta status %d", resposta.StatusCode)
	}
	var order respostaOrder
	if err := json.NewDecoder(io.LimitReader(resposta.Body, 1<<20)).Decode(&order); err != nil {
		return respostaOrder{}, fmt.Errorf("mercadopago: decodificar consulta: %w", err)
	}
	return order, nil
}

// assinaturaValida confere o header x-signature do Mercado Pago. A assinatura cobre o manifesto
// "id:<data.id>;request-id:<x-request-id>;ts:<ts>;", com HMAC-SHA256 sobre a webhookSecret. Sem
// secret configurado, rejeita (fail-closed).
func (m *MercadoPago) assinaturaValida(r *http.Request, idRecurso string) bool {
	if m.webhookSecret == "" {
		return false
	}
	ts, v1 := partesAssinatura(r.Header.Get("x-signature"))
	if ts == "" || v1 == "" {
		return false
	}
	requestID := r.Header.Get("x-request-id")
	// O Mercado Pago documenta o id em minusculas no manifesto, mas o id da order e alfanumerico
	// (ORDTST...); aceitamos tambem a forma original para nao depender dessa convencao.
	for _, id := range []string{strings.ToLower(idRecurso), idRecurso} {
		manifesto := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", id, requestID, ts)
		mac := hmac.New(sha256.New, []byte(m.webhookSecret))
		mac.Write([]byte(manifesto))
		if hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(v1)) {
			return true
		}
	}
	return false
}

// partesAssinatura extrai ts e v1 do header "ts=...,v1=...".
func partesAssinatura(cabecalho string) (ts, v1 string) {
	for _, parte := range strings.Split(cabecalho, ",") {
		chave, valor, ok := strings.Cut(strings.TrimSpace(parte), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(chave) {
		case "ts":
			ts = strings.TrimSpace(valor)
		case "v1":
			v1 = strings.TrimSpace(valor)
		}
	}
	return ts, v1
}

// dadosNotificacao extrai o id do recurso (data.id) e o tipo/topico da notificacao, aceitando o
// formato de query (?data.id=...&type=order) e o corpo JSON ({"type":"order","data":{"id":...}}).
// O corpo e lido no maximo uma vez.
func dadosNotificacao(r *http.Request) (id, tipo string) {
	id = r.URL.Query().Get("data.id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	tipo = r.URL.Query().Get("type")
	if tipo == "" {
		tipo = r.URL.Query().Get("topic")
	}
	if id != "" && tipo != "" {
		return id, tipo
	}
	var corpo struct {
		Type  string `json:"type"`
		Topic string `json:"topic"`
		Data  struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&corpo); err == nil {
		if id == "" {
			id = corpo.Data.ID
		}
		if tipo == "" {
			if corpo.Type != "" {
				tipo = corpo.Type
			} else {
				tipo = corpo.Topic
			}
		}
	}
	return id, tipo
}

// dividirNome separa o nome completo do pagador em primeiro nome e sobrenome para os campos
// payer.first_name e payer.last_name do Mercado Pago. No sandbox, o primeiro nome ("APRO",
// "CONT", "OTHE") define o desfecho do pagamento de teste.
func dividirNome(nome string) (primeiro, sobrenome string) {
	campos := strings.Fields(nome)
	if len(campos) == 0 {
		return "", ""
	}
	return campos[0], strings.Join(campos[1:], " ")
}

// formatarValor converte centavos no formato de string com duas casas decimais exigido pela
// Orders API (ex.: 11990 -> "119.90").
func formatarValor(centavos int64) string {
	return fmt.Sprintf("%d.%02d", centavos/100, centavos%100)
}

// mapearStatus traduz o status (agregado) da order do Mercado Pago para o StatusCobranca do
// dominio. Ver Status da order: processed (creditado), action_required/processing/created
// (aguardando), failed/canceled/expired/refunded/charged_back (sem sucesso).
func mapearStatus(status string) casosdeuso.StatusCobranca {
	switch status {
	case "processed":
		return casosdeuso.CobrancaAprovada
	case "failed", "canceled", "cancelled", "expired", "refunded", "charged_back":
		return casosdeuso.CobrancaRecusada
	default: // created, processing, action_required: aguarda confirmacao
		return casosdeuso.CobrancaPendente
	}
}
