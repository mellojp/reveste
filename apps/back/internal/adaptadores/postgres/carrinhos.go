package postgres

import (
	"context"
	"time"

	"reveste/apps/back/internal/dominio/compras"
)

func (s *Store) ObterOuCriarCarrinho(
	ctx context.Context,
	novoID string,
	idUsuario string,
	agora time.Time,
) (compras.Carrinho, error) {
	if err := s.criarCarrinhoSeNecessario(ctx, novoID, idUsuario, agora); err != nil {
		return compras.Carrinho{}, err
	}
	return s.buscarCarrinhoPorUsuario(ctx, idUsuario)
}

func (s *Store) AdicionarAnuncioAoCarrinho(
	ctx context.Context,
	novoID,
	idUsuario,
	idAnuncio string,
	agora time.Time,
) (compras.Carrinho, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return compras.Carrinho{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO carrinho (id, id_usuario, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (id_usuario) DO NOTHING
	`, novoID, idUsuario, agora)
	if err != nil {
		return compras.Carrinho{}, mapDatabaseError(err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO carrinho_anuncio (id_carrinho, id_anuncio, adicionado_em)
		SELECT id, $2, $3 FROM carrinho WHERE id_usuario = $1
		ON CONFLICT (id_carrinho, id_anuncio) DO NOTHING
	`, idUsuario, idAnuncio, agora)
	if err != nil {
		return compras.Carrinho{}, mapDatabaseError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE carrinho SET atualizado_em = $2 WHERE id_usuario = $1
	`, idUsuario, agora); err != nil {
		return compras.Carrinho{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return compras.Carrinho{}, err
	}
	return s.buscarCarrinhoPorUsuario(ctx, idUsuario)
}

func (s *Store) RemoverAnuncioDoCarrinho(
	ctx context.Context,
	novoID,
	idUsuario,
	idAnuncio string,
	agora time.Time,
) (compras.Carrinho, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return compras.Carrinho{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO carrinho (id, id_usuario, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (id_usuario) DO NOTHING
	`, novoID, idUsuario, agora)
	if err != nil {
		return compras.Carrinho{}, mapDatabaseError(err)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM carrinho_anuncio
		WHERE id_carrinho = (SELECT id FROM carrinho WHERE id_usuario = $1)
		  AND id_anuncio = $2
	`, idUsuario, idAnuncio); err != nil {
		return compras.Carrinho{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE carrinho SET atualizado_em = $2 WHERE id_usuario = $1
	`, idUsuario, agora); err != nil {
		return compras.Carrinho{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return compras.Carrinho{}, err
	}
	return s.buscarCarrinhoPorUsuario(ctx, idUsuario)
}

func (s *Store) criarCarrinhoSeNecessario(
	ctx context.Context,
	novoID,
	idUsuario string,
	agora time.Time,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO carrinho (id, id_usuario, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (id_usuario) DO NOTHING
	`, novoID, idUsuario, agora)
	return mapDatabaseError(err)
}

func (s *Store) buscarCarrinhoPorUsuario(ctx context.Context, idUsuario string) (compras.Carrinho, error) {
	carrinho := compras.Carrinho{IDsAnuncios: []string{}}
	err := s.pool.QueryRow(ctx, `
		SELECT id, id_usuario, criado_em, atualizado_em
		FROM carrinho
		WHERE id_usuario = $1
	`, idUsuario).Scan(&carrinho.ID, &carrinho.IDUsuario, &carrinho.CriadoEm, &carrinho.AtualizadoEm)
	if err != nil {
		return compras.Carrinho{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id_anuncio
		FROM carrinho_anuncio
		WHERE id_carrinho = $1
		ORDER BY adicionado_em, id_anuncio
	`, carrinho.ID)
	if err != nil {
		return compras.Carrinho{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var idAnuncio string
		if err := rows.Scan(&idAnuncio); err != nil {
			return compras.Carrinho{}, err
		}
		carrinho.IDsAnuncios = append(carrinho.IDsAnuncios, idAnuncio)
	}
	return carrinho, rows.Err()
}
