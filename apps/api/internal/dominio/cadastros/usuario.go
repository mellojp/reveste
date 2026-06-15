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
	u.Telefone = somenteDigitos(u.Telefone)
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
	} else if len(u.Nome) > 150 {
		campos["nome"] = "O nome deve conter no máximo 150 caracteres."
	}
	if !CPFValido(u.CPF) {
		campos["cpf"] = "Informe um CPF válido."
	}
	enderecoEmail, err := mail.ParseAddress(u.Email)
	if err != nil || enderecoEmail.Address != u.Email || len(u.Email) > 254 {
		campos["email"] = "Informe um e-mail válido."
	}
	if u.Telefone != "" && len(u.Telefone) != 10 && len(u.Telefone) != 11 {
		campos["telefone"] = "Informe um telefone com DDD."
	}
	if len(u.HashSenha) == 0 {
		campos["senha"] = "Informe uma senha válida."
	}
	if len(u.EnderecoPrincipal.CEP) != 8 {
		campos["endereco.cep"] = "O CEP deve conter 8 dígitos."
	}
	if u.EnderecoPrincipal.Logradouro == "" {
		campos["endereco.logradouro"] = "Informe o logradouro."
	} else if len(u.EnderecoPrincipal.Logradouro) > 200 {
		campos["endereco.logradouro"] = "O logradouro deve conter no máximo 200 caracteres."
	}
	if u.EnderecoPrincipal.Numero == "" {
		campos["endereco.numero"] = "Informe o número."
	} else if len(u.EnderecoPrincipal.Numero) > 20 {
		campos["endereco.numero"] = "O número deve conter no máximo 20 caracteres."
	}
	if len(u.EnderecoPrincipal.Complemento) > 100 {
		campos["endereco.complemento"] = "O complemento deve conter no máximo 100 caracteres."
	}
	if u.EnderecoPrincipal.Bairro == "" {
		campos["endereco.bairro"] = "Informe o bairro."
	} else if len(u.EnderecoPrincipal.Bairro) > 100 {
		campos["endereco.bairro"] = "O bairro deve conter no máximo 100 caracteres."
	}
	if u.EnderecoPrincipal.Cidade == "" {
		campos["endereco.cidade"] = "Informe a cidade."
	} else if len(u.EnderecoPrincipal.Cidade) > 100 {
		campos["endereco.cidade"] = "A cidade deve conter no máximo 100 caracteres."
	}
	if !estadoBrasileiroValido(u.EnderecoPrincipal.Estado) {
		campos["endereco.estado"] = "Informe uma sigla de estado válida."
	}
	if len(campos) > 0 {
		return common.NovaValidacao(campos)
	}
	return nil
}

func estadoBrasileiroValido(estado string) bool {
	_, existe := estadosBrasileiros[estado]
	return existe
}

var estadosBrasileiros = map[string]struct{}{
	"AC": {}, "AL": {}, "AP": {}, "AM": {}, "BA": {}, "CE": {}, "DF": {},
	"ES": {}, "GO": {}, "MA": {}, "MT": {}, "MS": {}, "MG": {}, "PA": {},
	"PB": {}, "PR": {}, "PE": {}, "PI": {}, "RJ": {}, "RN": {}, "RS": {},
	"RO": {}, "RR": {}, "SC": {}, "SP": {}, "SE": {}, "TO": {},
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
