package cadastros

import (
	"net/mail"
	"strings"
	"time"

	"reveste/apps/api/internal/common"
)

type Endereco struct {
	ID          string `json:"id,omitempty"`
	CEP         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Numero      string `json:"numero"`
	Complemento string `json:"complemento,omitempty"`
	Bairro      string `json:"bairro"`
	Cidade      string `json:"cidade"`
	Estado      string `json:"estado"`
	Principal   bool   `json:"principal,omitempty"`
}

func (e *Endereco) Normalizar() {
	e.CEP = somenteDigitos(e.CEP)
	e.Logradouro = strings.TrimSpace(e.Logradouro)
	e.Numero = strings.TrimSpace(e.Numero)
	e.Complemento = strings.TrimSpace(e.Complemento)
	e.Bairro = strings.TrimSpace(e.Bairro)
	e.Cidade = strings.TrimSpace(e.Cidade)
	e.Estado = strings.ToUpper(strings.TrimSpace(e.Estado))
}

// Validar devolve as mensagens de erro por campo (chaves sem prefixo: "cep", "logradouro"...).
// O mapa vazio significa endereco valido.
func (e Endereco) Validar() map[string]string {
	campos := make(map[string]string)
	if len(e.CEP) != 8 {
		campos["cep"] = "O CEP deve conter 8 dígitos."
	}
	if e.Logradouro == "" {
		campos["logradouro"] = "Informe o logradouro."
	} else if len(e.Logradouro) > 200 {
		campos["logradouro"] = "O logradouro deve conter no máximo 200 caracteres."
	}
	if e.Numero == "" {
		campos["numero"] = "Informe o número."
	} else if len(e.Numero) > 20 {
		campos["numero"] = "O número deve conter no máximo 20 caracteres."
	}
	if len(e.Complemento) > 100 {
		campos["complemento"] = "O complemento deve conter no máximo 100 caracteres."
	}
	if e.Bairro == "" {
		campos["bairro"] = "Informe o bairro."
	} else if len(e.Bairro) > 100 {
		campos["bairro"] = "O bairro deve conter no máximo 100 caracteres."
	}
	if e.Cidade == "" {
		campos["cidade"] = "Informe a cidade."
	} else if len(e.Cidade) > 100 {
		campos["cidade"] = "A cidade deve conter no máximo 100 caracteres."
	}
	if !estadoBrasileiroValido(e.Estado) {
		campos["estado"] = "Informe uma sigla de estado válida."
	}
	return campos
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
	u.EnderecoPrincipal.Normalizar()
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
	for campo, mensagem := range u.EnderecoPrincipal.Validar() {
		campos["endereco."+campo] = mensagem
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
