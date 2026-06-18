package casosdeuso

import (
	"context"
	"time"

	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
	"reveste/apps/api/internal/dominio/interacao"
)

type OperacoesUsuarios interface {
	CriarUsuario(context.Context, cadastros.Usuario) error
	AtualizarUsuario(context.Context, cadastros.Usuario) error
	BuscarUsuarioPorID(context.Context, string) (cadastros.Usuario, error)
	BuscarUsuarioPorEmailOuCPF(context.Context, string) (cadastros.Usuario, error)
	// Operacoes sobre os enderecos do usuario (1:N). Todas filtram por id_usuario e
	// ignoram enderecos com exclusao logica.
	ListarEnderecos(context.Context, string) ([]cadastros.Endereco, error)
	BuscarEndereco(ctx context.Context, idUsuario, idEndereco string) (cadastros.Endereco, error)
	AdicionarEndereco(ctx context.Context, idUsuario string, endereco cadastros.Endereco, agora time.Time) error
	AtualizarEndereco(ctx context.Context, idUsuario, idEndereco string, endereco cadastros.Endereco, agora time.Time) error
	RemoverEndereco(ctx context.Context, idUsuario, idEndereco string, agora time.Time) error
	DefinirEnderecoPrincipal(ctx context.Context, idUsuario, idEndereco string, agora time.Time) error
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

// OperacoesCheckout persiste o checkout em fases. A primeira fase reserva os anuncios e
// cria a intencao de compra antes de qualquer chamada ao provedor financeiro. As fases
// seguintes confirmam ou desfazem a intencao de forma transacional e idempotente.
type OperacoesCheckout interface {
	BuscarCompraPorChave(context.Context, string) (compras.Compra, error)
	// IniciarCompra reserva os anuncios (disponivel -> reservado) e grava compra,
	// pedidos, itens, entregas e pagamento pendentes. O bool informa se esta chamada
	// criou a intencao; chamadas concorrentes com a mesma chave recebem a existente.
	IniciarCompra(context.Context, compras.Compra, compras.Pagamento, string) (compras.Compra, bool, error)
	// ConfirmarCompraAprovada conclui a intencao: anuncios viram vendidos, compra e
	// pagamento ficam aprovados, pedidos aguardam envio e o carrinho e limpo.
	ConfirmarCompraAprovada(ctx context.Context, chave, provedor, identificadorExterno string, agora time.Time) (compras.Compra, error)
	// RecusarCompra desfaz a reserva e registra a recusa do pagamento.
	RecusarCompra(ctx context.Context, chave, provedor, identificadorExterno string, agora time.Time) error
	// ExpirarComprasPendentes libera reservas cujo prazo terminou.
	ExpirarComprasPendentes(ctx context.Context, agora time.Time) (int, error)
	// ListarPedidosDoComprador devolve os pedidos do comprador, com itens, mais recentes primeiro.
	ListarPedidosDoComprador(context.Context, string) ([]compras.Pedido, error)
}

// MediaAvaliacoes resume a reputacao de um vendedor.
type MediaAvaliacoes struct {
	Media      float64 `json:"media"`
	Quantidade int     `json:"quantidade"`
}

// OperacoesPedidos persiste as transicoes do ciclo de vida do pedido apos a compra.
type OperacoesPedidos interface {
	ListarPedidosDoVendedor(context.Context, string) ([]compras.Pedido, error)
	BuscarPedidoDoComprador(context.Context, string, string) (compras.Pedido, error)
	BuscarPedidoDoVendedor(context.Context, string, string) (compras.Pedido, error)
	// BuscarAvaliacaoDoPedido devolve a avaliacao ja registrada para o pedido, ou
	// ErrNaoEncontrado quando ainda nao foi avaliado.
	BuscarAvaliacaoDoPedido(context.Context, string) (interacao.Avaliacao, error)
	// MarcarPedidoEnviado avanca itens para enviado, a entrega para postado e o pedido
	// para aguardando_entrega. Autoriza pelo vendedor; ErrNaoPermitido se nao for dele
	// ou nao estiver aguardando envio.
	MarcarPedidoEnviado(ctx context.Context, idPedido, idVendedor, provedor, rastreio string, agora time.Time) error
	// ConfirmarRecebimentoPedido marca itens como recebidos, a entrega como entregue e
	// finaliza o pedido. Autoriza pelo comprador.
	ConfirmarRecebimentoPedido(ctx context.Context, idPedido, idComprador string, agora time.Time) error
	RegistrarAvaliacao(context.Context, interacao.Avaliacao) error
	// ProcessarItensVencidos marca como nao_enviado os itens cujo prazo expirou e ainda
	// aguardam envio, incrementa o contador do vendedor e o bloqueia ao atingir o limite.
	// Devolve quantos itens foram afetados.
	ProcessarItensVencidos(ctx context.Context, agora time.Time, limiteBloqueio int) (int, error)
	MediaAvaliacoesVendedor(context.Context, string) (MediaAvaliacoes, error)
}

type SolicitacaoPagamento struct {
	IDCompra          string
	ValorCentavos     int64
	ChaveIdempotencia string
}

type ResultadoPagamento struct {
	Aprovado             bool
	Provedor             string
	IdentificadorExterno string
}

// ProcessadorPagamento abstrai o provedor financeiro. Implementacoes devem tratar
// ChaveIdempotencia como idempotency key no provedor: repeticoes da mesma intencao podem
// ocorrer para recuperar uma resposta apos falha transitoria e nao podem cobrar novamente.
// O MVP usa um adaptador simulado.
type ProcessadorPagamento interface {
	Processar(context.Context, SolicitacaoPagamento) (ResultadoPagamento, error)
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
