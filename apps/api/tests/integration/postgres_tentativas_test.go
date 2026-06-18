package integration_test

import (
	"context"
	"testing"

	"reveste/apps/api/internal/transporte"
)

func TestIntegracaoLimiteLoginPersistente(t *testing.T) {
	store := abrirStorePostgres(t)
	ctx := context.Background()
	limitador := transporte.NovoLimitadorLogin(store)
	const chave = "203.0.113.9"

	for i := 0; i < 5; i++ {
		if !limitador.Permitido(ctx, chave) {
			t.Fatalf("tentativa %d deveria ser permitida", i)
		}
		limitador.RegistrarFalha(ctx, chave)
	}
	if limitador.Permitido(ctx, chave) {
		t.Fatal("deveria bloquear após 5 tentativas persistidas")
	}

	// Outra chave (IP) não é afetada.
	if !limitador.Permitido(ctx, "198.51.100.1") {
		t.Fatal("outro IP não deveria estar bloqueado")
	}

	limitador.Limpar(ctx, chave)
	if !limitador.Permitido(ctx, chave) {
		t.Fatal("deveria liberar após limpar as tentativas")
	}
}
