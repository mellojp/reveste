package casosdeuso

import (
	"context"
	"errors"
	"testing"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/cadastros"
)

type consultorCEPFake struct {
	chamado  bool
	cepVisto string
	endereco cadastros.Endereco
	err      error
}

func (c *consultorCEPFake) ConsultarCEP(_ context.Context, cep string) (cadastros.Endereco, error) {
	c.chamado = true
	c.cepVisto = cep
	return c.endereco, c.err
}

func TestControladorCEPRejeitaFormatoInvalido(t *testing.T) {
	consultor := &consultorCEPFake{}
	_, err := NovoControladorCEP(consultor).Consultar(context.Background(), "123")

	var validacao common.ErroValidacao
	if !errors.As(err, &validacao) {
		t.Fatalf("esperava erro de validacao, veio %v", err)
	}
	if consultor.chamado {
		t.Fatal("o provedor nao deveria ser chamado para CEP invalido")
	}
}

func TestControladorCEPNormalizaAntesDeConsultar(t *testing.T) {
	consultor := &consultorCEPFake{endereco: cadastros.Endereco{Cidade: "São Paulo"}}
	endereco, err := NovoControladorCEP(consultor).Consultar(context.Background(), "01310-100")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if consultor.cepVisto != "01310100" {
		t.Fatalf("o provedor deveria receber apenas digitos, veio %q", consultor.cepVisto)
	}
	if endereco.Cidade != "São Paulo" {
		t.Fatalf("endereco devolvido inesperado: %+v", endereco)
	}
}
