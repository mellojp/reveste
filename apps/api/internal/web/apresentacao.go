package web

import (
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
)

func funcoesApresentacaoTemplates() template.FuncMap {
	return template.FuncMap{
		"formatarDinheiro":  formatarDinheiro,
		"formatarData":      formatarData,
		"extrairAno":        func(valor time.Time) int { return valor.Year() },
		"formatarRotulo":    formatarRotulo,
		"iniciais":          iniciais,
		"primeiroNome":      primeiroNome,
		"incrementar":       func(valor int) int { return valor + 1 },
		"anuncioNoCarrinho": carrinhoContemAnuncio,
		"contarDisponiveis": contarAnunciosDisponiveis,
		"contarIndisponiveis": func(itens []anuncios.Anuncio) int {
			return len(itens) - contarAnunciosDisponiveis(itens)
		},
		"fotoCapa": fotoCapa,
		"contextoCartaoAnuncio": func(item anuncios.Anuncio, contexto contextoDocumento) contextoCartaoAnuncio {
			idUsuario := ""
			if contexto.UsuarioAutenticado != nil {
				idUsuario = contexto.UsuarioAutenticado.ID
			}
			return contextoCartaoAnuncio{
				Anuncio: item, IDUsuarioAutenticado: idUsuario, URLRetorno: contexto.URLRetorno,
				EstaNoCarrinho: carrinhoContemAnuncio(contexto.CarrinhoAutenticado, item.ID),
			}
		},
		"contextoErroCampo": func(erros map[string]string, nomeCampo string) contextoMensagemCampo {
			return contextoMensagemCampo{
				ErrosValidacao: erros,
				NomeCampo:      nomeCampo,
				IDMensagem:     "erro-" + strings.ReplaceAll(nomeCampo, ".", "-"),
			}
		},
		"temErroCampo": func(erros map[string]string, nome string) bool {
			return mensagemErroCampo(erros, nome) != ""
		},
		"formatarMesAno":    formatarMesAno,
		"mensagemErroCampo": mensagemErroCampo,
		"valorFormulario": func(valores map[string]string, nome string) string {
			return valores[nome]
		},
	}
}

func mensagemErroCampo(campos map[string]string, nome string) string {
	if mensagem := campos[nome]; mensagem != "" {
		return mensagem
	}
	return campos["endereco."+nome]
}

func apresentarErroCasoUso(err error) (string, map[string]string) {
	var validacao common.ErroValidacao
	var conflito common.ErroConflitoCampo
	switch {
	case errors.As(err, &validacao):
		return "Revise os campos destacados.", validacao.Campos
	case errors.As(err, &conflito):
		return "Já existe uma conta com os campos destacados.", conflito.Campos
	case errors.Is(err, common.ErrNaoAutorizado):
		return "E-mail, CPF ou senha inválidos.", map[string]string{}
	case errors.Is(err, common.ErrNaoPermitido):
		return "Você não pode realizar esta operação.", map[string]string{}
	case errors.Is(err, common.ErrAnuncioIndisponivel):
		return "Este anúncio não está mais disponível.", map[string]string{}
	default:
		return "Não foi possível concluir a operação.", map[string]string{}
	}
}

func formatarDinheiro(centavos int64) string {
	reais := float64(centavos) / 100
	texto := strconv.FormatFloat(reais, 'f', 2, 64)
	partes := strings.Split(texto, ".")
	return "R$ " + partes[0] + "," + partes[1]
}

func formatarData(valor time.Time) string {
	if valor.IsZero() {
		return ""
	}
	return valor.Format("02/01/2006")
}

func formatarMesAno(valor time.Time) string {
	meses := [...]string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}
	if valor.IsZero() {
		return ""
	}
	return meses[valor.Month()-1] + " de " + strconv.Itoa(valor.Year())
}

func formatarRotulo(valor any) string {
	texto := strings.ReplaceAll(fmt.Sprint(valor), "_", " ")
	if texto == "" {
		return ""
	}
	return strings.ToUpper(texto[:1]) + texto[1:]
}

func iniciais(nome string) string {
	partes := strings.Fields(nome)
	resultado := ""
	for i := 0; i < len(partes) && i < 2; i++ {
		resultado += strings.ToUpper(string([]rune(partes[i])[0]))
	}
	return resultado
}

func primeiroNome(nome string) string {
	partes := strings.Fields(nome)
	if len(partes) == 0 {
		return ""
	}
	return partes[0]
}

func carrinhoContemAnuncio(carrinho casosdeuso.CarrinhoDetalhado, id string) bool {
	for _, item := range carrinho.Anuncios {
		if item.ID == id {
			return true
		}
	}
	return false
}

func contarAnunciosDisponiveis(itens []anuncios.Anuncio) int {
	total := 0
	for _, item := range itens {
		if item.Status == anuncios.StatusAnuncioDisponivel {
			total++
		}
	}
	return total
}

func fotoCapa(item anuncios.Anuncio) string {
	if len(item.Fotos) == 0 {
		return ""
	}
	return item.Fotos[0].URL
}
