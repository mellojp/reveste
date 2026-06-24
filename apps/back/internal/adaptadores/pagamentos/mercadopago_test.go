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
		TransactionAmount float64 `json:"transaction_amount"`
		PaymentMethodID   string  `json:"payment_method_id"`
		ExternalReference string  `json:"external_reference"`
		Payer             struct {
			Email string `json:"email"`
		} `json:"payer"`
	}
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != caminhoPagamentos {
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
			"id": 987654321,
			"status": "pending",
			"external_reference": "chave-123",
			"point_of_interaction": {"transaction_data": {
				"qr_code": "00020126-copia-e-cola",
				"qr_code_base64": "iVBORw0KGgo=",
				"ticket_url": "https://mp/ticket/987"
			}}
		}`))
	}))
	defer servidor.Close()

	mp := NovoMercadoPago(servidor.URL, "token-teste", "segredo", "https://reveste.app/webhooks/pagamento")
	cobranca, err := mp.CriarCobranca(context.Background(), casosdeuso.SolicitacaoPagamento{
		IDCompra:          "compra-1",
		ValorCentavos:     11990,
		ChaveIdempotencia: "chave-123",
		EmailPagador:      "comprador@teste.local",
	})
	if err != nil {
		t.Fatalf("CriarCobranca() erro = %v", err)
	}
	if cobranca.Status != casosdeuso.CobrancaPendente {
		t.Fatalf("status = %v; esperado pendente", cobranca.Status)
	}
	if cobranca.IdentificadorExterno != "987654321" {
		t.Fatalf("idExterno = %q; esperado 987654321", cobranca.IdentificadorExterno)
	}
	if cobranca.Instrucoes.Tipo != "pix" || cobranca.Instrucoes.PixCopiaCola != "00020126-copia-e-cola" {
		t.Fatalf("instrucoes inesperadas: %+v", cobranca.Instrucoes)
	}
	if recebido.TransactionAmount != 119.90 || recebido.PaymentMethodID != "pix" {
		t.Fatalf("requisicao enviada inesperada: %+v", recebido)
	}
	if recebido.ExternalReference != "chave-123" || recebido.Payer.Email != "comprador@teste.local" {
		t.Fatalf("external_reference/payer inesperados: %+v", recebido)
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
		_, _ = w.Write([]byte(`{"id":1,"status":"pending","point_of_interaction":{"transaction_data":{"qr_code":"x"}}}`))
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
	const idPagamento = "111222333"
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != caminhoPagamentos+"/"+idPagamento {
			t.Errorf("consulta inesperada: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":111222333,"status":"approved","external_reference":"chave-abc"}`))
	}))
	defer servidor.Close()

	mp := NovoMercadoPago(servidor.URL, "token", "segredo", "")
	req := requisicaoWebhook(idPagamento, "segredo")

	chave, provedor, idExterno, status, err := mp.Interpretar(req)
	if err != nil {
		t.Fatalf("Interpretar() erro = %v", err)
	}
	if chave != "chave-abc" || provedor != provedorMercadoPago || idExterno != idPagamento {
		t.Fatalf("retorno inesperado: chave=%q provedor=%q idExterno=%q", chave, provedor, idExterno)
	}
	if status != casosdeuso.CobrancaAprovada {
		t.Fatalf("status = %v; esperado aprovada", status)
	}
}

func TestMercadoPagoInterpretarRejeitaAssinaturaInvalida(t *testing.T) {
	mp := NovoMercadoPago("http://nao-deve-chamar", "token", "segredo", "")
	req := requisicaoWebhook("123", "segredo-errado")
	if _, _, _, _, err := mp.Interpretar(req); err == nil {
		t.Fatal("Interpretar() deveria falhar com assinatura invalida")
	}
}

// requisicaoWebhook monta uma notificacao com x-signature valida para o segredo informado.
func requisicaoWebhook(idPagamento, segredo string) *http.Request {
	const (
		ts        = "1700000000"
		requestID = "req-xyz"
	)
	manifesto := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", strings.ToLower(idPagamento), requestID, ts)
	mac := hmac.New(sha256.New, []byte(segredo))
	mac.Write([]byte(manifesto))
	v1 := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/pagamento?data.id="+idPagamento, nil)
	req.Header.Set("x-signature", "ts="+ts+",v1="+v1)
	req.Header.Set("x-request-id", requestID)
	return req
}
