package casosdeuso

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"reveste/apps/api/internal/common"
)

const TamanhoMaximoImagemBytes int64 = 5 * 1024 * 1024

var TiposImagemPermitidos = []string{"image/jpeg", "image/png", "image/webp"}

type ControladorUpload struct {
	arquivos ArmazenamentoArquivos
	ids      GeradorID
	relogio  Relogio
}

type EntradaAutorizacaoUpload struct {
	NomeArquivo string
	Tipo        string
	Tamanho     int64
}

func NovoControladorUpload(
	arquivos ArmazenamentoArquivos,
	ids GeradorID,
	relogio Relogio,
) *ControladorUpload {
	return &ControladorUpload{arquivos: arquivos, ids: ids, relogio: relogio}
}

func (c *ControladorUpload) AutorizarImagemAnuncio(
	ctx context.Context,
	idUsuario string,
	entrada EntradaAutorizacaoUpload,
) (AutorizacaoUpload, error) {
	tipo := strings.ToLower(strings.TrimSpace(entrada.Tipo))
	if !tipoImagemPermitido(tipo) {
		return AutorizacaoUpload{}, common.NovaValidacao(map[string]string{
			"fotos": "Envie apenas imagens JPEG, PNG ou WebP.",
		})
	}
	if entrada.Tamanho <= 0 || entrada.Tamanho > TamanhoMaximoImagemBytes {
		return AutorizacaoUpload{}, common.NovaValidacao(map[string]string{
			"fotos": "Cada imagem deve ter no máximo 5 MB.",
		})
	}

	extensao := extensaoImagem(tipo)
	pathname := "anuncios/" + idUsuario + "/" + c.ids.Novo() + extensao
	return c.arquivos.AutorizarUpload(ctx, SolicitacaoUpload{
		Pathname:           filepath.ToSlash(pathname),
		TiposPermitidos:    TiposImagemPermitidos,
		TamanhoMaximoBytes: TamanhoMaximoImagemBytes,
		ExpiraEm:           c.relogio.Agora().Add(10 * time.Minute),
	})
}

func tipoImagemPermitido(tipo string) bool {
	for _, permitido := range TiposImagemPermitidos {
		if tipo == permitido {
			return true
		}
	}
	return false
}

func extensaoImagem(tipo string) string {
	switch tipo {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}
