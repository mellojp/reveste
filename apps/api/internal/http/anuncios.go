package http

import (
	nethttp "net/http"
	"net/url"
	"strconv"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
)

func (a *API) registrarRotasAnuncios(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /v1/anuncios", a.listarAnuncios)
	mux.HandleFunc("GET /v1/anuncios/{idAnuncio}", a.obterAnuncio)
	mux.HandleFunc("POST /v1/anuncios", a.autenticado(a.criarAnuncio))
	mux.HandleFunc("GET /v1/me/anuncios", a.autenticado(a.listarMeusAnuncios))
	mux.HandleFunc("PATCH /v1/me/anuncios/{idAnuncio}", a.autenticado(a.atualizarAnuncio))
	mux.HandleFunc("DELETE /v1/me/anuncios/{idAnuncio}", a.autenticado(a.excluirAnuncio))
	mux.HandleFunc("GET /v1/vendedores/{idVendedor}", a.obterPerfilPublicoVendedor)
}

func (a *API) atualizarAnuncio(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	idUsuario,
	_ string,
) {
	entrada, ok := decodificarEntradaAnuncio(w, r)
	if !ok {
		return
	}
	anuncio, err := a.anuncios.AtualizarAnuncio(
		r.Context(), idUsuario, r.PathValue("idAnuncio"), entrada,
	)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, anuncio)
}

func (a *API) excluirAnuncio(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	idUsuario,
	_ string,
) {
	if err := a.anuncios.ExcluirAnuncio(
		r.Context(), idUsuario, r.PathValue("idAnuncio"),
	); err != nil {
		a.escreverErro(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (a *API) obterPerfilPublicoVendedor(w nethttp.ResponseWriter, r *nethttp.Request) {
	perfil, err := a.anuncios.ObterPerfilPublicoVendedor(
		r.Context(), r.PathValue("idVendedor"),
	)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, perfil)
}

func (a *API) criarAnuncio(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	entrada, ok := decodificarEntradaAnuncio(w, r)
	if !ok {
		return
	}
	anuncio, err := a.anuncios.CriarAnuncio(r.Context(), idUsuario, entrada)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusCreated, anuncio)
}

func decodificarEntradaAnuncio(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) (casosdeuso.EntradaAnuncio, bool) {
	var entrada struct {
		Titulo            string                     `json:"titulo"`
		Descricao         string                     `json:"descricao"`
		Categoria         string                     `json:"categoria"`
		Tamanho           string                     `json:"tamanho"`
		Cor               string                     `json:"cor"`
		EstadoConservacao anuncios.EstadoConservacao `json:"estado_conservacao"`
		PrecoCentavos     int64                      `json:"preco_centavos"`
		PesoGramas        int                        `json:"peso_gramas"`
		AlturaCm          int                        `json:"altura_cm"`
		LarguraCm         int                        `json:"largura_cm"`
		ComprimentoCm     int                        `json:"comprimento_cm"`
		URLsFotos         []string                   `json:"urls_fotos"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return casosdeuso.EntradaAnuncio{}, false
	}
	return casosdeuso.EntradaAnuncio{
		Titulo: entrada.Titulo, Descricao: entrada.Descricao, Categoria: entrada.Categoria,
		Tamanho: entrada.Tamanho, Cor: entrada.Cor, EstadoConservacao: entrada.EstadoConservacao,
		PrecoCentavos: entrada.PrecoCentavos,
		PesoGramas:    entrada.PesoGramas, AlturaCm: entrada.AlturaCm,
		LarguraCm: entrada.LarguraCm, ComprimentoCm: entrada.ComprimentoCm,
		URLsFotos: entrada.URLsFotos,
	}, true
}

func (a *API) listarAnuncios(w nethttp.ResponseWriter, r *nethttp.Request) {
	filtro, err := filtroAnuncios(r.URL.Query())
	if err != nil {
		escreverJSON(w, nethttp.StatusBadRequest, erroResposta{
			Codigo: "FILTRO_INVALIDO", Mensagem: "Os filtros informados sao invalidos.",
			Campos: map[string]string{},
		})
		return
	}
	if token, porCookie := tokenDaRequisicao(r); token != "" {
		idUsuario, err := a.cadastros.IdentificarUsuario(r.Context(), token)
		if err != nil {
			if porCookie {
				a.removerCookieSessao(w, r)
			}
			a.escreverErro(w, err)
			return
		}
		filtro.ExcluirVendedor = idUsuario
	}
	lista, err := a.anuncios.ListarAnuncios(r.Context(), filtro)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	if lista == nil {
		lista = []anuncios.Anuncio{}
	}
	escreverJSON(w, nethttp.StatusOK, map[string]any{"dados": lista, "quantidade": len(lista)})
}

func (a *API) obterAnuncio(w nethttp.ResponseWriter, r *nethttp.Request) {
	anuncio, err := a.anuncios.ObterAnuncio(r.Context(), r.PathValue("idAnuncio"))
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, anuncio)
}

func (a *API) listarMeusAnuncios(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	idUsuario,
	_ string,
) {
	lista, err := a.anuncios.ListarAnunciosDoVendedor(r.Context(), idUsuario)
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	if lista == nil {
		lista = []anuncios.Anuncio{}
	}
	escreverJSON(w, nethttp.StatusOK, map[string]any{"dados": lista, "quantidade": len(lista)})
}

func filtroAnuncios(consulta url.Values) (casosdeuso.FiltroAnuncios, error) {
	precoMinimo, err := inteiro64NaoNegativo(consulta.Get("preco_min_centavos"))
	if err != nil {
		return casosdeuso.FiltroAnuncios{}, err
	}
	precoMaximo, err := inteiro64NaoNegativo(consulta.Get("preco_max_centavos"))
	if err != nil || precoMaximo > 0 && precoMinimo > precoMaximo {
		return casosdeuso.FiltroAnuncios{}, common.ErrDadosInvalidos
	}
	limite, err := inteiroNaoNegativo(consulta.Get("limite"))
	if err != nil || limite > 50 {
		return casosdeuso.FiltroAnuncios{}, common.ErrDadosInvalidos
	}
	deslocamento, err := inteiroNaoNegativo(consulta.Get("deslocamento"))
	if err != nil {
		return casosdeuso.FiltroAnuncios{}, err
	}
	estado := anuncios.EstadoConservacao(consulta.Get("estado_conservacao"))
	if estado != "" && !estado.Valido() {
		return casosdeuso.FiltroAnuncios{}, common.ErrDadosInvalidos
	}
	categoria := consulta.Get("categoria")
	if categoria != "" && !anuncios.CategoriaValida(categoria) {
		return casosdeuso.FiltroAnuncios{}, common.ErrDadosInvalidos
	}
	return casosdeuso.FiltroAnuncios{
		Palavra: consulta.Get("q"), Categoria: categoria,
		Tamanho:           consulta.Get("tamanho"),
		EstadoConservacao: estado,
		PrecoMinCentavos:  precoMinimo,
		PrecoMaxCentavos:  precoMaximo,
		Limite:            limite,
		Deslocamento:      deslocamento,
	}, nil
}

func inteiroNaoNegativo(valor string) (int, error) {
	if valor == "" {
		return 0, nil
	}
	numero, err := strconv.Atoi(valor)
	if err != nil || numero < 0 {
		return 0, common.ErrDadosInvalidos
	}
	return numero, nil
}

func inteiro64NaoNegativo(valor string) (int64, error) {
	if valor == "" {
		return 0, nil
	}
	numero, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || numero < 0 {
		return 0, common.ErrDadosInvalidos
	}
	return numero, nil
}
