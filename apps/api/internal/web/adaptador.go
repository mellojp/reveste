package web

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	nethttp "net/http"
	"sync"

	"reveste/apps/api/internal/casosdeuso"
)

//go:embed templates/*.html
var arquivosTemplates embed.FS

type AdaptadorPaginas struct {
	controladorCadastro *casosdeuso.ControladorCadastro
	controladorAnuncio  *casosdeuso.ControladorAnuncio
	controladorCarrinho *casosdeuso.ControladorCarrinho
	documentosHTML      *template.Template
	logger              *slog.Logger
	tentativasMu        sync.Mutex
	tentativasLogin     map[string]registroTentativasLogin
}

func NovoAdaptadorPaginas(
	controladorCadastro *casosdeuso.ControladorCadastro,
	controladorAnuncio *casosdeuso.ControladorAnuncio,
	controladorCarrinho *casosdeuso.ControladorCarrinho,
	logger *slog.Logger,
) (nethttp.Handler, error) {
	tmpl, err := template.New("web").
		Funcs(funcoesApresentacaoTemplates()).
		ParseFS(arquivosTemplates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("carregar templates web: %w", err)
	}
	adaptador := &AdaptadorPaginas{
		controladorCadastro: controladorCadastro,
		controladorAnuncio:  controladorAnuncio,
		controladorCarrinho: controladorCarrinho,
		documentosHTML:      tmpl,
		logger:              logger,
		tentativasLogin:     make(map[string]registroTentativasLogin),
	}
	mux := nethttp.NewServeMux()
	adaptador.registrarConsultasPaginas(mux)
	adaptador.registrarComandosFormularios(mux)
	return mux, nil
}

func (a *AdaptadorPaginas) registrarConsultasPaginas(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /", a.exibirPaginaInicial)
	mux.HandleFunc("GET /catalogo", a.exibirCatalogo)
	mux.HandleFunc("GET /fragmentos/catalogo", a.exibirProximoLoteCatalogo)
	mux.HandleFunc("GET /anuncios/{idAnuncio}", a.exibirDetalheAnuncio)
	mux.HandleFunc("GET /vendedores/{idVendedor}", a.exibirPerfilPublicoVendedor)
	mux.HandleFunc("GET /entrar", a.exibirLogin)
	mux.HandleFunc("GET /cadastro", a.exibirCadastroUsuario)
	mux.HandleFunc("GET /perfil", a.exigirSessao(a.exibirPerfilUsuario))
	mux.HandleFunc("GET /perfil/editar", a.exigirSessao(a.exibirEdicaoPerfilUsuario))
	mux.HandleFunc("GET /meus-anuncios", a.exigirSessao(a.exibirAnunciosUsuario))
	mux.HandleFunc("GET /meus-anuncios/{idAnuncio}/editar", a.exigirSessao(a.exibirEdicaoAnuncio))
	mux.HandleFunc("GET /vender", a.exigirSessao(a.exibirPublicacaoAnuncio))
	mux.HandleFunc("GET /carrinho", a.exigirSessao(a.exibirCarrinhoUsuario))
}

func (a *AdaptadorPaginas) registrarComandosFormularios(mux *nethttp.ServeMux) {
	mux.HandleFunc("POST /entrar", a.processarLogin)
	mux.HandleFunc("POST /cadastro", a.processarCadastroUsuario)
	mux.HandleFunc("POST /sair", a.exigirSessao(a.processarEncerramentoSessao))
	mux.HandleFunc("POST /perfil", a.exigirSessao(a.processarAtualizacaoPerfil))
	mux.HandleFunc("POST /carrinho/itens", a.exigirSessao(a.processarInclusaoCarrinho))
	mux.HandleFunc("POST /carrinho/itens/{idAnuncio}/remover", a.exigirSessao(a.processarRemocaoCarrinho))
	mux.HandleFunc("POST /meus-anuncios/{idAnuncio}/excluir", a.exigirSessao(a.processarExclusaoAnuncio))
	mux.HandleFunc("POST /anuncios", a.exigirSessao(a.processarCriacaoAnuncio))
	mux.HandleFunc("POST /meus-anuncios/{idAnuncio}", a.exigirSessao(a.processarAtualizacaoAnuncio))
}
