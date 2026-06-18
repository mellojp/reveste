package web

import (
	nethttp "net/http"

	"reveste/apps/api/internal/transporte"
)

func (a *AdaptadorPaginas) autenticacaoPermitida(r *nethttp.Request) bool {
	return a.limitador.Permitido(r.Context(), transporte.EnderecoCliente(r, a.confiarProxy))
}

func (a *AdaptadorPaginas) registrarFalhaAutenticacao(r *nethttp.Request) {
	a.limitador.RegistrarFalha(r.Context(), transporte.EnderecoCliente(r, a.confiarProxy))
}

func (a *AdaptadorPaginas) limparFalhasAutenticacao(r *nethttp.Request) {
	a.limitador.Limpar(r.Context(), transporte.EnderecoCliente(r, a.confiarProxy))
}
