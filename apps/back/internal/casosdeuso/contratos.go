package casosdeuso

import (
	"context"
	"time"

	"reveste/apps/back/internal/dominio/anuncios"
	"reveste/apps/back/internal/dominio/cadastros"
	"reveste/apps/back/internal/dominio/compras"
	"reveste/apps/back/internal/dominio/interacao"
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

// OperacoesReativacao cobre o ciclo de reativacao de um vendedor bloqueado: consulta do
// estado atual e a transicao atomica de desbloqueio com zeragem do contador.
type OperacoesReativacao interface {
	BuscarUsuarioPorID(context.Context, string) (cadastros.Usuario, error)
	// ReativarVendedor desbloqueia o vendedor e zera itens_nao_enviados. O bool informa se
	// havia de fato um bloqueio para reverter (false quando ja estava ativo).
	ReativarVendedor(ctx context.Context, idVendedor string, agora time.Time) (bool, error)
}

// RegistroNotificacoes e a porta minima usada por outros casos de uso para registrar uma
// notificacao como efeito de um evento de dominio (envio, recebimento, avaliacao, mensagem).
type RegistroNotificacoes interface {
	CriarNotificacao(context.Context, interacao.Notificacao) error
}

// OperacoesConversas cobre o chat por pedido: identificacao dos participantes (para
// autorizacao), criacao idempotente da conversa do pedido e leitura/escrita de mensagens.
type OperacoesConversas interface {
	// BuscarParticipantesPedido devolve comprador e vendedor do pedido, ou ErrNaoEncontrado.
	BuscarParticipantesPedido(ctx context.Context, idPedido string) (idComprador, idVendedor string, err error)
	// ObterOuCriarConversa devolve o id da conversa do pedido, criando-a quando ainda nao existe.
	ObterOuCriarConversa(ctx context.Context, novoID, idPedido string, agora time.Time) (string, error)
	ListarMensagens(ctx context.Context, idConversa string) ([]interacao.Mensagem, error)
	CriarMensagem(ctx context.Context, mensagem interacao.Mensagem) error
}

// OperacoesNotificacoes acrescenta a leitura e o controle de leitura da caixa de entrada.
type OperacoesNotificacoes interface {
	RegistroNotificacoes
	ListarNotificacoes(ctx context.Context, idUsuario string, limite int) ([]interacao.Notificacao, error)
	ContarNotificacoesNaoLidas(ctx context.Context, idUsuario string) (int, error)
	MarcarNotificacoesLidas(ctx context.Context, idUsuario string, agora time.Time) error
	RemoverNotificacao(ctx context.Context, idUsuario, idNotificacao string) error
	LimparNotificacoes(ctx context.Context, idUsuario string) error
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
	// BuscarCompraPendenteDoComprador devolve a compra do comprador que ainda aguarda pagamento
	// (a mais recente), ou ErrNaoEncontrado. Sustenta a tela dedicada de pagamento.
	BuscarCompraPendenteDoComprador(context.Context, string) (compras.Compra, error)
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
	// EmailPagador identifica o pagador junto ao provedor (exigido por alguns meios, como
	// PIX no Mercado Pago). Pode ficar vazio com provedores que nao o requerem.
	EmailPagador string
	// NomePagador e o nome do comprador, enviado ao provedor junto a cobranca (recomendado
	// pelo Mercado Pago). No sandbox do Mercado Pago, o primeiro nome controla o desfecho do
	// pagamento de teste: "APRO" aprova, "CONT" mantem pendente e "OTHE" recusa.
	NomePagador string
	// Cartao, quando presente, faz a cobranca por cartao (token gerado no frontend) em vez de PIX.
	// Nulo => PIX. O numero do cartao nunca chega ao backend; so o token.
	Cartao *DadosCartao
}

// DadosCartao traz o token do cartao (gerado no frontend pelo SDK do provedor, padrao PCI) e os
// dados nao sensiveis necessarios para criar a cobranca por cartao.
type DadosCartao struct {
	Token           string
	MetodoPagamento string // bandeira/payment_method_id (ex.: "master", "visa")
	Parcelas        int
	TipoDocumento   string // ex.: "CPF" (opcional)
	NumeroDocumento string // opcional
}

// StatusCobranca e o desfecho de uma cobranca no provedor financeiro.
type StatusCobranca string

const (
	// CobrancaPendente: o provedor aceitou a intencao, mas o comprador ainda nao concluiu o
	// pagamento (ex.: PIX aguardando, cartao em analise). A confirmacao chega depois por
	// webhook, em ControladorCheckout.ConfirmarPagamentoExterno.
	CobrancaPendente StatusCobranca = "pendente"
	// CobrancaAprovada: pagamento confirmado de forma sincrona (ex.: provedor simulado).
	CobrancaAprovada StatusCobranca = "aprovada"
	// CobrancaRecusada: o provedor recusou a cobranca.
	CobrancaRecusada StatusCobranca = "recusada"
)

// InstrucoesPagamento traz o que o comprador precisa para concluir um pagamento pendente.
// Fica vazio quando a cobranca ja nasce aprovada (provedores sincronos).
type InstrucoesPagamento struct {
	Tipo         string `json:"tipo,omitempty"`           // "pix", "redirect" ou vazio
	PixCopiaCola string `json:"pix_copia_cola,omitempty"` // payload PIX copia-e-cola
	PixQRCode    string `json:"pix_qr_code,omitempty"`    // QR Code do PIX (imagem base64 ou payload)
	URLPagamento string `json:"url_pagamento,omitempty"`  // pagina de pagamento do provedor
}

// Cobranca e a resposta do provedor a uma solicitacao de pagamento.
type Cobranca struct {
	Status               StatusCobranca
	Provedor             string
	IdentificadorExterno string
	Instrucoes           InstrucoesPagamento
}

// ProcessadorPagamento abstrai o provedor financeiro. CriarCobranca inicia a cobranca:
// provedores sincronos (ex.: o simulado do MVP) devolvem Aprovada ou Recusada na hora;
// provedores reais (PIX, cartao) devolvem Pendente mais as instrucoes de pagamento, e a
// confirmacao chega depois por webhook. ChaveIdempotencia deve ser tratada como idempotency
// key no provedor: repeticoes da mesma intencao recuperam a cobranca sem cobrar novamente.
type ProcessadorPagamento interface {
	CriarCobranca(context.Context, SolicitacaoPagamento) (Cobranca, error)
}

// ReconciliadorPagamento e opcional: provedores assincronos podem implementa-lo para que o sistema
// consulte o status atual de uma cobranca (pela chave de idempotencia) diretamente no provedor,
// sem depender da entrega do webhook. Usado como fallback no polling da tela de pagamento.
// encontrada=false quando o provedor ainda nao tem cobranca para a chave.
type ReconciliadorPagamento interface {
	ReconciliarCobranca(ctx context.Context, chave string) (status StatusCobranca, provedor, identificadorExterno string, encontrada bool, err error)
}

// ItemFrete descreve uma peca a transportar, usada na cotacao de frete.
type ItemFrete struct {
	PesoGramas    int
	AlturaCm      int
	LarguraCm     int
	ComprimentoCm int
	ValorCentavos int64 // valor declarado da peca, para seguro
}

// CotacaoFrete e o resultado de uma cotacao para um pedido (um vendedor).
type CotacaoFrete struct {
	ValorCentavos int64
	Provedor      string
	Servico       string
	PrazoDias     int
}

// CotadorFrete abstrai o provedor de cotacao de frete. A origem e o CEP do vendedor e o
// destino o CEP de entrega; ambos chegam com apenas digitos. Implementacoes devem devolver
// erro em caso de indisponibilidade para que o caso de uso aplique um valor de contingencia.
type CotadorFrete interface {
	Cotar(ctx context.Context, origemCEP, destinoCEP string, itens []ItemFrete) (CotacaoFrete, error)
}

// ConsultorCEP resolve um CEP em um endereco parcial (logradouro, bairro, cidade e estado;
// sem numero/complemento), consultando um provedor externo. Implementacoes devem tratar
// timeouts e indisponibilidade como ErrConsultaCEPIndisponivel e o CEP inexistente como
// ErrNaoEncontrado. O CEP recebido ja vem com apenas digitos (8 caracteres).
type ConsultorCEP interface {
	ConsultarCEP(ctx context.Context, cep string) (cadastros.Endereco, error)
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
