package common

import "errors"

var (
	ErrDadosInvalidos        = errors.New("dados invalidos")
	ErrNaoEncontrado         = errors.New("recurso nao encontrado")
	ErrConflito              = errors.New("recurso ja existente")
	ErrNaoAutorizado         = errors.New("nao autorizado")
	ErrAnuncioIndisponivel   = errors.New("anuncio indisponivel")
	ErrAnuncioDoProprioAutor = errors.New("nao e permitido adicionar o proprio anuncio ao carrinho")
	ErrTransicaoInvalida     = errors.New("transicao de estado invalida")
)
