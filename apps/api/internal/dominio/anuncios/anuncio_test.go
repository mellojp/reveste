package anuncios

import (
	"errors"
	"testing"

	"reveste/apps/api/internal/common"
)

func TestAnuncioExigeEntreDuasECincoFotos(t *testing.T) {
	t.Parallel()

	anuncio := anuncioValido()
	anuncio.Fotos = anuncio.Fotos[:1]
	if !errors.Is(anuncio.ValidarNovo(), common.ErrDadosInvalidos) {
		t.Fatal("anuncio com uma foto deveria ser invalido")
	}

	anuncio = anuncioValido()
	for len(anuncio.Fotos) < 6 {
		anuncio.Fotos = append(anuncio.Fotos, Foto{URL: "https://exemplo.test/foto.jpg"})
	}
	if !errors.Is(anuncio.ValidarNovo(), common.ErrDadosInvalidos) {
		t.Fatal("anuncio com seis fotos deveria ser invalido")
	}
}

func TestAnuncioDoProprioUsuarioNaoPodeIrAoCarrinho(t *testing.T) {
	t.Parallel()

	anuncio := anuncioValido()
	if err := anuncio.PodeSerAdicionadoAoCarrinho(anuncio.IDVendedor); !errors.Is(err, common.ErrAnuncioDoProprioAutor) {
		t.Fatalf("erro obtido = %v; esperado ErrAnuncioDoProprioAutor", err)
	}
}

func TestAnuncioIndisponivelNaoPodeIrAoCarrinho(t *testing.T) {
	t.Parallel()

	anuncio := anuncioValido()
	anuncio.Status = StatusAnuncioReservado
	if err := anuncio.PodeSerAdicionadoAoCarrinho("outro-usuario"); !errors.Is(err, common.ErrAnuncioIndisponivel) {
		t.Fatalf("erro obtido = %v; esperado ErrAnuncioIndisponivel", err)
	}
}

func TestCategoriaDoAnuncioDeveSerCanonica(t *testing.T) {
	t.Parallel()

	anuncio := anuncioValido()
	anuncio.Categoria = "categoria inventada"

	if !errors.Is(anuncio.ValidarNovo(), common.ErrDadosInvalidos) {
		t.Fatal("categoria livre deveria ser rejeitada")
	}
}

func TestAnuncioRejeitaURLDeFotoInsegura(t *testing.T) {
	anuncio := anuncioValido()
	anuncio.Fotos[0].URL = "javascript:alert(1)"

	if err := anuncio.ValidarNovo(); err == nil {
		t.Fatal("ValidarNovo() deveria rejeitar URL de foto insegura")
	}
}

func anuncioValido() Anuncio {
	return Anuncio{
		ID: "anuncio-1", IDVendedor: "vendedor-1", Titulo: "Jaqueta jeans",
		Descricao: "Jaqueta jeans em excelente estado", Categoria: CategoriaCasacos,
		Tamanho: "M", Cor: "azul", EstadoConservacao: EstadoSeminovo,
		PrecoCentavos: 12_000, Status: StatusAnuncioDisponivel,
		Fotos: []Foto{
			{ID: "foto-1", URL: "https://exemplo.test/1.jpg"},
			{ID: "foto-2", URL: "https://exemplo.test/2.jpg"},
		},
	}
}
