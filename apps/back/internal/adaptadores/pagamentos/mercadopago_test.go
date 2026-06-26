package pagamentos

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reveste/apps/back/internal/casosdeuso"
)

func TestMercadoPagoCriarCobrancaPendenteComPix(t *testing.T) {
	var recebido struct {
		Type           string `json:"type"`
		ProcessingMode string `json:"processing_mode"`
		TotalAmount    string `json:"total_amount"`
		ExternalRef    string `json:"external_reference"`
		Transactions   struct {
			Payments []struct {
				Amount        string `json:"amount"`
				PaymentMethod struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"payment_method"`
			} `json:"payments"`
		} `json:"transactions"`
		Payer struct {
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"payer"`
	}
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != caminhoOrders {
			t.Errorf("requisicao inesperada: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token-teste" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Idempotency-Key") != "chave-123" {
			t.Errorf("X-Idempotency-Key = %q", r.Header.Get("X-Idempotency-Key"))
		}
		_ = json.NewDecoder(r.Body).Decode(&recebido)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "ORD01ABCXYZ",
			"status": "action_required",
			"status_detail": "waiting_transfer",
			"external_reference": "chave-123",
			"transactions": {"payments": [{
				"id": "PAY01ABCXYZ",
				"status": "action_required",
				"payment_method": {
					"id": "pix",
					"type": "bank_transfer",
					"qr_code": "00020126-copia-e-cola",
					"qr_code_base64": "iVBORw0KGgo=",
					"ticket_url": "https://mp/ticket/987"
				}
			}]}
		}`))
	}))
	defer servidor.Close()

	mp := NovoMercadoPago(servidor.URL, "token-teste", "segredo", "https://reveste.app/webhooks/pagamento")
	cobranca, err := mp.CriarCobranca(context.Background(), casosdeuso.SolicitacaoPagamento{
		IDCompra:          "compra-1",
		ValorCentavos:     11990,
		ChaveIdempotencia: "chave-123",
		EmailPagador:      "comprador@teste.local",
		NomePagador:       "APRO Comprador Teste",
	})
	if err != nil {
		t.Fatalf("CriarCobranca() erro = %v", err)
	}
	if cobranca.Status != casosdeuso.CobrancaPendente {
		t.Fatalf("status = %v; esperado pendente", cobranca.Status)
	}
	if cobranca.IdentificadorExterno != "ORD01ABCXYZ" {
		t.Fatalf("idExterno = %q; esperado ORD01ABCXYZ", cobranca.IdentificadorExterno)
	}
	if cobranca.Instrucoes.Tipo != "pix" || cobranca.Instrucoes.PixCopiaCola != "00020126-copia-e-cola" {
		t.Fatalf("instrucoes inesperadas: %+v", cobranca.Instrucoes)
	}
	if recebido.Type != "online" || recebido.ProcessingMode != "automatic" {
		t.Fatalf("type/processing_mode inesperados: %+v", recebido)
	}
	if recebido.TotalAmount != "119.90" {
		t.Fatalf("total_amount = %q; esperado 119.90", recebido.TotalAmount)
	}
	if len(recebido.Transactions.Payments) != 1 {
		t.Fatalf("esperava 1 pagamento, veio %d", len(recebido.Transactions.Payments))
	}
	pagamento := recebido.Transactions.Payments[0]
	if pagamento.Amount != "119.90" || pagamento.PaymentMethod.ID != "pix" || pagamento.PaymentMethod.Type != "bank_transfer" {
		t.Fatalf("pagamento enviado inesperado: %+v", pagamento)
	}
	if recebido.ExternalRef != "chave-123" || recebido.Payer.Email != "comprador@teste.local" {
		t.Fatalf("external_reference/payer inesperados: %+v", recebido)
	}
	if recebido.Payer.FirstName != "APRO" || recebido.Payer.LastName != "Comprador Teste" {
		t.Fatalf("nome do pagador dividido incorretamente: first=%q last=%q", recebido.Payer.FirstName, recebido.Payer.LastName)
	}
}

func TestMercadoPagoCriarCobrancaCartaoAprovado(t *testing.T) {
	var recebido struct {
		Transactions struct {
			Payments []struct {
				PaymentMethod struct {
					ID           string `json:"id"`
					Type         string `json:"type"`
					Token        string `json:"token"`
					Installments int    `json:"installments"`
				} `json:"payment_method"`
			} `json:"payments"`
		} `json:"transactions"`
		Payer struct {
			Identification struct {
				Type   string `json:"type"`
				Number string `json:"number"`
			} `json:"identification"`
		} `json:"payer"`
	}
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&recebido)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"ORD9","status":"processed","status_detail":"accredited","external_reference":"k","transactions":{"payments":[{"id":"PAY9","status":"processed"}]}}`))
	}))
	defer servidor.Close()

	mp := NovoMercadoPago(servidor.URL, "t", "s", "")
	cobranca, err := mp.CriarCobranca(context.Background(), casosdeuso.SolicitacaoPagamento{
		ValorCentavos:     11990,
		ChaveIdempotencia: "k",
		EmailPagador:      "comprador@teste.local",
		NomePagador:       "APRO Teste",
		Cartao: &casosdeuso.DadosCartao{
			Token:           "tok-123",
			MetodoPagamento: "master",
			Parcelas:        3,
			TipoDocumento:   "CPF",
			NumeroDocumento: "12345678909",
		},
	})
	if err != nil {
		t.Fatalf("CriarCobranca() erro = %v", err)
	}
	if cobranca.Status != casosdeuso.CobrancaAprovada {
		t.Fatalf("status = %v; esperado aprovada", cobranca.Status)
	}
	if len(recebido.Transactions.Payments) != 1 {
		t.Fatalf("esperava 1 pagamento, veio %d", len(recebido.Transactions.Payments))
	}
	pm := recebido.Transactions.Payments[0].PaymentMethod
	if pm.ID != "master" || pm.Type != "credit_card" || pm.Token != "tok-123" || pm.Installments != 3 {
		t.Fatalf("payment_method de cartao inesperado: %+v", pm)
	}
	if recebido.Payer.Identification.Type != "CPF" || recebido.Payer.Identification.Number != "12345678909" {
		t.Fatalf("identificacao do pagador inesperada: %+v", recebido.Payer.Identification)
	}
}

