package casosdeuso

import (
	"context"
	"time"

	"reveste/apps/api/internal/common"
	dominioanuncios "reveste/apps/api/internal/dominio/anuncios"
)

// Controla operacoes do catalogo,
type ControladorAnuncio struct {
	usuarios OperacoesUsuarios
	anuncios OperacoesAnuncios
	ids      GeradorID
	relogio  Relogio
	hostBlob string
}

func NovoControladorAnuncio(
	usuarios OperacoesUsuarios,
	anuncios OperacoesAnuncios,
	ids GeradorID,
	relogio Relogio,
	hostBlob string,
) *ControladorAnuncio {
	return &ControladorAnuncio{
		usuarios: usuarios, anuncios: anuncios, ids: ids, relogio: relogio,
		hostBlob: hostBlob,
	}
}

type EntradaAnuncio struct {
	Titulo            string
	Descricao         string
	Categoria         string
	Tamanho           string
	Cor               string
	EstadoConservacao dominioanuncios.EstadoConservacao
	PrecoCentavos     int64
	PesoGramas        int
	AlturaCm          int
	LarguraCm         int
	ComprimentoCm     int
	URLsFotos         []string
}

type PerfilPublicoVendedor struct {
	ID                 string    `json:"id"`
	Nome               string    `json:"nome"`
	Cidade             string    `json:"cidade"`
	Estado             string    `json:"estado"`
	MembroDesde        time.Time `json:"membro_desde"`
	QuantidadeAnuncios int       `json:"quantidade_anuncios"`
}

type AnuncioDetalhado struct {
	dominioanuncios.Anuncio
	Vendedor PerfilPublicoVendedor `json:"vendedor"`
}

type PerfilVendedorDetalhado struct {
	Vendedor PerfilPublicoVendedor     `json:"vendedor"`
	Anuncios []dominioanuncios.Anuncio `json:"anuncios"`
}

