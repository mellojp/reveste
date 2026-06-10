package cadastros

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
