package casosdeuso

import (
	"context"

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
	if len(carrinho.IDsAnuncios) == 0 {
		return resultado, nil
	}
	lista, err := c.anuncios.ListarAnuncios(ctx, FiltroAnuncios{
		IDsAnuncios:        carrinho.IDsAnuncios,
		IncluirTodosStatus: true,
		Limite:             len(carrinho.IDsAnuncios),
	})
	if err != nil {
		return CarrinhoDetalhado{}, err
	}
	porID := make(map[string]anuncios.Anuncio, len(lista))
	for _, anuncio := range lista {
		porID[anuncio.ID] = anuncio
	}
	for _, idAnuncio := range carrinho.IDsAnuncios {
		anuncio, existe := porID[idAnuncio]
		if !existe {
			continue
		}
		resultado.Anuncios = append(resultado.Anuncios, anuncio)
		if anuncio.Status == anuncios.StatusAnuncioDisponivel {
			resultado.TotalCentavos += anuncio.PrecoCentavos
		}
	}
	return resultado, nil
}
