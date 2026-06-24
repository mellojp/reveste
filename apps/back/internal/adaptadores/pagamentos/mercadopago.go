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
	"strings"
	"time"

	"reveste/apps/back/internal/casosdeuso"
)

const (
	provedorMercadoPago = "mercadopago"
	caminhoPagamentos   = "/v1/payments"
	emailPadraoPagador  = "comprador@reveste.com.br"
)

// MercadoPago implementa casosdeuso.ProcessadorPagamento e a verificacao do webhook usando a
// API do Mercado Pago (https://www.mercadopago.com.br/developers). No MVP cria cobrancas PIX:
// CriarCobranca devolve a cobranca pendente com o QR Code, e a confirmacao chega depois pelo
// webhook, interpretado por Interpretar. Valores trafegam em reais; internamente usamos centavos.
type MercadoPago struct {
	cliente        *http.Client
	urlBase        string
	accessToken    string
	webhookSecret  string
	notificacaoURL string
}

// NovoMercadoPago cria o adaptador. urlBase aponta para a API (producao por padrao; o sandbox
// usa a mesma base com credenciais de teste). accessToken e o token da aplicacao; webhookSecret
// e a chave de assinatura das notificacoes; notificacaoURL e a URL publica do webhook (opcional:
// se vazia, usa-se a configurada no painel do Mercado Pago).
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

type pagador struct {
	Email string `json:"email"`
}

type requisicaoPagamento struct {
	TransactionAmount float64 `json:"transaction_amount"`
	Description       string  `json:"description"`
	PaymentMethodID   string  `json:"payment_method_id"`
	ExternalReference string  `json:"external_reference"`
	NotificationURL   string  `json:"notification_url,omitempty"`
	Payer             pagador `json:"payer"`
}

type respostaPagamento struct {
	ID                 json.Number `json:"id"`
	Status             string      `json:"status"`
	ExternalReference  string      `json:"external_reference"`
	PointOfInteraction struct {
		TransactionData struct {
			QRCode       string `json:"qr_code"`
			QRCodeBase64 string `json:"qr_code_base64"`
			TicketURL    string `json:"ticket_url"`
		} `json:"transaction_data"`
	} `json:"point_of_interaction"`
}

// CriarCobranca cria um pagamento PIX no Mercado Pago. A ChaveIdempotencia vai tanto no header
// X-Idempotency-Key quanto em external_reference, garantindo que repeticoes nao gerem cobranca
// nova e que o webhook consiga reassociar o pagamento a intencao de compra.
func (m *MercadoPago) CriarCobranca(
	ctx context.Context,
	solicitacao casosdeuso.SolicitacaoPagamento,
) (casosdeuso.Cobranca, error) {
	email := strings.TrimSpace(solicitacao.EmailPagador)
	if email == "" {
		email = emailPadraoPagador
	}
	corpo, err := json.Marshal(requisicaoPagamento{
		TransactionAmount: float64(solicitacao.ValorCentavos) / 100,
		Description:       "Compra Reveste " + solicitacao.IDCompra,
		PaymentMethodID:   "pix",
		ExternalReference: solicitacao.ChaveIdempotencia,
		NotificationURL:   m.notificacaoURL,
		Payer:             pagador{Email: email},
	})
	if err != nil {
		return casosdeuso.Cobranca{}, fmt.Errorf("mercadopago: montar requisicao: %w", err)
	}

	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, m.urlBase+caminhoPagamentos, bytes.NewReader(corpo))
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

	var pagamento respostaPagamento
	if err := json.Unmarshal(corpoResposta, &pagamento); err != nil {
		return casosdeuso.Cobranca{}, fmt.Errorf("mercadopago: decodificar resposta: %w", err)
	}

	cobranca := casosdeuso.Cobranca{
		Status:               mapearStatus(pagamento.Status),
		Provedor:             provedorMercadoPago,
		IdentificadorExterno: pagamento.ID.String(),
	}
	if cobranca.Status == casosdeuso.CobrancaPendente {
		dados := pagamento.PointOfInteraction.TransactionData
		cobranca.Instrucoes = casosdeuso.InstrucoesPagamento{
			Tipo:         "pix",
			PixCopiaCola: dados.QRCode,
			PixQRCode:    dados.QRCodeBase64,
			URLPagamento: dados.TicketURL,
		}
	}
	return cobranca, nil
}