func TestMercadoPagoCriarCobrancaUsaEmailPadraoQuandoVazio(t *testing.T) {
	var emailRecebido string
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var corpo struct {
			Payer struct {
				Email string `json:"email"`
			} `json:"payer"`
		}
		_ = json.NewDecoder(r.Body).Decode(&corpo)
		emailRecebido = corpo.Payer.Email
		_, _ = w.Write([]byte(`{"id":"ORD1","status":"action_required","transactions":{"payments":[{"payment_method":{"qr_code":"x"}}]}}`))
	}))
	defer servidor.Close()

	mp := NovoMercadoPago(servidor.URL, "t", "s", "")
	if _, err := mp.CriarCobranca(context.Background(), casosdeuso.SolicitacaoPagamento{ChaveIdempotencia: "k"}); err != nil {
		t.Fatalf("CriarCobranca() erro = %v", err)
	}
	if emailRecebido != emailPadraoPagador {
		t.Fatalf("email = %q; esperado o padrao %q", emailRecebido, emailPadraoPagador)
	}
}

func TestMercadoPagoInterpretarWebhookAprovado(t *testing.T) {
	const idOrder = "ORD01JQ4S4KY8HWQ6NA5PXB65B3D3"
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != caminhoOrders+"/"+idOrder {
			t.Errorf("consulta inesperada: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"` + idOrder + `","status":"processed","status_detail":"accredited","external_reference":"chave-abc"}`))
	}))
	defer servidor.Close()

	mp := NovoMercadoPago(servidor.URL, "token", "segredo", "")
	req := requisicaoWebhook(idOrder, "segredo")

	chave, provedor, idExterno, status, err := mp.Interpretar(req)
	if err != nil {
		t.Fatalf("Interpretar() erro = %v", err)
	}
	if chave != "chave-abc" || provedor != provedorMercadoPago || idExterno != idOrder {
		t.Fatalf("retorno inesperado: chave=%q provedor=%q idExterno=%q", chave, provedor, idExterno)
	}
	if status != casosdeuso.CobrancaAprovada {
		t.Fatalf("status = %v; esperado aprovada", status)
	}
}

func TestMercadoPagoInterpretarRejeitaAssinaturaInvalida(t *testing.T) {
	mp := NovoMercadoPago("http://nao-deve-chamar", "token", "segredo", "")
	req := requisicaoWebhook("ORD123", "segredo-errado")
	if _, _, _, _, err := mp.Interpretar(req); err == nil {
		t.Fatal("Interpretar() deveria falhar com assinatura invalida")
	}
}

func TestMercadoPagoInterpretarWebhookPaymentAprovado(t *testing.T) {
	const idPagamento = "165048251043"
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/payments/"+idPagamento {
			t.Errorf("consulta inesperada: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":165048251043,"status":"approved","external_reference":"chave-pay"}`))
	}))
	defer servidor.Close()

	mp := NovoMercadoPago(servidor.URL, "token", "segredo", "")
	req := requisicaoWebhookTopico(idPagamento, "payment", "segredo")

	chave, provedor, idExterno, status, err := mp.Interpretar(req)
	if err != nil {
		t.Fatalf("Interpretar() erro = %v", err)
	}
	if chave != "chave-pay" || provedor != provedorMercadoPago || idExterno != idPagamento {
		t.Fatalf("retorno inesperado: chave=%q provedor=%q idExterno=%q", chave, provedor, idExterno)
	}
	if status != casosdeuso.CobrancaAprovada {
		t.Fatalf("status = %v; esperado aprovada", status)
	}
}

func TestMercadoPagoInterpretarIgnoraTopicoDesconhecido(t *testing.T) {
	// Topicos irrelevantes (ex.: merchant_order) sao confirmados sem consultar a API nem agir.
	mp := NovoMercadoPago("http://nao-deve-chamar", "token", "segredo", "")
	req := httptest.NewRequest(http.MethodPost, "/webhooks/pagamento?data.id=123&type=merchant_order", nil)
	_, _, _, status, err := mp.Interpretar(req)
	if err != nil {
		t.Fatalf("Interpretar() erro = %v", err)
	}
	if status != casosdeuso.CobrancaPendente {
		t.Fatalf("status = %v; esperado pendente (ignorado)", status)
	}
}

// requisicaoWebhook monta uma notificacao do topico order com x-signature valida para o segredo.
func requisicaoWebhook(idOrder, segredo string) *http.Request {
	return requisicaoWebhookTopico(idOrder, "order", segredo)
}

// requisicaoWebhookTopico monta uma notificacao do topico informado com x-signature valida.
func requisicaoWebhookTopico(idRecurso, tipo, segredo string) *http.Request {
	const (
		ts        = "1700000000"
		requestID = "req-xyz"
	)
	manifesto := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", strings.ToLower(idRecurso), requestID, ts)
	mac := hmac.New(sha256.New, []byte(segredo))
	mac.Write([]byte(manifesto))
	v1 := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/pagamento?data.id="+idRecurso+"&type="+tipo, nil)
	req.Header.Set("x-signature", "ts="+ts+",v1="+v1)
	req.Header.Set("x-request-id", requestID)
	return req
}
