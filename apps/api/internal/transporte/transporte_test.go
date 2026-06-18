package transporte_test

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"reveste/apps/api/internal/transporte"
)

func TestHTTPSConfiaProxySomenteQuandoHabilitado(t *testing.T) {
	r := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	if transporte.HTTPS(r, false) {
		t.Fatal("não deveria confiar em X-Forwarded-Proto sem proxy confiável")
	}
	if !transporte.HTTPS(r, true) {
		t.Fatal("deveria reconhecer HTTPS via X-Forwarded-Proto com proxy confiável")
	}
}

func TestEnderecoClienteUsaForwardedSomenteComProxy(t *testing.T) {
	r := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	if ip := transporte.EnderecoCliente(r, false); ip != "10.0.0.1" {
		t.Fatalf("sem proxy: IP = %q; esperado 10.0.0.1", ip)
	}
	if ip := transporte.EnderecoCliente(r, true); ip != "203.0.113.5" {
		t.Fatalf("com proxy: IP = %q; esperado 203.0.113.5", ip)
	}
}

func TestOrigemPermitidaComparaEsquemaEHost(t *testing.T) {
	r := httptest.NewRequest(nethttp.MethodPost, "http://exemplo.local/entrar", nil)
	r.Host = "exemplo.local"
	r.Header.Set("Origin", "http://exemplo.local")
	if !transporte.OrigemPermitida(r, false) {
		t.Fatal("mesma origem deveria ser permitida")
	}
	r.Header.Set("Origin", "https://atacante.example")
	if transporte.OrigemPermitida(r, false) {
		t.Fatal("origem externa deveria ser rejeitada")
	}
}

func TestLimitadorBloqueiaAposLimite(t *testing.T) {
	limitador := transporte.NovoLimitadorLogin(transporte.NovoRegistroMemoria())
	ctx := context.Background()
	const chave = "203.0.113.9"
	for i := 0; i < 5; i++ {
		if !limitador.Permitido(ctx, chave) {
			t.Fatalf("tentativa %d deveria ser permitida", i)
		}
		limitador.RegistrarFalha(ctx, chave)
	}
	if limitador.Permitido(ctx, chave) {
		t.Fatal("deveria bloquear após 5 tentativas")
	}
	limitador.Limpar(ctx, chave)
	if !limitador.Permitido(ctx, chave) {
		t.Fatal("deveria liberar após limpar as tentativas")
	}
}
