package web

import (
	nethttp "net/http"
	"net/url"
	"strings"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
)

func (a *AdaptadorPaginas) processarLogin(w nethttp.ResponseWriter, r *nethttp.Request) {
	_ = r.ParseForm()
	contexto := a.prepararContextoDocumento(r, "Entrar", conteudoLogin)
	contexto.URLRetorno = normalizarRotaRetorno(r.FormValue("retorno"))
	contexto.ValoresFormulario["identificador"] = r.FormValue("identificador")
	if !a.autenticacaoPermitida(r) {
		contexto.MensagemErro = "Tente autenticar novamente mais tarde."
		a.responderDocumentoHTML(w, nethttp.StatusTooManyRequests, contexto)
		return
	}
	sessao, err := a.controladorCadastro.Autenticar(
		r.Context(), r.FormValue("identificador"), r.FormValue("senha"),
	)
	if err != nil {
		a.registrarFalhaAutenticacao(r)
		contexto.MensagemErro, contexto.ErrosValidacao = apresentarErroCasoUso(err)
		a.responderDocumentoHTML(w, nethttp.StatusUnauthorized, contexto)
		return
	}
	a.limparFalhasAutenticacao(r)
	definirCookieSessao(w, r, sessao)
	a.responderRedirecionamento(w, r, contexto.URLRetorno)
}

func (a *AdaptadorPaginas) processarCadastroUsuario(w nethttp.ResponseWriter, r *nethttp.Request) {
	_ = r.ParseForm()
	contexto := a.prepararContextoDocumento(r, "Criar conta", conteudoCadastroUsuario)
	contexto.URLRetorno = normalizarRotaRetorno(r.FormValue("retorno"))
	contexto.ValoresFormulario = capturarValoresFormulario(r)
	if r.FormValue("senha") != r.FormValue("confirmar_senha") {
		contexto.MensagemErro = "Revise os campos destacados."
		contexto.ErrosValidacao["confirmar_senha"] = "As senhas informadas não coincidem."
		a.responderDocumentoHTML(w, nethttp.StatusUnprocessableEntity, contexto)
		return
	}
	_, err := a.controladorCadastro.CadastrarUsuario(r.Context(), casosdeuso.EntradaCadastro{
		Nome: r.FormValue("nome"), CPF: r.FormValue("cpf"), Email: r.FormValue("email"),
		Senha: r.FormValue("senha"), Telefone: r.FormValue("telefone"),
		Endereco: enderecoDoFormulario(r),
	})
	if err != nil {
		contexto.MensagemErro, contexto.ErrosValidacao = apresentarErroCasoUso(err)
		a.responderDocumentoHTML(w, nethttp.StatusUnprocessableEntity, contexto)
		return
	}
	destino := "/entrar?email=" + url.QueryEscape(r.FormValue("email"))
	if contexto.URLRetorno != "/catalogo" {
		destino += "&retorno=" + url.QueryEscape(contexto.URLRetorno)
	}
	a.responderRedirecionamento(w, r, destino)
}

func (a *AdaptadorPaginas) processarEncerramentoSessao(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	_ = a.controladorCadastro.EncerrarSessao(r.Context(), sessao.Token)
	removerCookieSessao(w, r)
	a.responderRedirecionamento(w, r, "/")
}

func (a *AdaptadorPaginas) processarAtualizacaoPerfil(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	_ = r.ParseForm()
	contexto := a.prepararContextoDocumento(r, "Meu perfil", conteudoPerfilUsuario)
	contexto.EditandoPerfil = true
	usuario, err := a.controladorCadastro.AtualizarPerfil(r.Context(), sessao.IDUsuario, casosdeuso.EntradaAtualizacaoPerfil{
		Nome: r.FormValue("nome"), Email: r.FormValue("email"),
		Telefone: r.FormValue("telefone"), Endereco: enderecoDoFormulario(r),
	})
	if err != nil {
		contexto.MensagemErro, contexto.ErrosValidacao = apresentarErroCasoUso(err)
		contexto.ValoresFormulario = capturarValoresFormulario(r)
		a.responderDocumentoHTML(w, nethttp.StatusUnprocessableEntity, contexto)
		return
	}
	contexto.UsuarioAutenticado = &usuario
	contexto.EditandoPerfil = false
	contexto.MensagemTemporaria = "Perfil atualizado."
	a.responderDocumentoHTML(w, nethttp.StatusOK, contexto)
}

func (a *AdaptadorPaginas) processarInclusaoCarrinho(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	_ = r.ParseForm()
	_, err := a.controladorCarrinho.AdicionarAoCarrinho(r.Context(), sessao.IDUsuario, r.FormValue("id_anuncio"))
	if err != nil {
		destino := normalizarRotaRetorno(r.FormValue("retorno"))
		a.responderRedirecionamento(w, r, adicionarMensagemNaURL(destino, "Não foi possível adicionar a peça."))
		return
	}
	destino := normalizarRotaRetorno(r.FormValue("retorno"))
	a.responderRedirecionamento(w, r, adicionarMensagemNaURL(destino, "Peça adicionada à sacola."))
}

