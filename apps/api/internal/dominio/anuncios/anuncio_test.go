package anuncios

import (
	"errors"
	"testing"

	errosdominio "reveste/apps/api/internal/dominio/erros"
)

func TestAnuncioExigeEntreDuasECincoFotos(t *testing.T) {
	t.Parallel()

	anuncio := anuncioValido()
	anuncio.Fotos = anuncio.Fotos[:1]
	if !errors.Is(anuncio.ValidarNovo(), errosdominio.ErrDadosInvalidos) {
		t.Fatal("anuncio com uma foto deveria ser invalido")
	}

	anuncio = anuncioValido()
	for len(anuncio.Fotos) < 6 {
		anuncio.Fotos = append(anuncio.Fotos, Foto{URL: "https://exemplo.test/foto.jpg"})
	}
	if !errors.Is(anuncio.ValidarNovo(), errosdominio.ErrDadosInvalidos) {
		t.Fatal("anuncio com seis fotos deveria ser invalido")
	}
}

func TestAnuncioDoProprioUsuarioNaoPodeIrAoCarrinho(t *testing.T) {
	t.Parallel()

	anuncio := anuncioValido()
	if err := anuncio.PodeSerAdicionadoAoCarrinho(anuncio.IDVendedor); !errors.Is(err, errosdominio.ErrAnuncioDoProprioAutor) {
		t.Fatalf("erro obtido = %v; esperado ErrAnuncioDoProprioAutor", err)
	}
}

func TestAnuncioIndisponivelNaoPodeIrAoCarrinho(t *testing.T) {
	t.Parallel()

	anuncio := anuncioValido()
	anuncio.Status = StatusAnuncioReservado
	if err := anuncio.PodeSerAdicionadoAoCarrinho("outro-usuario"); !errors.Is(err, errosdominio.ErrAnuncioIndisponivel) {
		t.Fatalf("erro obtido = %v; esperado ErrAnuncioIndisponivel", err)
	}
}

func anuncioValido() Anuncio {
	return Anuncio{
		ID: "anuncio-1", IDVendedor: "vendedor-1", Titulo: "Jaqueta jeans",
		Descricao: "Jaqueta jeans em excelente estado", Categoria: "jaqueta",
		Tamanho: "M", Cor: "azul", EstadoConservacao: EstadoSeminovo,
		PrecoCentavos: 12_000, Status: StatusAnuncioDisponivel,
		Fotos: []Foto{
			{ID: "foto-1", URL: "https://exemplo.test/1.jpg"},
			{ID: "foto-2", URL: "https://exemplo.test/2.jpg"},
		},
	}
}
