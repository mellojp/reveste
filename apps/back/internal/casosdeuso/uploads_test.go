package casosdeuso_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/common"
)

type armazenamentoUploadFake struct {
	solicitacao casosdeuso.SolicitacaoUpload
}

func (a *armazenamentoUploadFake) AutorizarUpload(
	_ context.Context,
	solicitacao casosdeuso.SolicitacaoUpload,
) (casosdeuso.AutorizacaoUpload, error) {
	a.solicitacao = solicitacao
	return casosdeuso.AutorizacaoUpload{Pathname: solicitacao.Pathname}, nil
}

func TestAutorizarImagemValidaEscopoDoUpload(t *testing.T) {
	armazenamento := &armazenamentoUploadFake{}
	agora := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	controlador := casosdeuso.NovoControladorUpload(
		armazenamento,
		&geradorSequencial{},
		relogioFixo{agora: agora},
	)

	autorizacao, err := controlador.AutorizarImagemAnuncio(
		context.Background(),
		"usuario-1",
		casosdeuso.EntradaAutorizacaoUpload{
			NomeArquivo: "foto.png",
			Tipo:        "image/png",
			Tamanho:     1024,
		},
	)

	if err != nil {
		t.Fatalf("AutorizarImagemAnuncio() erro = %v", err)
	}
	if autorizacao.Pathname == "" || armazenamento.solicitacao.Pathname == "" {
		t.Fatal("pathname do upload nao foi definido")
	}
	if armazenamento.solicitacao.TamanhoMaximoBytes != casosdeuso.TamanhoMaximoImagemBytes {
		t.Fatalf("limite = %d", armazenamento.solicitacao.TamanhoMaximoBytes)
	}
	if !armazenamento.solicitacao.ExpiraEm.Equal(agora.Add(10 * time.Minute)) {
		t.Fatalf("expiracao = %v", armazenamento.solicitacao.ExpiraEm)
	}
}

func TestAutorizarImagemRejeitaTipoETamanhoInvalidos(t *testing.T) {
	controlador := casosdeuso.NovoControladorUpload(
		&armazenamentoUploadFake{},
		&geradorSequencial{},
		relogioFixo{agora: time.Now()},
	)

	_, err := controlador.AutorizarImagemAnuncio(
		context.Background(),
		"usuario-1",
		casosdeuso.EntradaAutorizacaoUpload{Tipo: "application/pdf", Tamanho: 100},
	)
	if !errors.Is(err, common.ErrDadosInvalidos) {
		t.Fatalf("tipo invalido retornou %v", err)
	}

	_, err = controlador.AutorizarImagemAnuncio(
		context.Background(),
		"usuario-1",
		casosdeuso.EntradaAutorizacaoUpload{
			Tipo:    "image/jpeg",
			Tamanho: casosdeuso.TamanhoMaximoImagemBytes + 1,
		},
	)
	if !errors.Is(err, common.ErrDadosInvalidos) {
		t.Fatalf("tamanho invalido retornou %v", err)
	}
}
