package cadastros

import (
	"net/mail"
	"strings"
	"time"

	"reveste/apps/api/internal/common"
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
	u.EnderecoPrincipal.Logradouro = strings.TrimSpace(u.EnderecoPrincipal.Logradouro)
	u.EnderecoPrincipal.Numero = strings.TrimSpace(u.EnderecoPrincipal.Numero)
	u.EnderecoPrincipal.Complemento = strings.TrimSpace(u.EnderecoPrincipal.Complemento)
	u.EnderecoPrincipal.Bairro = strings.TrimSpace(u.EnderecoPrincipal.Bairro)
	u.EnderecoPrincipal.Cidade = strings.TrimSpace(u.EnderecoPrincipal.Cidade)
	u.EnderecoPrincipal.Estado = strings.ToUpper(strings.TrimSpace(u.EnderecoPrincipal.Estado))
}

func (u Usuario) Validar() error {
	campos := make(map[string]string)
	if len(u.Nome) < 3 {
		campos["nome"] = "Informe o nome completo com pelo menos 3 caracteres."
	}
	if !CPFValido(u.CPF) {
		campos["cpf"] = "Informe um CPF válido."
	}
	enderecoEmail, err := mail.ParseAddress(u.Email)
	if err != nil || enderecoEmail.Address != u.Email {
		campos["email"] = "Informe um e-mail válido."
	}
	if len(u.HashSenha) == 0 {
		campos["senha"] = "Informe uma senha válida."
	}
	if len(u.EnderecoPrincipal.CEP) != 8 {
		campos["endereco.cep"] = "O CEP deve conter 8 dígitos."
	}
	if u.EnderecoPrincipal.Logradouro == "" {
		campos["endereco.logradouro"] = "Informe o logradouro."
	}
	if u.EnderecoPrincipal.Numero == "" {
		campos["endereco.numero"] = "Informe o número."
	}
	if u.EnderecoPrincipal.Bairro == "" {
		campos["endereco.bairro"] = "Informe o bairro."
	}
	if u.EnderecoPrincipal.Cidade == "" {
		campos["endereco.cidade"] = "Informe a cidade."
	}
	if len(u.EnderecoPrincipal.Estado) != 2 {
		campos["endereco.estado"] = "Use a sigla do estado com 2 letras."
	}
	if len(campos) > 0 {
		return common.NovaValidacao(campos)
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
