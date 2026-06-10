package cadastros

import (
	"net/mail"
	"strings"
	"time"

	"reveste/apps/api/internal/dominio/erros"
)

type Endereco struct {
	CEP         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Numero      string `json:"numero"`
	Complemento string `json:"complemento,omitempty"`
	Bairro      string `json:"bairro"`
	Cidade      string `json:"cidade"`
	Estado      string `json:"estado"`
}

type Usuario struct {
	ID                  string    `json:"id"`
	Nome                string    `json:"nome"`
	CPF                 string    `json:"-"`
	Email               string    `json:"email"`
	HashSenha           string    `json:"-"`
	Telefone            string    `json:"telefone,omitempty"`
	EnderecoPrincipal   Endereco  `json:"endereco_principal"`
	ItensNaoEnviados    int       `json:"itens_nao_enviados"`
	BloqueadoParaVendas bool      `json:"bloqueado_para_vendas"`
	CriadoEm            time.Time `json:"criado_em"`
	AtualizadoEm        time.Time `json:"atualizado_em"`
}

func (u *Usuario) Normalizar() {
	u.Nome = strings.TrimSpace(u.Nome)
	u.CPF = NormalizarCPF(u.CPF)
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	u.Telefone = strings.TrimSpace(u.Telefone)
	u.EnderecoPrincipal.CEP = somenteDigitos(u.EnderecoPrincipal.CEP)
	u.EnderecoPrincipal.Estado = strings.ToUpper(strings.TrimSpace(u.EnderecoPrincipal.Estado))
}

func (u Usuario) Validar() error {
	if len(u.Nome) < 3 || !CPFValido(u.CPF) {
		return erros.ErrDadosInvalidos
	}
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return erros.ErrDadosInvalidos
	}
	if len(u.HashSenha) == 0 || len(u.EnderecoPrincipal.CEP) != 8 ||
		u.EnderecoPrincipal.Logradouro == "" || u.EnderecoPrincipal.Numero == "" ||
		u.EnderecoPrincipal.Bairro == "" || u.EnderecoPrincipal.Cidade == "" ||
		len(u.EnderecoPrincipal.Estado) != 2 {
		return erros.ErrDadosInvalidos
	}
	return nil
}

func somenteDigitos(valor string) string {
	resultado := make([]rune, 0, len(valor))
	for _, caractere := range valor {
		if caractere >= '0' && caractere <= '9' {
			resultado = append(resultado, caractere)
		}
	}
	return string(resultado)
}
