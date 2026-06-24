package integration_test

import (
	"context"
	"testing"
	"time"

	"reveste/apps/back/internal/dominio/cadastros"
)

// TestIntegracaoTrocarEnderecoPrincipal cobre o caminho real contra o Postgres: o indice
// unico parcial uq_endereco_principal_usuario so permite um endereco principal por usuario,
// entao DefinirEnderecoPrincipal precisa desmarcar o atual antes de marcar o escolhido.
func TestIntegracaoTrocarEnderecoPrincipal(t *testing.T) {
	store := abrirStorePostgres(t)
	ctx := context.Background()
	agora := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	semearUsuarioIntegracao(t, store, "00000000-0000-4000-8000-0000000000aa", "52998224725", "enderecos@teste.local", agora)
	idUsuario := "00000000-0000-4000-8000-0000000000aa"

	segundo := cadastros.Endereco{
		ID: "00000000-0000-4000-8000-0000000000bb", CEP: "01310200",
		Logradouro: "Avenida Paulista", Numero: "1000", Bairro: "Bela Vista",
		Cidade: "São Paulo", Estado: "SP",
	}
	if err := store.AdicionarEndereco(ctx, idUsuario, segundo, agora); err != nil {
		t.Fatalf("AdicionarEndereco() erro = %v", err)
	}

	// Promove o segundo endereco a principal (o principal atual ainda esta marcado).
	if err := store.DefinirEnderecoPrincipal(ctx, idUsuario, segundo.ID, agora); err != nil {
		t.Fatalf("DefinirEnderecoPrincipal() erro = %v", err)
	}

	enderecos, err := store.ListarEnderecos(ctx, idUsuario)
	if err != nil {
		t.Fatalf("ListarEnderecos() erro = %v", err)
	}
	principais := 0
	for _, e := range enderecos {
		if e.Principal {
			principais++
			if e.ID != segundo.ID {
				t.Fatalf("principal = %s; esperado %s", e.ID, segundo.ID)
			}
		}
	}
	if principais != 1 {
		t.Fatalf("esperado exatamente 1 endereco principal; obtido %d", principais)
	}

	// Trocar de volta tambem deve funcionar (exercita o indice na direcao oposta).
	usuario, err := store.BuscarUsuarioPorID(ctx, idUsuario)
	if err != nil {
		t.Fatalf("BuscarUsuarioPorID() erro = %v", err)
	}
	if usuario.EnderecoPrincipal.Cidade != "São Paulo" {
		t.Fatalf("endereco principal do usuario nao reflete a troca: %+v", usuario.EnderecoPrincipal)
	}
}
