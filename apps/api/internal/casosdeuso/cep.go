package casosdeuso

import (
	"context"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/cadastros"
)

// ControladorCEP coordena a consulta de CEP: normaliza e valida o formato antes de delegar
// ao provedor externo, devolvendo um endereco parcial para preencher formularios.
type ControladorCEP struct {
	consultor ConsultorCEP
}

func NovoControladorCEP(consultor ConsultorCEP) *ControladorCEP {
	return &ControladorCEP{consultor: consultor}
}

// Consultar valida o formato (8 digitos) e busca o endereco no provedor. O endereco
// devolvido nao traz numero nem complemento, que sao informados pelo usuario.
func (c *ControladorCEP) Consultar(ctx context.Context, cep string) (cadastros.Endereco, error) {
	cep = apenasDigitos(cep)
	if len(cep) != 8 {
		return cadastros.Endereco{}, common.NovaValidacao(map[string]string{
			"cep": "O CEP deve conter 8 dígitos.",
		})
	}
	return c.consultor.ConsultarCEP(ctx, cep)
}

func apenasDigitos(valor string) string {
	resultado := make([]rune, 0, len(valor))
	for _, caractere := range valor {
		if caractere >= '0' && caractere <= '9' {
			resultado = append(resultado, caractere)
		}
	}
	return string(resultado)
}
