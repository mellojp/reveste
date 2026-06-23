package http

import (
	nethttp "net/http"
)

func (a *API) registrarRotasCEP(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /v1/cep/{cep}", a.consultarCEP)
}

// consultarCEP devolve o endereco parcial de um CEP para preencher formularios. E publico:
// expoe apenas dados de logradouro publicos e nao depende de sessao.
func (a *API) consultarCEP(w nethttp.ResponseWriter, r *nethttp.Request) {
	endereco, err := a.cep.Consultar(r.Context(), r.PathValue("cep"))
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, endereco)
}
