package cadastros

import "testing"

func TestCPFValido(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nome     string
		cpf      string
		esperado bool
	}{
		{nome: "formatado", cpf: "529.982.247-25", esperado: true},
		{nome: "somente digitos", cpf: "52998224725", esperado: true},
		{nome: "digito incorreto", cpf: "52998224724", esperado: false},
		{nome: "todos iguais", cpf: "11111111111", esperado: false},
		{nome: "incompleto", cpf: "123", esperado: false},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nome, func(t *testing.T) {
			t.Parallel()
			if obtido := CPFValido(caso.cpf); obtido != caso.esperado {
				t.Fatalf("CPFValido(%q) = %v; esperado %v", caso.cpf, obtido, caso.esperado)
			}
		})
	}
}
