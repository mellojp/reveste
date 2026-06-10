package compras

import "time"

type Carrinho struct {
	ID           string    `json:"id"`
	IDUsuario    string    `json:"id_usuario"`
	IDsAnuncios  []string  `json:"ids_anuncios"`
	CriadoEm     time.Time `json:"criado_em"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}

func (c Carrinho) Contem(idAnuncio string) bool {
	for _, id := range c.IDsAnuncios {
		if id == idAnuncio {
			return true
		}
	}
	return false
}

func (c *Carrinho) Adicionar(idAnuncio string) {
	if !c.Contem(idAnuncio) {
		c.IDsAnuncios = append(c.IDsAnuncios, idAnuncio)
	}
}

func (c *Carrinho) Remover(idAnuncio string) {
	for indice, id := range c.IDsAnuncios {
		if id == idAnuncio {
			c.IDsAnuncios = append(c.IDsAnuncios[:indice], c.IDsAnuncios[indice+1:]...)
			return
		}
	}
}
