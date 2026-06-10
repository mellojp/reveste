package anuncios

import (
	"context"

	"reveste/apps/api/internal/casosdeuso"
	dominioanuncios "reveste/apps/api/internal/dominio/anuncios"
)

type FluxoAnuncio struct {
	usuarios casosdeuso.OperacoesUsuarios
	anuncios casosdeuso.OperacoesAnuncios
	ids      casosdeuso.GeradorID
	relogio  casosdeuso.Relogio
}

func NovoFluxoAnuncio(
	usuarios casosdeuso.OperacoesUsuarios,
	anuncios casosdeuso.OperacoesAnuncios,
	ids casosdeuso.GeradorID,
	relogio casosdeuso.Relogio,
) *FluxoAnuncio {
	return &FluxoAnuncio{usuarios: usuarios, anuncios: anuncios, ids: ids, relogio: relogio}
}

type EntradaAnuncio struct {
	Titulo            string
	Descricao         string
	Categoria         string
	Tamanho           string
	Cor               string
	EstadoConservacao dominioanuncios.EstadoConservacao
	PrecoCentavos     int64
	URLsFotos         []string
}

func (c *FluxoAnuncio) CriarAnuncio(
	ctx context.Context,
	idVendedor string,
	entrada EntradaAnuncio,
) (dominioanuncios.Anuncio, error) {
	if _, err := c.usuarios.BuscarUsuarioPorID(ctx, idVendedor); err != nil {
		return dominioanuncios.Anuncio{}, err
	}
	agora := c.relogio.Agora()
	anuncio := dominioanuncios.Anuncio{
		ID: c.ids.Novo(), IDVendedor: idVendedor, Titulo: entrada.Titulo,
		Descricao: entrada.Descricao, Categoria: entrada.Categoria, Tamanho: entrada.Tamanho,
		Cor: entrada.Cor, EstadoConservacao: entrada.EstadoConservacao,
		PrecoCentavos: entrada.PrecoCentavos, Status: dominioanuncios.StatusAnuncioDisponivel,
		CriadoEm: agora, AtualizadoEm: agora,
	}
	for indice, url := range entrada.URLsFotos {
		anuncio.Fotos = append(anuncio.Fotos, dominioanuncios.Foto{
			ID: c.ids.Novo(), URL: url, Ordem: indice,
		})
	}
	anuncio.Normalizar()
	if err := anuncio.ValidarNovo(); err != nil {
		return dominioanuncios.Anuncio{}, err
	}
	if err := c.anuncios.CriarAnuncio(ctx, anuncio); err != nil {
		return dominioanuncios.Anuncio{}, err
	}
	return anuncio, nil
}

func (c *FluxoAnuncio) ListarAnuncios(
	ctx context.Context,
	filtro casosdeuso.FiltroAnuncios,
) ([]dominioanuncios.Anuncio, error) {
	if filtro.Limite <= 0 || filtro.Limite > 50 {
		filtro.Limite = 20
	}
	return c.anuncios.ListarAnuncios(ctx, filtro)
}
