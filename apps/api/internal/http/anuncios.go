package http

import (
	nethttp "net/http"

	"reveste/apps/api/internal/casosdeuso"
	casosdeusoanuncios "reveste/apps/api/internal/casosdeuso/anuncios"
	"reveste/apps/api/internal/dominio/anuncios"
)

func (a *API) registrarRotasAnuncios(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /v1/anuncios", a.listarAnuncios)
	mux.HandleFunc("POST /v1/anuncios", a.autenticado(a.criarAnuncio))
}

func (a *API) criarAnuncio(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, _ string) {
	var entrada struct {
		Titulo            string                     `json:"titulo"`
		Descricao         string                     `json:"descricao"`
		Categoria         string                     `json:"categoria"`
		Tamanho           string                     `json:"tamanho"`
		Cor               string                     `json:"cor"`
		EstadoConservacao anuncios.EstadoConservacao `json:"estado_conservacao"`
		PrecoCentavos     int64                      `json:"preco_centavos"`
		URLsFotos         []string                   `json:"urls_fotos"`
	}
	if !decodificarJSON(w, r, &entrada) {
		return
	}
	anuncio, err := a.anuncios.CriarAnuncio(r.Context(), idUsuario, casosdeusoanuncios.EntradaAnuncio{
		Titulo: entrada.Titulo, Descricao: entrada.Descricao, Categoria: entrada.Categoria,
		Tamanho: entrada.Tamanho, Cor: entrada.Cor, EstadoConservacao: entrada.EstadoConservacao,
		PrecoCentavos: entrada.PrecoCentavos, URLsFotos: entrada.URLsFotos,
	})
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusCreated, anuncio)
}

func (a *API) listarAnuncios(w nethttp.ResponseWriter, r *nethttp.Request) {
	consulta := r.URL.Query()
	lista, err := a.anuncios.ListarAnuncios(r.Context(), casosdeuso.FiltroAnuncios{
		Palavra: consulta.Get("q"), Categoria: consulta.Get("categoria"),
		Tamanho:           consulta.Get("tamanho"),
		EstadoConservacao: anuncios.EstadoConservacao(consulta.Get("estado_conservacao")),
		PrecoMinCentavos:  inteiro64(consulta.Get("preco_min_centavos")),
		PrecoMaxCentavos:  inteiro64(consulta.Get("preco_max_centavos")),
		Limite:            inteiro(consulta.Get("limite")),
		Deslocamento:      inteiro(consulta.Get("deslocamento")),
	})
	if err != nil {
		a.escreverErro(w, err)
		return
	}
	escreverJSON(w, nethttp.StatusOK, map[string]any{"dados": lista, "quantidade": len(lista)})
}
