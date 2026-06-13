package casosdeuso

import (
	"context"
	"time"

	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
)

type OperacoesUsuarios interface {
	CriarUsuario(context.Context, cadastros.Usuario) error
	AtualizarUsuario(context.Context, cadastros.Usuario) error
	BuscarUsuarioPorID(context.Context, string) (cadastros.Usuario, error)
	BuscarUsuarioPorEmailOuCPF(context.Context, string) (cadastros.Usuario, error)
}

type OperacoesSessoes interface {
	CriarSessao(context.Context, string, string, time.Time) error
	BuscarUsuarioDaSessao(context.Context, string, time.Time) (string, error)
	RemoverSessao(context.Context, string) error
}

type FiltroAnuncios struct {
	Palavra            string
	Categoria          string
	Tamanho            string
	EstadoConservacao  anuncios.EstadoConservacao
	PrecoMinCentavos   int64
	PrecoMaxCentavos   int64
	IDsAnuncios        []string
	IDVendedor         string
	ExcluirVendedor    string
	IncluirTodosStatus bool
	Limite             int
	Deslocamento       int
}

type OperacoesAnuncios interface {
	CriarAnuncio(context.Context, anuncios.Anuncio) error
	AtualizarAnuncio(context.Context, anuncios.Anuncio) error
	ExcluirAnuncio(context.Context, string, string, time.Time) error
	BuscarAnuncioPorID(context.Context, string) (anuncios.Anuncio, error)
	ListarAnuncios(context.Context, FiltroAnuncios) ([]anuncios.Anuncio, error)
}

type OperacoesCarrinhos interface {
	ObterOuCriarCarrinho(context.Context, string, string, time.Time) (compras.Carrinho, error)
	AdicionarAnuncioAoCarrinho(context.Context, string, string, string, time.Time) (compras.Carrinho, error)
	RemoverAnuncioDoCarrinho(context.Context, string, string, string, time.Time) (compras.Carrinho, error)
}

type SolicitacaoUpload struct {
	Pathname           string
	TiposPermitidos    []string
	TamanhoMaximoBytes int64
	ExpiraEm           time.Time
}

type AutorizacaoUpload struct {
	URLUpload          string   `json:"url_upload"`
	Pathname           string   `json:"pathname"`
	Token              string   `json:"token"`
	TiposAceitos       []string `json:"tipos_aceitos"`
	TamanhoMaximoBytes int64    `json:"tamanho_maximo_bytes"`
}

type ArmazenamentoArquivos interface {
	AutorizarUpload(context.Context, SolicitacaoUpload) (AutorizacaoUpload, error)
}

type GeradorID interface {
	Novo() string
}

type GerenciadorSenhas interface {
	Gerar(string) (string, error)
	Comparar(string, string) bool
}

type Relogio interface {
	Agora() time.Time
}