// Interpretar valida a assinatura da notificacao, consulta o pagamento referenciado e devolve
// a chave de idempotencia (external_reference), o identificador externo e o status atual.
func (m *MercadoPago) Interpretar(
	r *http.Request,
) (chave, provedor, idExterno string, status casosdeuso.StatusCobranca, err error) {
	idPagamento := idPagamentoDaNotificacao(r)
	if idPagamento == "" {
		return "", "", "", "", fmt.Errorf("mercadopago: notificacao sem id de pagamento")
	}
	if !m.assinaturaValida(r, idPagamento) {
		return "", "", "", "", fmt.Errorf("mercadopago: assinatura do webhook invalida")
	}

	pagamento, err := m.consultarPagamento(r.Context(), idPagamento)
	if err != nil {
		return "", "", "", "", err
	}
	return pagamento.ExternalReference, provedorMercadoPago, pagamento.ID.String(), mapearStatus(pagamento.Status), nil
}

func (m *MercadoPago) consultarPagamento(ctx context.Context, id string) (respostaPagamento, error) {
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodGet, m.urlBase+caminhoPagamentos+"/"+id, nil)
	if err != nil {
		return respostaPagamento{}, fmt.Errorf("mercadopago: criar consulta: %w", err)
	}
	requisicao.Header.Set("Accept", "application/json")
	requisicao.Header.Set("Authorization", "Bearer "+m.accessToken)

	resposta, err := m.cliente.Do(requisicao)
	if err != nil {
		return respostaPagamento{}, fmt.Errorf("mercadopago: consultar pagamento: %w", err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		return respostaPagamento{}, fmt.Errorf("mercadopago: consulta status %d", resposta.StatusCode)
	}
	var pagamento respostaPagamento
	if err := json.NewDecoder(io.LimitReader(resposta.Body, 1<<20)).Decode(&pagamento); err != nil {
		return respostaPagamento{}, fmt.Errorf("mercadopago: decodificar consulta: %w", err)
	}
	return pagamento, nil
}

// assinaturaValida confere o header x-signature do Mercado Pago. A assinatura cobre o manifesto
// "id:<data.id>;request-id:<x-request-id>;ts:<ts>;", com HMAC-SHA256 sobre a webhookSecret. Sem
// secret configurado, rejeita (fail-closed).
func (m *MercadoPago) assinaturaValida(r *http.Request, idPagamento string) bool {
	if m.webhookSecret == "" {
		return false
	}
	ts, v1 := partesAssinatura(r.Header.Get("x-signature"))
	if ts == "" || v1 == "" {
		return false
	}
	manifesto := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", strings.ToLower(idPagamento), r.Header.Get("x-request-id"), ts)
	mac := hmac.New(sha256.New, []byte(m.webhookSecret))
	mac.Write([]byte(manifesto))
	esperado := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(esperado), []byte(v1))
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

// idPagamentoDaNotificacao extrai o id do pagamento da notificacao, aceitando o formato de
// query (?data.id=... ou ?id=...) e o corpo JSON ({"data":{"id":...}}).
func idPagamentoDaNotificacao(r *http.Request) string {
	if id := r.URL.Query().Get("data.id"); id != "" {
		return id
	}
	if id := r.URL.Query().Get("id"); id != "" {
		return id
	}
	var corpo struct {
		Data struct {
			ID json.Number `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&corpo); err != nil {
		return ""
	}
	return corpo.Data.ID.String()
}

// mapearStatus traduz o status do Mercado Pago para o StatusCobranca do dominio.
func mapearStatus(status string) casosdeuso.StatusCobranca {
	switch status {
	case "approved":
		return casosdeuso.CobrancaAprovada
	case "rejected", "cancelled":
		return casosdeuso.CobrancaRecusada
	default: // pending, in_process, authorized, in_mediation: aguarda confirmacao
		return casosdeuso.CobrancaPendente
	}
}
