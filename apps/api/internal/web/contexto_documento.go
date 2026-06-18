package web

import (
	nethttp "net/http"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
	"reveste/apps/api/internal/dominio/interacao"
	"reveste/apps/api/internal/transporte"
)

const (
	conteudoPaginaInicial      = "inicio"
	conteudoCatalogo           = "catalogo"
	conteudoDetalheAnuncio     = "detalhe-anuncio"
	conteudoPerfilVendedor     = "vendedor"
	conteudoLogin              = "entrar"
	conteudoCadastroUsuario    = "cadastro"
	conteudoPerfilUsuario      = "perfil"
	conteudoEnderecos          = "enderecos"
	conteudoFormularioEndereco = "formulario-endereco"
	conteudoAnunciosUsuario    = "meus-anuncios"
	conteudoFormularioAnuncio  = "formulario-anuncio"
	conteudoCarrinhoUsuario    = "carrinho"
	conteudoCheckout           = "checkout"
	conteudoPedidosUsuario     = "meus-pedidos"
	conteudoDetalhePedido      = "detalhe-pedido"
	conteudoVendasUsuario      = "minhas-vendas"
	conteudoDetalheVenda       = "detalhe-venda"
	conteudoNaoEncontrado      = "nao-encontrado"
	fragmentoProximoLote       = "catalogo-lote"
)

// contextoDocumento concentra somente os contexto de apresentacao entregues aos templates.
// Os manipuladores consultam os casos de uso, preenchem este contexto e o renderer
// transforma o resultado em um documento HTML ou fragmento HTMX.
type contextoDocumento struct {
	Conteudo            string
	Titulo              string
	RotaAtual           string
	URLRetorno          string
	UsuarioAutenticado  *cadastros.Usuario
	CarrinhoAutenticado casosdeuso.CarrinhoDetalhado
	AnunciosListados    []anuncios.Anuncio
	EnderecosUsuario    []cadastros.Endereco
	EnderecoEmEdicao    *cadastros.Endereco
	PedidosListados     []compras.Pedido
	PedidoDetalhe       *compras.Pedido
	AvaliacaoPedido     *interacao.Avaliacao
	ResumoCompra        *compras.Compra
	CompraConfirmada    bool
	AvaliacaoVendedor   casosdeuso.MediaAvaliacoes
	DetalheAnuncio      *casosdeuso.AnuncioDetalhado
	PerfilVendedor      *casosdeuso.PerfilVendedorDetalhado
	FiltrosCatalogo     filtrosCatalogo
	URLProximoLote      string
	PossuiProximoLote   bool
	QuantidadeCarregada int
	EditandoPerfil      bool
	EditandoAnuncio     bool
	MensagemErro        string
	ErrosValidacao      map[string]string
	ValoresFormulario   map[string]string
	MensagemTemporaria  string
	OpcoesCategoria     []opcaoFormulario
	OpcoesConservacao   []opcaoFormulario
}

type opcaoFormulario struct {
	Valor  string
	Rotulo string
}

type contextoCartaoAnuncio struct {
	Anuncio              anuncios.Anuncio
	IDUsuarioAutenticado string
	URLRetorno           string
	EstaNoCarrinho       bool
}

type contextoMensagemCampo struct {
	ErrosValidacao map[string]string
	NomeCampo      string
	IDMensagem     string
}

type filtrosCatalogo struct {
	Busca       string
	Categoria   string
	Conservacao string
	Tamanho     string
	PrecoMinimo string
	PrecoMaximo string
}

func (a *AdaptadorPaginas) prepararContextoDocumento(
	r *nethttp.Request,
	titulo,
	conteudo string,
) contextoDocumento {
	contexto := contextoDocumento{
		Conteudo: conteudo, Titulo: titulo, RotaAtual: r.URL.Path, URLRetorno: r.URL.RequestURI(),
		ErrosValidacao: map[string]string{}, ValoresFormulario: map[string]string{},
		MensagemTemporaria: r.URL.Query().Get("mensagem"),
		OpcoesCategoria: []opcaoFormulario{
			{"", "Tudo"}, {"vestidos", "Vestidos"}, {"camisetas", "Camisetas"},
			{"calcas", "Calças"}, {"saias_e_shorts", "Saias e shorts"},
			{"casacos", "Casacos"}, {"acessorios", "Acessórios"},
			{"calcados", "Calçados"}, {"outros", "Outros"},
		},
		OpcoesConservacao: []opcaoFormulario{
			{"", "Todos os estados"}, {"novo", "Novo"}, {"seminovo", "Seminovo"},
			{"usado", "Usado"}, {"muito_usado", "Muito usado"}, {"desgastado", "Desgastado"},
		},
	}
	token := transporte.TokenSessaoDoCookie(r)
	idUsuario, err := a.controladorCadastro.IdentificarUsuario(r.Context(), token)
	if err != nil {
		return contexto
	}
	usuario, err := a.controladorCadastro.ObterPerfil(r.Context(), idUsuario)
	if err == nil {
		contexto.UsuarioAutenticado = &usuario
	}
	carrinho, err := a.controladorCarrinho.ObterCarrinho(r.Context(), idUsuario)
	if err == nil {
		contexto.CarrinhoAutenticado = carrinho
	}
	return contexto
}
