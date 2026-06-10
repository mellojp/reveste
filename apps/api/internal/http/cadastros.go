package http

import (
	nethttp "net/http"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/dominio/cadastros"
)

func (a *API) registrarRotasCadastros(mux *nethttp.ServeMux) {
	mux.HandleFunc("POST /v1/usuarios", a.cadastrarUsuario)
	mux.HandleFunc("POST /v1/sessoes", a.autenticar)
	mux.HandleFunc("DELETE /v1/sessoes/atual", a.autenticado(a.encerrarSessao))
}

func (a *API) cadastrarUsuario(w nethttp.ResponseWriter, r *nethttp.Request) {
	var entrada struct {
		Nome     string             `json:"nome"`
		CPF      string             `json:"cpf"`
		Email    string             `json:"email"`
		Senha    string             `json:"senha"`
		Telefone string             `json:"telefone"`
		Endereco cadastros.Endereco `json:"endereco"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	usuario, err := a.cadastros.CadastrarUsuario(r.Context(), casosdeuso.EntradaCadastro{
		Nome: entrada.Nome, CPF: entrada.CPF, Email: entrada.Email, Senha: entrada.Senha,
		Telefone: entrada.Telefone, Endereco: entrada.Endereco,
	})
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusCreated, usuario)
}

func (a *API) autenticar(w nethttp.ResponseWriter, r *nethttp.Request) {
	var entrada struct {
		Identificador string `json:"identificador"`
		Senha         string `json:"senha"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	if !a.loginPermitido(r) {
		escreverJSON(w, nethttp.StatusTooManyRequests, erroResposta{
			Codigo: "MUITAS_TENTATIVAS", Mensagem: "Tente autenticar novamente mais tarde.",
			Campos: map[string]string{},
		})
		return
	}
	sessao, err := a.cadastros.Autenticar(r.Context(), entrada.Identificador, entrada.Senha)
	if err != nil {
		a.registrarFalhaLogin(r)
		a.escreverErro(w, err)
		return
	}
	a.limparFalhasLogin(r)
	escreverJSON(w, nethttp.StatusCreated, sessao)
}

func (a *API) encerrarSessao(w nethttp.ResponseWriter, r *nethttp.Request, _ string, token string) {
	if err := a.cadastros.EncerrarSessao(r.Context(), token); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}