func (c *ControladorAnuncio) CriarAnuncio(
	ctx context.Context,
	idVendedor string,
	entrada EntradaAnuncio,
) (dominioanuncios.Anuncio, error) {
	if !dominioanuncios.CategoriaValida(entrada.Categoria) {
		return dominioanuncios.Anuncio{}, common.NovaValidacao(map[string]string{
			"categoria": "Selecione uma categoria válida.",
		})
	}
	if !urlsFotosPermitidas(entrada.URLsFotos, c.hostBlob) {
		return dominioanuncios.Anuncio{}, common.NovaValidacao(map[string]string{
			"fotos": "As fotos devem pertencer ao armazenamento oficial da ReVeste.",
		})
	}
	vendedor, err := c.usuarios.BuscarUsuarioPorID(ctx, idVendedor)
	if err != nil {
		return dominioanuncios.Anuncio{}, err
	}
	if vendedor.BloqueadoParaVendas {
		return dominioanuncios.Anuncio{}, common.ErrVendedorBloqueado
	}
	agora := c.relogio.Agora()
	anuncio := dominioanuncios.Anuncio{
		ID: c.ids.Novo(), IDVendedor: idVendedor, Titulo: entrada.Titulo,
		Descricao: entrada.Descricao, Categoria: entrada.Categoria, Tamanho: entrada.Tamanho,
		Cor: entrada.Cor, EstadoConservacao: entrada.EstadoConservacao,
		PrecoCentavos: entrada.PrecoCentavos,
		PesoGramas:    entrada.PesoGramas, AlturaCm: entrada.AlturaCm,
		LarguraCm: entrada.LarguraCm, ComprimentoCm: entrada.ComprimentoCm,
		Status:   dominioanuncios.StatusAnuncioDisponivel,
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

func (c *ControladorAnuncio) ListarAnuncios(
	ctx context.Context,
	filtro FiltroAnuncios,
) ([]dominioanuncios.Anuncio, error) {
	if filtro.Limite <= 0 || filtro.Limite > 50 {
		filtro.Limite = 20
	}
	if filtro.Deslocamento < 0 {
		filtro.Deslocamento = 0
	}
	return c.anuncios.ListarAnuncios(ctx, filtro)
}

func (c *ControladorAnuncio) ObterAnuncio(
	ctx context.Context,
	idAnuncio string,
) (AnuncioDetalhado, error) {
	if idAnuncio == "" {
		return AnuncioDetalhado{}, common.ErrNaoEncontrado
	}
	anuncio, err := c.anuncios.BuscarAnuncioPorID(ctx, idAnuncio)
	if err != nil {
		return AnuncioDetalhado{}, err
	}
	vendedor, err := c.perfilPublicoVendedor(ctx, anuncio.IDVendedor)
	if err != nil {
		return AnuncioDetalhado{}, err
	}
	return AnuncioDetalhado{Anuncio: anuncio, Vendedor: vendedor}, nil
}

func (c *ControladorAnuncio) ListarAnunciosDoVendedor(
	ctx context.Context,
	idVendedor string,
) ([]dominioanuncios.Anuncio, error) {
	return c.ListarAnuncios(ctx, FiltroAnuncios{
		IDVendedor:         idVendedor,
		IncluirTodosStatus: true,
		Limite:             50,
	})
}

func (c *ControladorAnuncio) AtualizarAnuncio(
	ctx context.Context,
	idVendedor,
	idAnuncio string,
	entrada EntradaAnuncio,
) (dominioanuncios.Anuncio, error) {
	atual, err := c.anuncios.BuscarAnuncioPorID(ctx, idAnuncio)
	if err != nil {
		return dominioanuncios.Anuncio{}, err
	}
	if err := atual.PodeSerGerenciadoPor(idVendedor); err != nil {
		return dominioanuncios.Anuncio{}, err
	}
	if !urlsFotosPermitidas(entrada.URLsFotos, c.hostBlob) {
		return dominioanuncios.Anuncio{}, common.NovaValidacao(map[string]string{
			"fotos": "As fotos devem pertencer ao armazenamento oficial da ReVeste.",
		})
	}
	atual.Titulo = entrada.Titulo
	atual.Descricao = entrada.Descricao
	atual.Categoria = entrada.Categoria
	atual.Tamanho = entrada.Tamanho
	atual.Cor = entrada.Cor
	atual.EstadoConservacao = entrada.EstadoConservacao
	atual.PrecoCentavos = entrada.PrecoCentavos
	atual.PesoGramas = entrada.PesoGramas
	atual.AlturaCm = entrada.AlturaCm
	atual.LarguraCm = entrada.LarguraCm
	atual.ComprimentoCm = entrada.ComprimentoCm
	atual.AtualizadoEm = c.relogio.Agora()
	atual.Fotos = make([]dominioanuncios.Foto, 0, len(entrada.URLsFotos))
	for indice, url := range entrada.URLsFotos {
		atual.Fotos = append(atual.Fotos, dominioanuncios.Foto{
			ID: c.ids.Novo(), URL: url, Ordem: indice,
		})
	}
	atual.Normalizar()
	if err := atual.ValidarNovo(); err != nil {
		return dominioanuncios.Anuncio{}, err
	}
	if err := c.anuncios.AtualizarAnuncio(ctx, atual); err != nil {
		return dominioanuncios.Anuncio{}, err
	}
	return atual, nil
}

func urlsFotosPermitidas(urls []string, host string) bool {
	for _, endereco := range urls {
		if !dominioanuncios.URLFotoValidaParaHost(endereco, host) {
			return false
		}
	}
	return true
}

func (c *ControladorAnuncio) ExcluirAnuncio(
	ctx context.Context,
	idVendedor,
	idAnuncio string,
) error {
	anuncio, err := c.anuncios.BuscarAnuncioPorID(ctx, idAnuncio)
	if err != nil {
		return err
	}
	if err := anuncio.PodeSerGerenciadoPor(idVendedor); err != nil {
		return err
	}
	return c.anuncios.ExcluirAnuncio(ctx, idAnuncio, idVendedor, c.relogio.Agora())
}

func (c *ControladorAnuncio) ObterPerfilPublicoVendedor(
	ctx context.Context,
	idVendedor string,
) (PerfilVendedorDetalhado, error) {
	vendedor, err := c.perfilPublicoVendedor(ctx, idVendedor)
	if err != nil {
		return PerfilVendedorDetalhado{}, err
	}
	lista, err := c.ListarAnuncios(ctx, FiltroAnuncios{
		IDVendedor: idVendedor,
		Limite:     50,
	})
	if err != nil {
		return PerfilVendedorDetalhado{}, err
	}
	if lista == nil {
		lista = []dominioanuncios.Anuncio{}
	}
	vendedor.QuantidadeAnuncios = len(lista)
	return PerfilVendedorDetalhado{Vendedor: vendedor, Anuncios: lista}, nil
}

func (c *ControladorAnuncio) perfilPublicoVendedor(
	ctx context.Context,
	idVendedor string,
) (PerfilPublicoVendedor, error) {
	usuario, err := c.usuarios.BuscarUsuarioPorID(ctx, idVendedor)
	if err != nil {
		return PerfilPublicoVendedor{}, err
	}
	return PerfilPublicoVendedor{
		ID: idVendedor, Nome: usuario.Nome,
		Cidade:      usuario.EnderecoPrincipal.Cidade,
		Estado:      usuario.EnderecoPrincipal.Estado,
		MembroDesde: usuario.CriadoEm,
	}, nil
}
