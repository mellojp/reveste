package cadastros

import (
	"errors"
	"testing"

	"reveste/apps/api/internal/common"
)

func TestValidarUsuarioInformaCamposInvalidos(t *testing.T) {
	usuario := Usuario{
		Nome:      "A",
		CPF:       "123",
		Email:     "invalido",
		HashSenha: "hash",
		EnderecoPrincipal: Endereco{
			CEP:    "1",
			Estado: "S",
		},
	}

	err := usuario.Validar()

	var validacao common.ErroValidacao
	if !errors.As(err, &validacao) {
		t.Fatalf("erro = %v; esperado ErroValidacao", err)
	}
	for _, campo := range []string{
		"nome", "cpf", "email", "endereco.cep", "endereco.logradouro",
		"endereco.numero", "endereco.bairro", "endereco.cidade", "endereco.estado",
	} {
		if validacao.Campos[campo] == "" {
			t.Errorf("mensagem ausente para %q", campo)
		}
	}
}

func TestNormalizarEValidarContatoEEstadoBrasileiros(t *testing.T) {
	usuario := Usuario{
		Nome:      "Pessoa Teste",
		CPF:       "529.982.247-25",
		Email:     "pessoa@teste.local",
		HashSenha: "hash",
		Telefone:  "(79) 99999-9999",
		EnderecoPrincipal: Endereco{
			CEP: "49000-000", Logradouro: "Rua Teste", Numero: "10",
			Bairro: "Centro", Cidade: "Aracaju", Estado: "se",
		},
	}

	usuario.Normalizar()

	if usuario.Telefone != "79999999999" || usuario.EnderecoPrincipal.Estado != "SE" {
		t.Fatalf("normalização inesperada: telefone=%q estado=%q", usuario.Telefone, usuario.EnderecoPrincipal.Estado)
	}
	if err := usuario.Validar(); err != nil {
		t.Fatalf("usuário válido rejeitado: %v", err)
	}

	usuario.Telefone = "123"
	usuario.EnderecoPrincipal.Estado = "XX"
	err := usuario.Validar()
	var validacao common.ErroValidacao
	if !errors.As(err, &validacao) {
		t.Fatalf("erro = %v; esperado ErroValidacao", err)
	}
	if validacao.Campos["telefone"] == "" || validacao.Campos["endereco.estado"] == "" {
		t.Fatalf("erros esperados ausentes: %+v", validacao.Campos)
	}
}
