package http

import (
	nethttp "net/http"

	"reveste/apps/back/internal/dominio/cadastros"
)

func (a *API) registrarRotasEnderecos(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /v1/me/enderecos", a.autenticado(a.listarEnderecos))
	mux.HandleFunc("POST /v1/me/enderecos", a.autenticado(a.adicionarEndereco))
	mux.HandleFunc("PATCH /v1/me/enderecos/{idEndereco}", a.autenticado(a.atualizarEndereco))
	mux.HandleFunc("DELETE /v1/me/enderecos/{idEndereco}", a.autenticado(a.removerEndereco))
	mux.HandleFunc("POST /v1/me/enderecos/{idEndereco}/principal", a.autenticado(a.definirEnderecoPrincipal))
}

func (a *API) listarEnderecos(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	enderecos, err := a.cadastros.ListarEnderecos(r.Context(), idUsuario)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, map[string]any{"enderecos": enderecos})
}

func (a *API) adicionarEndereco(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	var entrada cadastros.Endereco
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	endereco, err := a.cadastros.AdicionarEndereco(r.Context(), idUsuario, entrada)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusCreated, endereco)
}

func (a *API) atualizarEndereco(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	var entrada cadastros.Endereco
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	if err := a.cadastros.AtualizarEndereco(r.Context(), idUsuario, r.PathValue("idEndereco"), entrada); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (a *API) removerEndereco(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	if err := a.cadastros.RemoverEndereco(r.Context(), idUsuario, r.PathValue("idEndereco")); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (a *API) definirEnderecoPrincipal(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	if err := a.cadastros.DefinirEnderecoPrincipal(r.Context(), idUsuario, r.PathValue("idEndereco")); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}
