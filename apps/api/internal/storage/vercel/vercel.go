package vercel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
)

const urlUpload = "https://vercel.com/api/blob/"

type Storage struct {
	token string
}

func Novo(token string) *Storage {
	return &Storage{token: strings.TrimSpace(token)}
}

func (s *Storage) AutorizarUpload(
	ctx context.Context,
	solicitacao casosdeuso.SolicitacaoUpload,
) (casosdeuso.AutorizacaoUpload, error) {
	if err := ctx.Err(); err != nil {
		return casosdeuso.AutorizacaoUpload{}, err
	}
	partes := strings.Split(s.token, "_")
	if len(partes) < 4 || partes[3] == "" {
		return casosdeuso.AutorizacaoUpload{}, common.ErrServicoIndisponivel
	}

	payload, err := json.Marshal(map[string]any{
		"pathname":            solicitacao.Pathname,
		"allowedContentTypes": solicitacao.TiposPermitidos,
		"maximumSizeInBytes":  solicitacao.TamanhoMaximoBytes,
		"addRandomSuffix":     false,
		"allowOverwrite":      false,
		"validUntil":          solicitacao.ExpiraEm.UnixMilli(),
	})
	if err != nil {
		return casosdeuso.AutorizacaoUpload{}, err
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payload)
	assinatura := hmac.New(sha256.New, []byte(s.token))
	_, _ = assinatura.Write([]byte(payloadBase64))
	conteudoAssinado := hex.EncodeToString(assinatura.Sum(nil)) + "." + payloadBase64
	tokenCliente := "vercel_blob_client_" + partes[3] + "_" +
		base64.StdEncoding.EncodeToString([]byte(conteudoAssinado))

	return casosdeuso.AutorizacaoUpload{
		URLUpload:          urlUpload,
		Pathname:           solicitacao.Pathname,
		Token:              tokenCliente,
		TiposAceitos:       append([]string(nil), solicitacao.TiposPermitidos...),
		TamanhoMaximoBytes: solicitacao.TamanhoMaximoBytes,
	}, nil
}

var _ casosdeuso.ArmazenamentoArquivos = (*Storage)(nil)
