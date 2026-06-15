package web

import (
	nethttp "net/http"
	"net/url"
	"strconv"
	"strings"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
)

func (a *AdaptadorPaginas) exibirPaginaInicial(w nethttp.ResponseWriter, r *nethttp.Request) {
	contexto := a.prepararContextoDocumento(r, "Início", conteudoPaginaInicial)
	filtro := casosdeuso.FiltroAnuncios{Limite: 8}
	if contexto.UsuarioAutenticado != nil {
		filtro.ExcluirVendedor = contexto.UsuarioAutenticado.ID
	}
	anunciosEncontrados, err := a.controladorAnuncio.ListarAnuncios(r.Context(), filtro)
	if err != nil {
		contexto.MensagemErro = "Não foi possível carregar as peças."
	} else {
		contexto.AnunciosListados = anunciosEncontrados
	}
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirCatalogo(w nethttp.ResponseWriter, r *nethttp.Request) {
	contexto := a.prepararContextoDocumento(r, "Catálogo", conteudoCatalogo)
	contexto.FiltrosCatalogo = interpretarFiltrosCatalogo(r)
	anunciosEncontrados, possuiProximoLote, err := a.consultarLoteCatalogo(r, 0)
	if err != nil {
		contexto.MensagemErro = "Não foi possível carregar o catálogo."
	} else {
		contexto.AnunciosListados = anunciosEncontrados
		contexto.PossuiProximoLote = possuiProximoLote
		contexto.QuantidadeCarregada = len(anunciosEncontrados)
		contexto.URLProximoLote = urlProximoLoteCatalogo(r.URL.Query(), len(anunciosEncontrados))
	}
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirProximoLoteCatalogo(w nethttp.ResponseWriter, r *nethttp.Request) {
	deslocamento, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	contexto := a.prepararContextoDocumento(r, "", "")
	queryCatalogo := r.URL.Query()
	queryCatalogo.Del("offset")
	contexto.URLRetorno = "/catalogo"
	if codificada := queryCatalogo.Encode(); codificada != "" {
		contexto.URLRetorno += "?" + codificada
	}
	anunciosEncontrados, possuiProximoLote, err := a.consultarLoteCatalogo(r, deslocamento)
	if err != nil {
		contexto.MensagemErro = "Não foi possível carregar mais peças."
	} else {
		contexto.AnunciosListados = anunciosEncontrados
		contexto.PossuiProximoLote = possuiProximoLote
		contexto.QuantidadeCarregada = deslocamento + len(anunciosEncontrados)
		contexto.URLProximoLote = urlProximoLoteCatalogo(
			r.URL.Query(),
			deslocamento+len(anunciosEncontrados),
		)
	}
	a.responderFragmentoHTML(w, fragmentoProximoLote, contexto)
}

func (a *AdaptadorPaginas) consultarLoteCatalogo(r *nethttp.Request, deslocamento int) ([]anuncios.Anuncio, bool, error) {
	filtrosSolicitados := interpretarFiltrosCatalogo(r)
	filtro := casosdeuso.FiltroAnuncios{
		Palavra: filtrosSolicitados.Busca, Categoria: filtrosSolicitados.Categoria,
		Tamanho:           filtrosSolicitados.Tamanho,
		EstadoConservacao: anuncios.EstadoConservacao(filtrosSolicitados.Conservacao),
		PrecoMinCentavos:  converterPrecoFormulario(filtrosSolicitados.PrecoMinimo),
		PrecoMaxCentavos:  converterPrecoFormulario(filtrosSolicitados.PrecoMaximo),
		Limite:            25,
		Deslocamento:      deslocamento,
	}
	if token := tokenSessaoDoCookie(r); token != "" {
		if id, err := a.controladorCadastro.IdentificarUsuario(r.Context(), token); err == nil {
			filtro.ExcluirVendedor = id
		}
	}
	anunciosEncontrados, err := a.controladorAnuncio.ListarAnuncios(r.Context(), filtro)
	if err != nil {
		return nil, false, err
	}
	possuiProximoLote := len(anunciosEncontrados) > 24
	if possuiProximoLote {
		anunciosEncontrados = anunciosEncontrados[:24]
	}
	return anunciosEncontrados, possuiProximoLote, nil
}

func (a *AdaptadorPaginas) exibirDetalheAnuncio(w nethttp.ResponseWriter, r *nethttp.Request) {
	contexto := a.prepararContextoDocumento(r, "Detalhes do anúncio", conteudoDetalheAnuncio)
	anuncio, err := a.controladorAnuncio.ObterAnuncio(r.Context(), r.PathValue("idAnuncio"))
	if err != nil {
		contexto.Conteudo = conteudoNaoEncontrado
		contexto.Titulo = "Anúncio não encontrado"
		a.responderDocumentoHTML(w, nethttp.StatusNotFound, contexto)
		return
	}
	contexto.DetalheAnuncio = &anuncio
	contexto.Titulo = anuncio.Titulo
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirPerfilPublicoVendedor(w nethttp.ResponseWriter, r *nethttp.Request) {
	contexto := a.prepararContextoDocumento(r, "Perfil do vendedor", conteudoPerfilVendedor)
	perfil, err := a.controladorAnuncio.ObterPerfilPublicoVendedor(r.Context(), r.PathValue("idVendedor"))
	if err != nil {
		contexto.Conteudo = conteudoNaoEncontrado
		contexto.Titulo = "Vendedor não encontrado"
		a.responderDocumentoHTML(w, nethttp.StatusNotFound, contexto)
		return
	}
	contexto.PerfilVendedor = &perfil
	contexto.Titulo = perfil.Vendedor.Nome
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirLogin(w nethttp.ResponseWriter, r *nethttp.Request) {
	contexto := a.prepararContextoDocumento(r, "Entrar", conteudoLogin)
	if contexto.UsuarioAutenticado != nil {
		a.responderRedirecionamento(w, r, "/perfil")
		return
	}
	contexto.URLRetorno = normalizarRotaRetorno(r.URL.Query().Get("retorno"))
	contexto.ValoresFormulario["identificador"] = r.URL.Query().Get("email")
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirCadastroUsuario(w nethttp.ResponseWriter, r *nethttp.Request) {
	contexto := a.prepararContextoDocumento(r, "Criar conta", conteudoCadastroUsuario)
	if contexto.UsuarioAutenticado != nil {
		a.responderRedirecionamento(w, r, "/perfil")
		return
	}
	contexto.URLRetorno = normalizarRotaRetorno(r.URL.Query().Get("retorno"))
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirPerfilUsuario(w nethttp.ResponseWriter, r *nethttp.Request, _ sessaoNavegador) {
	contexto := a.prepararContextoDocumento(r, "Meu perfil", conteudoPerfilUsuario)
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirEdicaoPerfilUsuario(w nethttp.ResponseWriter, r *nethttp.Request, _ sessaoNavegador) {
	contexto := a.prepararContextoDocumento(r, "Meu perfil", conteudoPerfilUsuario)
	contexto.EditandoPerfil = true
	contexto.ValoresFormulario = valoresFormularioPerfil(*contexto.UsuarioAutenticado)
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirAnunciosUsuario(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	contexto := a.prepararContextoDocumento(r, "Meus anúncios", conteudoAnunciosUsuario)
	anunciosDoUsuario, err := a.controladorAnuncio.ListarAnunciosDoVendedor(r.Context(), sessao.IDUsuario)
	if err != nil {
		contexto.MensagemErro = "Não foi possível carregar seus anúncios."
	} else {
		contexto.AnunciosListados = anunciosDoUsuario
	}
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirPublicacaoAnuncio(w nethttp.ResponseWriter, r *nethttp.Request, _ sessaoNavegador) {
	contexto := a.prepararContextoDocumento(r, "Vender", conteudoFormularioAnuncio)
	contexto.ValoresFormulario["acao"] = "/anuncios"
	contexto.ValoresFormulario["titulo_pagina"] = "Conte a história da sua peça."
	contexto.ValoresFormulario["eyebrow"] = "Novo anúncio"
	contexto.ValoresFormulario["botao"] = "Publicar anúncio"
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirEdicaoAnuncio(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	contexto := a.prepararContextoDocumento(r, "Editar anúncio", conteudoFormularioAnuncio)
	anuncio, err := a.controladorAnuncio.ObterAnuncio(r.Context(), r.PathValue("idAnuncio"))
	if err != nil || anuncio.IDVendedor != sessao.IDUsuario || anuncio.Status != anuncios.StatusAnuncioDisponivel {
		contexto.Conteudo = conteudoNaoEncontrado
		a.responderDocumentoHTML(w, nethttp.StatusNotFound, contexto)
		return
	}
	contexto.DetalheAnuncio = &anuncio
	contexto.EditandoAnuncio = true
	contexto.ValoresFormulario = valoresFormularioAnuncio(anuncio.Anuncio)
	contexto.ValoresFormulario["acao"] = "/meus-anuncios/" + url.PathEscape(anuncio.ID)
	contexto.ValoresFormulario["titulo_pagina"] = "Atualize os detalhes da peça."
	contexto.ValoresFormulario["eyebrow"] = "Editar anúncio"
	contexto.ValoresFormulario["botao"] = "Salvar alterações"
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) exibirCarrinhoUsuario(w nethttp.ResponseWriter, r *nethttp.Request, _ sessaoNavegador) {
	contexto := a.prepararContextoDocumento(r, "Minha sacola", conteudoCarrinhoUsuario)
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func interpretarFiltrosCatalogo(r *nethttp.Request) filtrosCatalogo {
	return filtrosCatalogo{
		Busca: r.URL.Query().Get("q"), Categoria: r.URL.Query().Get("categoria"),
		Conservacao: r.URL.Query().Get("estado_conservacao"),
		Tamanho:     r.URL.Query().Get("tamanho"),
		PrecoMinimo: r.URL.Query().Get("preco_min"),
		PrecoMaximo: r.URL.Query().Get("preco_max"),
	}
}

func converterPrecoFormulario(valorFormulario string) int64 {
	texto := strings.ReplaceAll(strings.TrimSpace(valorFormulario), ".", "")
	texto = strings.ReplaceAll(texto, ",", ".")
	numero, err := strconv.ParseFloat(texto, 64)
	if err != nil || numero <= 0 {
		return 0
	}
	return int64(numero*100 + .5)
}

func urlProximoLoteCatalogo(query url.Values, deslocamento int) string {
	copia := url.Values{}
	for chave, valores := range query {
		for _, valorFormulario := range valores {
			copia.Add(chave, valorFormulario)
		}
	}
	copia.Set("offset", strconv.Itoa(deslocamento))
	return "/fragmentos/catalogo?" + copia.Encode()
}

func normalizarRotaRetorno(valorFormulario string) string {
	if strings.HasPrefix(valorFormulario, "/") && !strings.HasPrefix(valorFormulario, "//") {
		return valorFormulario
	}
	return "/catalogo"
}

func valoresFormularioAnuncio(item anuncios.Anuncio) map[string]string {
	return map[string]string{
		"titulo": item.Titulo, "descricao": item.Descricao, "categoria": item.Categoria,
		"tamanho": item.Tamanho, "cor": item.Cor,
		"estado_conservacao": string(item.EstadoConservacao),
		"preco":              strings.ReplaceAll(formatarDinheiro(item.PrecoCentavos), "R$ ", ""),
	}
}

func valoresFormularioPerfil(usuario cadastros.Usuario) map[string]string {
	return map[string]string{
		"nome": usuario.Nome, "email": usuario.Email, "telefone": usuario.Telefone,
		"cep": usuario.EnderecoPrincipal.CEP, "estado": usuario.EnderecoPrincipal.Estado,
		"logradouro": usuario.EnderecoPrincipal.Logradouro, "numero": usuario.EnderecoPrincipal.Numero,
		"complemento": usuario.EnderecoPrincipal.Complemento, "bairro": usuario.EnderecoPrincipal.Bairro,
		"cidade": usuario.EnderecoPrincipal.Cidade,
	}
}
