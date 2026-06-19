package interacao

import "time"

type Conversa struct {
	ID        string     `json:"id"`
	IDPedido  string     `json:"id_pedido"`
	Mensagens []Mensagem `json:"mensagens"`
	CriadaEm  time.Time  `json:"criada_em"`
}

type Mensagem struct {
	ID                 string     `json:"id"`
	IDConversa         string     `json:"id_conversa"`
	IDUsuarioRemetente string     `json:"id_usuario_remetente"`
	Conteudo           string     `json:"conteudo"`
	LidaEm             *time.Time `json:"lida_em,omitempty"`
	CriadaEm           time.Time  `json:"criada_em"`
}

func (m *Mensagem) MarcarComoLida(agora time.Time) {
	m.LidaEm = &agora
}

// Tipos canonicos de notificacao. O tipo classifica o evento de origem e permite que a
// apresentacao escolha o tom visual sem interpretar o texto.
const (
	NotificacaoVendaRealizada    = "venda_realizada"
	NotificacaoPedidoEnviado     = "pedido_enviado"
	NotificacaoPedidoRecebido    = "pedido_recebido"
	NotificacaoAvaliacaoRecebida = "avaliacao_recebida"
	NotificacaoMensagemRecebida  = "mensagem_recebida"
)

type Notificacao struct {
	ID        string     `json:"id"`
	IDUsuario string     `json:"id_usuario"`
	Tipo      string     `json:"tipo"`
	Conteudo  string     `json:"conteudo"`
	IDPedido  string     `json:"id_pedido,omitempty"`
	LidaEm    *time.Time `json:"lida_em,omitempty"`
	CriadaEm  time.Time  `json:"criada_em"`
}

func (n Notificacao) Lida() bool {
	return n.LidaEm != nil
}

type Avaliacao struct {
	ID                string    `json:"id"`
	IDPedido          string    `json:"id_pedido"`
	IDUsuarioAutor    string    `json:"id_usuario_autor"`
	IDUsuarioAvaliado string    `json:"id_usuario_avaliado"`
	Nota              int       `json:"nota"`
	Comentario        string    `json:"comentario,omitempty"`
	CriadaEm          time.Time `json:"criada_em"`
}

func (a Avaliacao) Valida() bool {
	return a.Nota >= 1 && a.Nota <= 5 && a.IDUsuarioAutor != a.IDUsuarioAvaliado
}
