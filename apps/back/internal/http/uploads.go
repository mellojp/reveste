package http

import (
	nethttp "net/http"

	"reveste/apps/back/internal/casosdeuso"
)

func (a *API) registrarRotasUploads(mux *nethttp.ServeMux) {
	mux.HandleFunc(
		"POST /v1/uploads/imagens/autorizacoes",
		a.autenticado(a.autorizarUploadImagem),
	)
}

func (a *API) autorizarUploadImagem(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	idUsuario,
	_ string,
) {
	var entrada struct {
		NomeArquivo string `json:"nome_arquivo"`
		Tipo        string `json:"tipo"`
		Tamanho     int64  `json:"tamanho"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	autorizacao, err := a.uploads.AutorizarImagemAnuncio(
		r.Context(),
		idUsuario,
		casosdeuso.EntradaAutorizacaoUpload{
			NomeArquivo: entrada.NomeArquivo,
			Tipo:        entrada.Tipo,
			Tamanho:     entrada.Tamanho,
		},
	)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusCreated, autorizacao)
}
