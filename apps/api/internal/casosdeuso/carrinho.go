package casosdeuso

import (
	"context"
	"errors"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
	dominiocompras "reveste/apps/api/internal/dominio/compras"
)

// Controla operacoes do carrinho
type ControladorCarrinho struct {
	anuncios  OperacoesAnuncios
	carrinhos OperacoesCarrinhos
	ids       GeradorID
	relogio   Relogio
}

func NovoControladorCarrinho(
	anuncios OperacoesAnuncios,
	carrinhos OperacoesCarrinhos,
	ids GeradorID,
	relogio Relogio,
) *ControladorCarrinho {
	return &ControladorCarrinho{anuncios: anuncios, carrinhos: carrinhos, ids: ids, relogio: relogio}
}

type CarrinhoDetalhado struct {
	ID            string             `json:"id"`
	IDUsuario     string             `json:"id_usuario"`
	Anuncios      []anuncios.Anuncio `json:"anuncios"`
	TotalCentavos int64              `json:"total_centavos"`
}

func (c *ControladorCarrinho) ObterCarrinho(ctx context.Context, idUsuario string) (CarrinhoDetalhado, error) {
	carrinho, err := c.carrinhos.ObterOuCriarCarrinho(ctx, c.ids.Novo(), idUsuario, c.relogio.Agora())
	if err != nil {
		return CarrinhoDetalhado{}, err
	}
	return c.detalharCarrinho(ctx, carrinho)
}

func (c *ControladorCarrinho) AdicionarAoCarrinho(
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
	carrinho, err := c.carrinhos.AdicionarAnuncioAoCarrinho(
		ctx, c.ids.Novo(), idUsuario, idAnuncio, agora,
	)
	if err != nil {
		return CarrinhoDetalhado{}, err
	}
	return c.detalharCarrinho(ctx, carrinho)
}

func (c *ControladorCarrinho) RemoverDoCarrinho(
	ctx context.Context,
	idUsuario,
	idAnuncio string,
) (CarrinhoDetalhado, error) {
	agora := c.relogio.Agora()
	carrinho, err := c.carrinhos.RemoverAnuncioDoCarrinho(
		ctx, c.ids.Novo(), idUsuario, idAnuncio, agora,
	)
	if err != nil {
		return CarrinhoDetalhado{}, err
	}
	return c.detalharCarrinho(ctx, carrinho)
}

func (c *ControladorCarrinho) detalharCarrinho(
	ctx context.Context,
	carrinho dominiocompras.Carrinho,
) (CarrinhoDetalhado, error) {
	resultado := CarrinhoDetalhado{
		ID: carrinho.ID, IDUsuario: carrinho.IDUsuario, Anuncios: []anuncios.Anuncio{},
	}
	for _, idAnuncio := range carrinho.IDsAnuncios {
		anuncio, err := c.anuncios.BuscarAnuncioPorID(ctx, idAnuncio)
		if errors.Is(err, common.ErrNaoEncontrado) {
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
