package anuncios

import (
	"strings"
	"time"

	"reveste/apps/api/internal/dominio/erros"
)

type StatusAnuncio string

const (
	StatusAnuncioDisponivel StatusAnuncio = "disponivel"
	StatusAnuncioReservado  StatusAnuncio = "reservado"
	StatusAnuncioVendido    StatusAnuncio = "vendido"
	StatusAnuncioSuspenso   StatusAnuncio = "suspenso"
	StatusAnuncioExcluido   StatusAnuncio = "excluido"
)

type EstadoConservacao string

const (
	EstadoNovo       EstadoConservacao = "novo"
	EstadoSeminovo   EstadoConservacao = "seminovo"
	EstadoUsado      EstadoConservacao = "usado"
	EstadoMuitoUsado EstadoConservacao = "muito_usado"
	EstadoDesgastado EstadoConservacao = "desgastado"
)

type Foto struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Ordem   int    `json:"ordem"`
	Legenda string `json:"legenda,omitempty"`
}

type Anuncio struct {
	ID                string            `json:"id"`
	IDVendedor        string            `json:"id_vendedor"`
	Titulo            string            `json:"titulo"`
	Descricao         string            `json:"descricao"`
	Categoria         string            `json:"categoria"`
	Tamanho           string            `json:"tamanho"`
	Cor               string            `json:"cor"`
	EstadoConservacao EstadoConservacao `json:"estado_conservacao"`
	PrecoCentavos     int64             `json:"preco_centavos"`
	Status            StatusAnuncio     `json:"status"`
	Fotos             []Foto            `json:"fotos"`
	CriadoEm          time.Time         `json:"criado_em"`
	AtualizadoEm      time.Time         `json:"atualizado_em"`
	ExcluidoEm        *time.Time        `json:"excluido_em,omitempty"`
}

func (a *Anuncio) Normalizar() {
	a.Titulo = strings.TrimSpace(a.Titulo)
	a.Descricao = strings.TrimSpace(a.Descricao)
	a.Categoria = strings.ToLower(strings.TrimSpace(a.Categoria))
	a.Tamanho = strings.ToUpper(strings.TrimSpace(a.Tamanho))
	a.Cor = strings.ToLower(strings.TrimSpace(a.Cor))
	for indice := range a.Fotos {
		a.Fotos[indice].URL = strings.TrimSpace(a.Fotos[indice].URL)
		a.Fotos[indice].Ordem = indice
	}
}

func (a Anuncio) ValidarNovo() error {
	if a.IDVendedor == "" || len(a.Titulo) < 3 || len(a.Descricao) < 10 ||
		a.Categoria == "" || a.Tamanho == "" || a.Cor == "" ||
		a.PrecoCentavos <= 0 || !a.EstadoConservacao.Valido() ||
		len(a.Fotos) < 2 || len(a.Fotos) > 5 {
		return erros.ErrDadosInvalidos
	}
	for _, foto := range a.Fotos {
		if foto.URL == "" {
			return erros.ErrDadosInvalidos
		}
	}
	return nil
}

func (e EstadoConservacao) Valido() bool {
	switch e {
	case EstadoNovo, EstadoSeminovo, EstadoUsado, EstadoMuitoUsado, EstadoDesgastado:
		return true
	default:
		return false
	}
}

func (a Anuncio) PodeSerAdicionadoAoCarrinho(idComprador string) error {
	if a.Status != StatusAnuncioDisponivel {
		return erros.ErrAnuncioIndisponivel
	}
	if a.IDVendedor == idComprador {
		return erros.ErrAnuncioDoProprioAutor
	}
	return nil
}
