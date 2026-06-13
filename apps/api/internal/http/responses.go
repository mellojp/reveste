package http

import (
	"encoding/json"
	"errors"
	"io"
	nethttp "net/http"

	"reveste/apps/api/internal/common"
)

type erroResposta struct {
	Codigo   string            `json:"codigo"`
	Mensagem string            `json:"mensagem"`
	Campos   map[string]string `json:"campos"`
}

func (a *API) escreverErro(w nethttp.ResponseWriter, err error) {
	status := nethttp.StatusInternalServerError
	codigo := "ERRO_INTERNO"
	mensagem := "Ocorreu um erro interno."
	campos := map[string]string{}
	var validacao common.ErroValidacao
	var conflitoCampo common.ErroConflitoCampo
	switch {
	case errors.As(err, &validacao):
		status, codigo, mensagem = nethttp.StatusUnprocessableEntity, "DADOS_INVALIDOS", "Revise os campos destacados."
		campos = validacao.Campos
	case errors.As(err, &conflitoCampo):
		status, codigo, mensagem = nethttp.StatusConflict, "CONFLITO", "Já existe uma conta com os dados destacados."
		campos = conflitoCampo.Campos
	case errors.Is(err, common.ErrDadosInvalidos):
		status, codigo, mensagem = nethttp.StatusUnprocessableEntity, "DADOS_INVALIDOS", "Os dados informados sao invalidos."
	case errors.Is(err, common.ErrNaoEncontrado):
		status, codigo, mensagem = nethttp.StatusNotFound, "NAO_ENCONTRADO", "O recurso solicitado nao foi encontrado."
	case errors.Is(err, common.ErrConflito):
		status, codigo, mensagem = nethttp.StatusConflict, "CONFLITO", "Ja existe um recurso com os dados informados."
	case errors.Is(err, common.ErrNaoAutorizado):
		status, codigo, mensagem = nethttp.StatusUnauthorized, "NAO_AUTORIZADO", "Autenticacao obrigatoria ou invalida."
	case errors.Is(err, common.ErrNaoPermitido):
		status, codigo, mensagem = nethttp.StatusForbidden, "NAO_PERMITIDO", "Você não pode realizar esta operação."
	case errors.Is(err, common.ErrAnuncioIndisponivel):
		status, codigo, mensagem = nethttp.StatusConflict, "ANUNCIO_INDISPONIVEL", err.Error()
	case errors.Is(err, common.ErrAnuncioDoProprioAutor):
		status, codigo, mensagem = nethttp.StatusUnprocessableEntity, "ANUNCIO_PROPRIO", err.Error()
	case errors.Is(err, common.ErrVendedorBloqueado):
		status, codigo, mensagem = nethttp.StatusForbidden, "VENDEDOR_BLOQUEADO", err.Error()
	case errors.Is(err, common.ErrTransicaoInvalida):
		status, codigo, mensagem = nethttp.StatusConflict, "TRANSICAO_INVALIDA", err.Error()
	case errors.Is(err, common.ErrServicoIndisponivel):
		status, codigo, mensagem = nethttp.StatusServiceUnavailable, "SERVICO_INDISPONIVEL", "O armazenamento de imagens não está configurado."
	default:
		a.logger.Error("erro nao tratado", "erro", err)
	}
	escreverJSON(w, status, erroResposta{Codigo: codigo, Mensagem: mensagem, Campos: campos})
}

func decodificarJSON(w nethttp.ResponseWriter, r *nethttp.Request, destino any) bool {
	decodificador := json.NewDecoder(nethttp.MaxBytesReader(w, r.Body, 1<<20))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		escreverJSON(w, nethttp.StatusBadRequest, erroResposta{
			Codigo: "JSON_INVALIDO", Mensagem: "O corpo da requisicao possui JSON invalido.",
			Campos: map[string]string{},
		})
		return false
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		escreverJSON(w, nethttp.StatusBadRequest, erroResposta{
			Codigo: "JSON_INVALIDO", Mensagem: "O corpo deve conter um unico valor JSON.",
			Campos: map[string]string{},
		})
		return false
	}
	return true
}

func escreverJSON(w nethttp.ResponseWriter, status int, valor any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(valor)
}
