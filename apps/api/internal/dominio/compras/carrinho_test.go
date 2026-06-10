package compras

import "testing"

func TestCarrinhoNaoDuplicaAnuncio(t *testing.T) {
	t.Parallel()

	carrinho := Carrinho{}
	carrinho.Adicionar("anuncio-1")
	carrinho.Adicionar("anuncio-1")

	if len(carrinho.IDsAnuncios) != 1 {
		t.Fatalf("quantidade = %d; esperado 1", len(carrinho.IDsAnuncios))
	}
}
