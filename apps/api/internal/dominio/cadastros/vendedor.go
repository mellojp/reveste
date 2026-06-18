package cadastros

// LimiteItensNaoEnviados e a quantidade de itens nao enviados que bloqueia o vendedor
// para novas vendas, conforme o modelo canonico do MVP.
const LimiteItensNaoEnviados = 3

type PerfilVendedor struct {
	ID               string `json:"id"`
	IDUsuario        string `json:"id_usuario"`
	ItensNaoEnviados int    `json:"itens_nao_enviados"`
	Bloqueado        bool   `json:"bloqueado"`
}

type DadosBancarios struct {
	ID                   string `json:"id"`
	IDUsuario            string `json:"id_usuario"`
	Provedor             string `json:"provedor"`
	IdentificadorExterno string `json:"identificador_externo"`
	Habilitado           bool   `json:"habilitado"`
}
