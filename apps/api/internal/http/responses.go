package http

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"strconv"

	errosdominio "reveste/apps/api/internal/dominio/erros"
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
	switch {
	case errors.Is(err, errosdominio.ErrDadosInvalidos):
		status, codigo, mensagem = nethttp.StatusUnprocessableEntity, "DADOS_INVALIDOS", "Os dados informados sao invalidos."
	case errors.Is(err, errosdominio.ErrNaoEncontrado):
		status, codigo, mensagem = nethttp.StatusNotFound, "NAO_ENCONTRADO", "O recurso solicitado nao foi encontrado."
	case errors.Is(err, errosdominio.ErrConflito):
		status, codigo, mensagem = nethttp.StatusConflict, "CONFLITO", "Ja existe um recurso com os dados informados."
	case errors.Is(err, errosdominio.ErrNaoAutorizado):
		status, codigo, mensagem = nethttp.StatusUnauthorized, "NAO_AUTORIZADO", "Autenticacao obrigatoria ou invalida."
	case errors.Is(err, errosdominio.ErrAnuncioIndisponivel):
		status, codigo, mensagem = nethttp.StatusConflict, "ANUNCIO_INDISPONIVEL", err.Error()
	case errors.Is(err, errosdominio.ErrAnuncioDoProprioAutor):
		status, codigo, mensagem = nethttp.StatusUnprocessableEntity, "ANUNCIO_PROPRIO", err.Error()
	default:
		a.logger.Error("erro nao tratado", "erro", err)
	}
	escreverJSON(w, status, erroResposta{Codigo: codigo, Mensagem: mensagem, Campos: map[string]string{}})
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
	return true
}

func escreverJSON(w nethttp.ResponseWriter, status int, valor any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(valor)
}

func inteiro(valor string) int {
	numero, _ := strconv.Atoi(valor)
	return numero
}

func inteiro64(valor string) int64 {
	numero, _ := strconv.ParseInt(valor, 10, 64)
	return numero
}