func adicionarMensagemNaURL(destino, mensagem string) string {
	separador := "?"
	if strings.Contains(destino, "?") {
		separador = "&"
	}
	return destino + separador + "mensagem=" + url.QueryEscape(mensagem)
}

func (a *AdaptadorPaginas) processarRemocaoCarrinho(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	_, _ = a.controladorCarrinho.RemoverDoCarrinho(r.Context(), sessao.IDUsuario, r.PathValue("idAnuncio"))
	a.responderRedirecionamento(w, r, "/carrinho")
}

func (a *AdaptadorPaginas) processarExclusaoAnuncio(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	err := a.controladorAnuncio.ExcluirAnuncio(r.Context(), sessao.IDUsuario, r.PathValue("idAnuncio"))
	if err != nil {
		a.responderRedirecionamento(w, r, "/meus-anuncios?mensagem="+url.QueryEscape("Não foi possível excluir o anúncio."))
		return
	}
	a.responderRedirecionamento(w, r, "/meus-anuncios?mensagem="+url.QueryEscape("Anúncio excluído."))
}

func (a *AdaptadorPaginas) processarCriacaoAnuncio(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	a.processarPersistenciaAnuncio(w, r, sessao.IDUsuario, "")
}

func (a *AdaptadorPaginas) processarAtualizacaoAnuncio(w nethttp.ResponseWriter, r *nethttp.Request, sessao sessaoNavegador) {
	a.processarPersistenciaAnuncio(w, r, sessao.IDUsuario, r.PathValue("idAnuncio"))
}

func (a *AdaptadorPaginas) processarPersistenciaAnuncio(w nethttp.ResponseWriter, r *nethttp.Request, idUsuario, idAnuncio string) {
	_ = r.ParseForm()
	contexto := a.prepararContextoDocumento(r, "Vender", conteudoFormularioAnuncio)
	contexto.ValoresFormulario = capturarValoresFormulario(r)
	contexto.ValoresFormulario["acao"] = "/anuncios"
	contexto.ValoresFormulario["titulo_pagina"] = "Conte a história da sua peça."
	contexto.ValoresFormulario["eyebrow"] = "Novo anúncio"
	contexto.ValoresFormulario["botao"] = "Publicar anúncio"
	if idAnuncio != "" {
		contexto.EditandoAnuncio = true
		contexto.ValoresFormulario["acao"] = "/meus-anuncios/" + url.PathEscape(idAnuncio)
		contexto.ValoresFormulario["titulo_pagina"] = "Atualize os detalhes da peça."
		contexto.ValoresFormulario["eyebrow"] = "Editar anúncio"
		contexto.ValoresFormulario["botao"] = "Salvar alterações"
	}
	entrada := casosdeuso.EntradaAnuncio{
		Titulo: r.FormValue("titulo"), Descricao: r.FormValue("descricao"),
		Categoria: r.FormValue("categoria"), Tamanho: r.FormValue("tamanho"),
		Cor:               r.FormValue("cor"),
		EstadoConservacao: anuncios.EstadoConservacao(r.FormValue("estado_conservacao")),
		PrecoCentavos:     converterPrecoFormulario(r.FormValue("preco")),
		URLsFotos:         r.Form["urls_fotos"],
	}
	var (
		anuncioPersistido anuncios.Anuncio
		err               error
	)
	if idAnuncio == "" {
		anuncioPersistido, err = a.controladorAnuncio.CriarAnuncio(r.Context(), idUsuario, entrada)
	} else {
		anuncioPersistido, err = a.controladorAnuncio.AtualizarAnuncio(
			r.Context(),
			idUsuario,
			idAnuncio,
			entrada,
		)
	}
	if err != nil {
		contexto.MensagemErro, contexto.ErrosValidacao = apresentarErroCasoUso(err)
		a.responderDocumentoHTML(w, nethttp.StatusUnprocessableEntity, contexto)
		return
	}
	a.responderRedirecionamento(w, r, "/anuncios/"+url.PathEscape(anuncioPersistido.ID))
}

func enderecoDoFormulario(r *nethttp.Request) cadastros.Endereco {
	return cadastros.Endereco{
		CEP: r.FormValue("cep"), Logradouro: r.FormValue("logradouro"),
		Numero: r.FormValue("numero"), Complemento: r.FormValue("complemento"),
		Bairro: r.FormValue("bairro"), Cidade: r.FormValue("cidade"),
		Estado: r.FormValue("estado"),
	}
}

func capturarValoresFormulario(r *nethttp.Request) map[string]string {
	resultado := make(map[string]string, len(r.Form))
	for chave := range r.Form {
		resultado[chave] = r.FormValue(chave)
	}
	return resultado
}
