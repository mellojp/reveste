package vercel

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"reveste/apps/back/internal/casosdeuso"
)

func TestAutorizarUploadGeraTokenRestrito(t *testing.T) {
	storage := Novo("vercel_blob_rw_store123_segredo")
	expiraEm := time.Date(2026, 6, 12, 12, 10, 0, 0, time.UTC)

	autorizacao, err := storage.AutorizarUpload(context.Background(), casosdeuso.SolicitacaoUpload{
		Pathname:           "anuncios/usuario/foto.jpg",
		TiposPermitidos:    []string{"image/jpeg"},
		TamanhoMaximoBytes: 5 * 1024 * 1024,
		ExpiraEm:           expiraEm,
	})
	if err != nil {
		t.Fatalf("AutorizarUpload() erro = %v", err)
	}
	if !strings.HasPrefix(autorizacao.Token, "vercel_blob_client_store123_") {
		t.Fatalf("token inesperado: %s", autorizacao.Token)
	}

	partes := strings.Split(autorizacao.Token, "_")
	conteudo, err := base64.StdEncoding.DecodeString(partes[4])
	if err != nil {
		t.Fatalf("decodificar token: %v", err)
	}
	assinaturaEPayload := strings.SplitN(string(conteudo), ".", 2)
	payload, err := base64.StdEncoding.DecodeString(assinaturaEPayload[1])
	if err != nil {
		t.Fatalf("decodificar payload: %v", err)
	}
	var dados map[string]any
	if err := json.Unmarshal(payload, &dados); err != nil {
		t.Fatalf("decodificar JSON: %v", err)
	}
	if dados["pathname"] != "anuncios/usuario/foto.jpg" {
		t.Fatalf("pathname = %v", dados["pathname"])
	}
	if dados["maximumSizeInBytes"] != float64(5*1024*1024) {
		t.Fatalf("maximumSizeInBytes = %v", dados["maximumSizeInBytes"])
	}
	if dados["validUntil"] != float64(expiraEm.UnixMilli()) {
		t.Fatalf("validUntil = %v", dados["validUntil"])
	}
}
