package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"reveste/apps/api/internal/common"
)

func (s *Store) CriarSessao(ctx context.Context, token, idUsuario string, expiraEm time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessao (token_hash, id_usuario, expira_em)
		VALUES ($1, $2, $3)
	`, hashToken(token), idUsuario, expiraEm)
	return mapDatabaseError(err)
}

func (s *Store) BuscarUsuarioDaSessao(ctx context.Context, token string, agora time.Time) (string, error) {
	var idUsuario string
	err := s.pool.QueryRow(ctx, `
		SELECT s.id_usuario
		FROM sessao s
		JOIN usuario u ON u.id = s.id_usuario AND u.excluido_em IS NULL
		WHERE s.token_hash = $1 AND s.expira_em > $2
	`, hashToken(token), agora).Scan(&idUsuario)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", common.ErrNaoAutorizado
	}
	if err != nil {
		return "", mapDatabaseError(err)
	}
	return idUsuario, nil
}

func (s *Store) RemoverSessao(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessao WHERE token_hash = $1`, hashToken(token))
	return err
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
