package cadastros

import "unicode"

func NormalizarCPF(valor string) string {
	digitos := make([]rune, 0, 11)
	for _, caractere := range valor {
		if unicode.IsDigit(caractere) {
			digitos = append(digitos, caractere)
		}
	}
	return string(digitos)
}

func CPFValido(valor string) bool {
	cpf := NormalizarCPF(valor)
	if len(cpf) != 11 || todosDigitosIguais(cpf) {
		return false
	}

	return calcularDigitoCPF(cpf[:9], 10) == int(cpf[9]-'0') &&
		calcularDigitoCPF(cpf[:10], 11) == int(cpf[10]-'0')
}

func todosDigitosIguais(cpf string) bool {
	for i := 1; i < len(cpf); i++ {
		if cpf[i] != cpf[0] {
			return false
		}
	}
	return true
}

func calcularDigitoCPF(base string, pesoInicial int) int {
	soma := 0
	for i := range base {
		soma += int(base[i]-'0') * (pesoInicial - i)
	}
	resto := (soma * 10) % 11
	if resto == 10 {
		return 0
	}
	return resto
}
