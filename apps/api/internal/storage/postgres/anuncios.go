package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
)

func (s *Store) CriarAnuncio(ctx context.Context, anuncio anuncios.Anuncio) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO anuncio (
			id, id_perfil_vendedor, titulo, descricao, categoria, tamanho, cor,
			estado_conservacao, preco_centavos, status, criado_em, atualizado_em
		) SELECT $1, pv.id, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		  FROM perfil_vendedor pv
		 WHERE pv.id_usuario = $2
	`, anuncio.ID, anuncio.IDVendedor, anuncio.Titulo, anuncio.Descricao,
		anuncio.Categoria, anuncio.Tamanho, anuncio.Cor, anuncio.EstadoConservacao,
		anuncio.PrecoCentavos, anuncio.Status, anuncio.CriadoEm, anuncio.AtualizadoEm)
	if err != nil {
		return mapDatabaseError(err)
	}
	for _, foto := range anuncio.Fotos {
		_, err = tx.Exec(ctx, `
			INSERT INTO foto_anuncio (id, id_anuncio, url, ordem, legenda, criado_em)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
		`, foto.ID, anuncio.ID, foto.URL, foto.Ordem, foto.Legenda, anuncio.CriadoEm)
		if err != nil {
			return mapDatabaseError(err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) BuscarAnuncioPorID(ctx context.Context, id string) (anuncios.Anuncio, error) {
	var anuncio anuncios.Anuncio
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, pv.id_usuario, a.titulo, a.descricao, a.categoria, a.tamanho, a.cor,
		       a.estado_conservacao, a.preco_centavos, a.status, a.criado_em,
		       a.atualizado_em, a.excluido_em
		FROM anuncio a
		JOIN perfil_vendedor pv ON pv.id = a.id_perfil_vendedor
		WHERE a.id = $1 AND a.excluido_em IS NULL
	`, id).Scan(
		&anuncio.ID, &anuncio.IDVendedor, &anuncio.Titulo, &anuncio.Descricao,
		&anuncio.Categoria, &anuncio.Tamanho, &anuncio.Cor, &anuncio.EstadoConservacao,
		&anuncio.PrecoCentavos, &anuncio.Status, &anuncio.CriadoEm,
		&anuncio.AtualizadoEm, &anuncio.ExcluidoEm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return anuncios.Anuncio{}, common.ErrNaoEncontrado
	}
	if err != nil {
		return anuncios.Anuncio{}, err
	}
	anuncio.Fotos, err = s.buscarFotosAnuncio(ctx, anuncio.ID)
	return anuncio, err
}

func (s *Store) ListarAnuncios(
	ctx context.Context,
	filtro casosdeuso.FiltroAnuncios,
) ([]anuncios.Anuncio, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, pv.id_usuario, a.titulo, a.descricao, a.categoria, a.tamanho, a.cor,
		       a.estado_conservacao, a.preco_centavos, a.status, a.criado_em, a.atualizado_em
		FROM anuncio a
		JOIN perfil_vendedor pv ON pv.id = a.id_perfil_vendedor
		WHERE ($7 OR a.status = 'disponivel')
		  AND ($7 OR a.excluido_em IS NULL)
		  AND ($1 = '' OR a.titulo ILIKE '%' || $1 || '%' OR a.descricao ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR a.categoria = LOWER($2))
		  AND ($3 = '' OR a.tamanho = UPPER($3))
		  AND ($4 = '' OR a.estado_conservacao = $4)
		  AND ($5 = 0 OR a.preco_centavos >= $5)
		  AND ($6 = 0 OR a.preco_centavos <= $6)
		  AND ($8 = '' OR pv.id_usuario::text = $8)
		  AND ($9 = '' OR pv.id_usuario::text <> $9)
		ORDER BY a.criado_em DESC
		LIMIT $10 OFFSET $11
	`, strings.TrimSpace(filtro.Palavra), filtro.Categoria, filtro.Tamanho,
		filtro.EstadoConservacao, filtro.PrecoMinCentavos, filtro.PrecoMaxCentavos,
		filtro.IncluirTodosStatus, filtro.IDVendedor, filtro.ExcluirVendedor,
		filtro.Limite, filtro.Deslocamento)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultado []anuncios.Anuncio
	for rows.Next() {
		var anuncio anuncios.Anuncio
		if err := rows.Scan(
			&anuncio.ID, &anuncio.IDVendedor, &anuncio.Titulo, &anuncio.Descricao,
			&anuncio.Categoria, &anuncio.Tamanho, &anuncio.Cor, &anuncio.EstadoConservacao,
			&anuncio.PrecoCentavos, &anuncio.Status, &anuncio.CriadoEm, &anuncio.AtualizadoEm,
		); err != nil {
			return nil, err
		}
		anuncio.Fotos, err = s.buscarFotosAnuncio(ctx, anuncio.ID)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, anuncio)
	}
	return resultado, rows.Err()
}

func (s *Store) buscarFotosAnuncio(ctx context.Context, idAnuncio string) ([]anuncios.Foto, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, url, ordem, COALESCE(legenda, '')
		FROM foto_anuncio
		WHERE id_anuncio = $1
		ORDER BY ordem
	`, idAnuncio)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fotos []anuncios.Foto
	for rows.Next() {
		var foto anuncios.Foto
		if err := rows.Scan(&foto.ID, &foto.URL, &foto.Ordem, &foto.Legenda); err != nil {
			return nil, err
		}
		fotos = append(fotos, foto)
	}
	return fotos, rows.Err()
}
