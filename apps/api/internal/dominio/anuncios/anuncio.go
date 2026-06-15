package anuncios

import (
	"net/url"
	"strings"
	"time"

	"reveste/apps/api/internal/common"
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

const (
	CategoriaVestidos    = "vestidos"
	CategoriaCamisetas   = "camisetas"
	CategoriaCalcas      = "calcas"
	CategoriaSaiasShorts = "saias_e_shorts"
	CategoriaCasacos     = "casacos"
	CategoriaAcessorios  = "acessorios"
	CategoriaCalcados    = "calcados"
	CategoriaOutros      = "outros"
)

func CategoriaValida(categoria string) bool {
	switch strings.ToLower(strings.TrimSpace(categoria)) {
	case CategoriaVestidos, CategoriaCamisetas, CategoriaCalcas,
		CategoriaSaiasShorts, CategoriaCasacos, CategoriaAcessorios,
		CategoriaCalcados, CategoriaOutros:
		return true
	default:
		return false
	}
}

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
		!CategoriaValida(a.Categoria) || a.Tamanho == "" || a.Cor == "" ||
		a.PrecoCentavos <= 0 || !a.EstadoConservacao.Valido() ||
		len(a.Fotos) < 2 || len(a.Fotos) > 5 {
		return common.ErrDadosInvalidos
	}
	for _, foto := range a.Fotos {
		if !URLFotoValida(foto.URL) {
			return common.ErrDadosInvalidos
		}
	}
	return nil
}

func URLFotoValida(valor string) bool {
	endereco, err := url.ParseRequestURI(strings.TrimSpace(valor))
	if err != nil || endereco.Scheme != "https" || endereco.User != nil ||
		endereco.RawQuery != "" || endereco.Fragment != "" {
		return false
	}
	host := strings.ToLower(endereco.Hostname())
	return strings.HasSuffix(host, ".public.blob.vercel-storage.com") &&
		host != ".public.blob.vercel-storage.com"
}

func URLFotoValidaParaHost(valor, hostPermitido string) bool {
	if !URLFotoValida(valor) || strings.TrimSpace(hostPermitido) == "" {
		return false
	}
	endereco, _ := url.ParseRequestURI(strings.TrimSpace(valor))
	return strings.EqualFold(endereco.Hostname(), strings.TrimSpace(hostPermitido))
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
		return common.ErrAnuncioIndisponivel
	}
	if a.IDVendedor == idComprador {
		return common.ErrAnuncioDoProprioAutor
	}
	return nil
}

func (a Anuncio) PodeSerGerenciadoPor(idVendedor string) error {
	if a.IDVendedor != idVendedor {
		return common.ErrNaoPermitido
	}
	if a.Status != StatusAnuncioDisponivel {
		return common.ErrTransicaoInvalida
	}
	return nil
}
