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
	IDVendedor         string
	ExcluirVendedor    string
	IncluirTodosStatus bool
	Limite             int
	Deslocamento       int
}

type OperacoesAnuncios interface {
	CriarAnuncio(context.Context, anuncios.Anuncio) error
	BuscarAnuncioPorID(context.Context, string) (anuncios.Anuncio, error)
	ListarAnuncios(context.Context, FiltroAnuncios) ([]anuncios.Anuncio, error)
}

type OperacoesCarrinhos interface {
	ObterOuCriarCarrinho(context.Context, string, string, time.Time) (compras.Carrinho, error)
	AdicionarAnuncioAoCarrinho(context.Context, string, string, string, time.Time) (compras.Carrinho, error)
	RemoverAnuncioDoCarrinho(context.Context, string, string, string, time.Time) (compras.Carrinho, error)
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
