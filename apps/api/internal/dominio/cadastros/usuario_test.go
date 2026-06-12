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
