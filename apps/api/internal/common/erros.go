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
	// ErrConsultaCEPIndisponivel indica falha transitoria ao consultar o provedor de CEP
	// (timeout, indisponibilidade). O CEP inexistente usa ErrNaoEncontrado.
	ErrConsultaCEPIndisponivel = errors.New("consulta de cep indisponivel")
	// ErrCotacaoFreteIndisponivel indica falha ao cotar o frete no provedor externo. O caso
	// de uso de checkout trata esse erro aplicando um valor de frete de contingencia.
	ErrCotacaoFreteIndisponivel = errors.New("cotacao de frete indisponivel")
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
