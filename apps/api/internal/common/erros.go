package common

import "errors"

var (
	ErrDadosInvalidos        = errors.New("dados invalidos")
	ErrNaoEncontrado         = errors.New("recurso nao encontrado")
	ErrConflito              = errors.New("recurso ja existente")
	ErrNaoAutorizado         = errors.New("nao autorizado")
	ErrNaoPermitido          = errors.New("operacao nao permitida")
	ErrAnuncioIndisponivel   = errors.New("anuncio indisponivel")
	ErrAnuncioDoProprioAutor = errors.New("nao e permitido adicionar o proprio anuncio ao carrinho")
	ErrVendedorBloqueado     = errors.New("conta bloqueada para novas vendas")
	ErrTransicaoInvalida     = errors.New("transicao de estado invalida")
	ErrServicoIndisponivel   = errors.New("servico temporariamente indisponivel")
	ErrCarrinhoVazio         = errors.New("carrinho sem itens para finalizar")
	ErrSemItensDisponiveis   = errors.New("nenhum item do carrinho esta disponivel")
	ErrPagamentoRecusado     = errors.New("pagamento recusado pelo provedor")
)

type ErroValidacao struct {
	Campos map[string]string
}

func (e ErroValidacao) Error() string {
	return ErrDadosInvalidos.Error()
}

func (e ErroValidacao) Unwrap() error {
	return ErrDadosInvalidos
}

func NovaValidacao(campos map[string]string) error {
	return ErroValidacao{Campos: campos}
}

type ErroConflitoCampo struct {
	Campos map[string]string
}

func (e ErroConflitoCampo) Error() string {
	return ErrConflito.Error()
}

func (e ErroConflitoCampo) Unwrap() error {
	return ErrConflito
}

func NovoConflitoCampo(campo, mensagem string) error {
	return ErroConflitoCampo{Campos: map[string]string{campo: mensagem}}
}
