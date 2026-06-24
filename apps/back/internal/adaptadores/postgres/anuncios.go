package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"reveste/apps/back/internal/casosdeuso"
	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/anuncios"
)

func (s *Store) CriarAnuncio(ctx context.Context, anuncio anuncios.Anuncio) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	resultado, err := tx.Exec(ctx, `
		INSERT INTO anuncio (
			id, id_perfil_vendedor, titulo, descricao, categoria, tamanho, cor,
			estado_conservacao, preco_centavos, status, criado_em, atualizado_em,
			peso_g, altura_cm, largura_cm, comprimento_cm
		) SELECT $1, pv.id, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		  FROM perfil_vendedor pv
		 WHERE pv.id_usuario = $2
	`, anuncio.ID, anuncio.IDVendedor, anuncio.Titulo, anuncio.Descricao,
		anuncio.Categoria, anuncio.Tamanho, anuncio.Cor, anuncio.EstadoConservacao,
		anuncio.PrecoCentavos, anuncio.Status, anuncio.CriadoEm, anuncio.AtualizadoEm,
		anuncio.PesoGramas, anuncio.AlturaCm, anuncio.LarguraCm, anuncio.ComprimentoCm)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoEncontrado
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

func (s *Store) AtualizarAnuncio(ctx context.Context, anuncio anuncios.Anuncio) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	resultado, err := tx.Exec(ctx, `
		UPDATE anuncio a
		SET titulo = $3, descricao = $4, categoria = $5, tamanho = $6, cor = $7,
		    estado_conservacao = $8, preco_centavos = $9, atualizado_em = $10,
		    peso_g = $11, altura_cm = $12, largura_cm = $13, comprimento_cm = $14
		FROM perfil_vendedor pv
		WHERE a.id = $1 AND a.id_perfil_vendedor = pv.id AND pv.id_usuario = $2
		  AND a.status = 'disponivel' AND a.excluido_em IS NULL
	`, anuncio.ID, anuncio.IDVendedor, anuncio.Titulo, anuncio.Descricao,
		anuncio.Categoria, anuncio.Tamanho, anuncio.Cor, anuncio.EstadoConservacao,
		anuncio.PrecoCentavos, anuncio.AtualizadoEm,
		anuncio.PesoGramas, anuncio.AlturaCm, anuncio.LarguraCm, anuncio.ComprimentoCm)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoPermitido
	}
	if _, err := tx.Exec(ctx, `DELETE FROM foto_anuncio WHERE id_anuncio = $1`, anuncio.ID); err != nil {
		return err
	}
	for _, foto := range anuncio.Fotos {
		if _, err := tx.Exec(ctx, `
			INSERT INTO foto_anuncio (id, id_anuncio, url, ordem, legenda, criado_em)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
		`, foto.ID, anuncio.ID, foto.URL, foto.Ordem, foto.Legenda, anuncio.AtualizadoEm); err != nil {
			return mapDatabaseError(err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ExcluirAnuncio(
	ctx context.Context,
	idAnuncio,
	idVendedor string,
	agora time.Time,
) error {
	resultado, err := s.pool.Exec(ctx, `
		UPDATE anuncio a
		SET status = 'excluido', excluido_em = $3, atualizado_em = $3
		FROM perfil_vendedor pv
		WHERE a.id = $1 AND a.id_perfil_vendedor = pv.id AND pv.id_usuario = $2
		  AND a.status = 'disponivel' AND a.excluido_em IS NULL
	`, idAnuncio, idVendedor, agora)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoPermitido
	}
	return nil
}

func (s *Store) BuscarAnuncioPorID(ctx context.Context, id string) (anuncios.Anuncio, error) {
	var anuncio anuncios.Anuncio
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, pv.id_usuario, a.titulo, a.descricao, a.categoria, a.tamanho, a.cor,
		       a.estado_conservacao, a.preco_centavos, a.status, a.criado_em,
		       a.atualizado_em, a.excluido_em,
		       a.peso_g, a.altura_cm, a.largura_cm, a.comprimento_cm
		FROM anuncio a
		JOIN perfil_vendedor pv ON pv.id = a.id_perfil_vendedor
		WHERE a.id = $1 AND a.excluido_em IS NULL
	`, id).Scan(
		&anuncio.ID, &anuncio.IDVendedor, &anuncio.Titulo, &anuncio.Descricao,
		&anuncio.Categoria, &anuncio.Tamanho, &anuncio.Cor, &anuncio.EstadoConservacao,
		&anuncio.PrecoCentavos, &anuncio.Status, &anuncio.CriadoEm,
		&anuncio.AtualizadoEm, &anuncio.ExcluidoEm,
		&anuncio.PesoGramas, &anuncio.AlturaCm, &anuncio.LarguraCm, &anuncio.ComprimentoCm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return anuncios.Anuncio{}, common.ErrNaoEncontrado
	}
	if err != nil {
		return anuncios.Anuncio{}, mapDatabaseError(err)
	}
	anuncio.Fotos, err = s.buscarFotosAnuncio(ctx, anuncio.ID)
	return anuncio, err
}

func (s *Store) ListarAnuncios(
	ctx context.Context,
	filtro casosdeuso.FiltroAnuncios,
) ([]anuncios.Anuncio, error) {
	idsAnuncios := filtro.IDsAnuncios
	if idsAnuncios == nil {
		idsAnuncios = []string{}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, pv.id_usuario, a.titulo, a.descricao, a.categoria, a.tamanho, a.cor,
		       a.estado_conservacao, a.preco_centavos, a.status, a.criado_em, a.atualizado_em,
		       a.peso_g, a.altura_cm, a.largura_cm, a.comprimento_cm
		FROM anuncio a
		JOIN perfil_vendedor pv ON pv.id = a.id_perfil_vendedor
		WHERE ($7 OR a.status = 'disponivel')
		  AND a.excluido_em IS NULL
		  AND ($1 = '' OR a.titulo ILIKE '%' || $1 || '%' OR a.descricao ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR a.categoria = LOWER($2))
		  AND ($3 = '' OR a.tamanho = UPPER($3))
		  AND ($4 = '' OR a.estado_conservacao = $4)
		  AND ($5 = 0 OR a.preco_centavos >= $5)
		  AND ($6 = 0 OR a.preco_centavos <= $6)
		  AND ($8 = '' OR pv.id_usuario::text = $8)
		  AND ($9 = '' OR pv.id_usuario::text <> $9)
		  AND (cardinality($12::text[]) = 0 OR a.id::text = ANY($12::text[]))
		ORDER BY a.criado_em DESC
		LIMIT $10 OFFSET $11
	`, strings.TrimSpace(filtro.Palavra), filtro.Categoria, filtro.Tamanho,
		filtro.EstadoConservacao, filtro.PrecoMinCentavos, filtro.PrecoMaxCentavos,
		filtro.IncluirTodosStatus, filtro.IDVendedor, filtro.ExcluirVendedor,
		filtro.Limite, filtro.Deslocamento, idsAnuncios)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultado []anuncios.Anuncio
	ids := make([]string, 0, filtro.Limite)
	for rows.Next() {
		var anuncio anuncios.Anuncio
		if err := rows.Scan(
			&anuncio.ID, &anuncio.IDVendedor, &anuncio.Titulo, &anuncio.Descricao,
			&anuncio.Categoria, &anuncio.Tamanho, &anuncio.Cor, &anuncio.EstadoConservacao,
			&anuncio.PrecoCentavos, &anuncio.Status, &anuncio.CriadoEm, &anuncio.AtualizadoEm,
			&anuncio.PesoGramas, &anuncio.AlturaCm, &anuncio.LarguraCm, &anuncio.ComprimentoCm,
		); err != nil {
			return nil, err
		}
		resultado = append(resultado, anuncio)
		ids = append(ids, anuncio.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	fotosPorAnuncio, err := s.buscarFotosAnuncios(ctx, ids)
	if err != nil {
		return nil, err
	}
	for indice := range resultado {
		resultado[indice].Fotos = fotosPorAnuncio[resultado[indice].ID]
	}
	return resultado, nil
}

func (s *Store) buscarFotosAnuncio(ctx context.Context, idAnuncio string) ([]anuncios.Foto, error) {
	fotosPorAnuncio, err := s.buscarFotosAnuncios(ctx, []string{idAnuncio})
	return fotosPorAnuncio[idAnuncio], err
}

func (s *Store) buscarFotosAnuncios(
	ctx context.Context,
	idsAnuncios []string,
) (map[string][]anuncios.Foto, error) {
	resultado := make(map[string][]anuncios.Foto, len(idsAnuncios))
	if len(idsAnuncios) == 0 {
		return resultado, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id_anuncio::text, id, url, ordem, COALESCE(legenda, '')
		FROM foto_anuncio
		WHERE id_anuncio::text = ANY($1::text[])
		ORDER BY id_anuncio, ordem
	`, idsAnuncios)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idAnuncio string
		var foto anuncios.Foto
		if err := rows.Scan(&idAnuncio, &foto.ID, &foto.URL, &foto.Ordem, &foto.Legenda); err != nil {
			return nil, err
		}
		resultado[idAnuncio] = append(resultado[idAnuncio], foto)
	}
	return resultado, rows.Err()
}
