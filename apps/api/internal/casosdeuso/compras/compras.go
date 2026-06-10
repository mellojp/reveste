package compras

import (
	"context"
	"errors"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/dominio/anuncios"
	dominiocompras "reveste/apps/api/internal/dominio/compras"
	errosdominio "reveste/apps/api/internal/dominio/erros"
)

type FluxoCarrinho struct {
	anuncios  casosdeuso.OperacoesAnuncios
	carrinhos casosdeuso.OperacoesCarrinhos
	ids       casosdeuso.GeradorID
	relogio   casosdeuso.Relogio
}

func NovoFluxoCarrinho(
	anuncios casosdeuso.OperacoesAnuncios,
	carrinhos casosdeuso.OperacoesCarrinhos,
	ids casosdeuso.GeradorID,
	relogio casosdeuso.Relogio,
) *FluxoCarrinho {
	return &FluxoCarrinho{anuncios: anuncios, carrinhos: carrinhos, ids: ids, relogio: relogio}
}

type CarrinhoDetalhado struct {
	ID            string             `json:"id"`
	IDUsuario     string             `json:"id_usuario"`
	Anuncios      []anuncios.Anuncio `json:"anuncios"`
	TotalCentavos int64              `json:"total_centavos"`
}

func (c *FluxoCarrinho) ObterCarrinho(ctx context.Context, idUsuario string) (CarrinhoDetalhado, error) {
	carrinho, err := c.carrinhos.ObterOuCriarCarrinho(ctx, c.ids.Novo(), idUsuario, c.relogio.Agora())
	if err != nil {
		return CarrinhoDetalhado{}, err
	}
	return c.detalharCarrinho(ctx, carrinho)
}

func (c *FluxoCarrinho) AdicionarAoCarrinho(
	ctx context.Context,
	idUsuario,
	idAnuncio string,
) (CarrinhoDetalhado, error) {
	anuncio, err := c.anuncios.BuscarAnuncioPorID(ctx, idAnuncio)
	if err != nil {
		return CarrinhoDetalhado{}, err
	}
	if err := anuncio.PodeSerAdicionadoAoCarrinho(idUsuario); err != nil {
		return CarrinhoDetalhado{}, err
	}
	agora := c.relogio.Agora()
	carrinho, err := c.carrinhos.ObterOuCriarCarrinho(ctx, c.ids.Novo(), idUsuario, agora)
	if err != nil {
		return CarrinhoDetalhado{}, err
	}
	carrinho.Adicionar(idAnuncio)
	carrinho.AtualizadoEm = agora
	if err := c.carrinhos.SalvarCarrinho(ctx, carrinho); err != nil {
		return CarrinhoDetalhado{}, err
	}
	return c.detalharCarrinho(ctx, carrinho)
}

func (c *FluxoCarrinho) RemoverDoCarrinho(
	ctx context.Context,
	idUsuario,
	idAnuncio string,
) (CarrinhoDetalhado, error) {
	agora := c.relogio.Agora()
	carrinho, err := c.carrinhos.ObterOuCriarCarrinho(ctx, c.ids.Novo(), idUsuario, agora)
	if err != nil {
		return CarrinhoDetalhado{}, err
	}
	carrinho.Remover(idAnuncio)
	carrinho.AtualizadoEm = agora
	if err := c.carrinhos.SalvarCarrinho(ctx, carrinho); err != nil {
		return CarrinhoDetalhado{}, err
	}
	return c.detalharCarrinho(ctx, carrinho)
}

func (c *FluxoCarrinho) detalharCarrinho(
	ctx context.Context,
	carrinho dominiocompras.Carrinho,
) (CarrinhoDetalhado, error) {
	resultado := CarrinhoDetalhado{ID: carrinho.ID, IDUsuario: carrinho.IDUsuario}
	for _, idAnuncio := range carrinho.IDsAnuncios {
		anuncio, err := c.anuncios.BuscarAnuncioPorID(ctx, idAnuncio)
		if errors.Is(err, errosdominio.ErrNaoEncontrado) {
			continue
		}
		if err != nil {
			return CarrinhoDetalhado{}, err
		}
		resultado.Anuncios = append(resultado.Anuncios, anuncio)
		resultado.TotalCentavos += anuncio.PrecoCentavos
	}
	return resultado, nil
}
