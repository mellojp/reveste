package postgres

import (
	"context"
	"time"

	"reveste/apps/api/internal/dominio/interacao"
)

func (s *Store) CriarNotificacao(ctx context.Context, n interacao.Notificacao) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notificacao (id, id_usuario, tipo, conteudo, id_pedido, criada_em)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, $6)
	`, n.ID, n.IDUsuario, n.Tipo, n.Conteudo, n.IDPedido, n.CriadaEm)
	return mapDatabaseError(err)
}

func (s *Store) ListarNotificacoes(ctx context.Context, idUsuario string, limite int) ([]interacao.Notificacao, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, id_usuario::text, tipo, conteudo,
		       COALESCE(id_pedido::text, ''), lida_em, criada_em
		FROM notificacao
		WHERE id_usuario = $1
		ORDER BY criada_em DESC
		LIMIT $2
	`, idUsuario, limite)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()

	notificacoes := make([]interacao.Notificacao, 0)
	for rows.Next() {
		var n interacao.Notificacao
		if err := rows.Scan(
			&n.ID, &n.IDUsuario, &n.Tipo, &n.Conteudo, &n.IDPedido, &n.LidaEm, &n.CriadaEm,
		); err != nil {
			return nil, err
		}
		notificacoes = append(notificacoes, n)
	}
	return notificacoes, rows.Err()
}

func (s *Store) ContarNotificacoesNaoLidas(ctx context.Context, idUsuario string) (int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notificacao WHERE id_usuario = $1 AND lida_em IS NULL
	`, idUsuario).Scan(&total)
	if err != nil {
		return 0, mapDatabaseError(err)
	}
	return total, nil
}

func (s *Store) MarcarNotificacoesLidas(ctx context.Context, idUsuario string, agora time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notificacao SET lida_em = $2
		WHERE id_usuario = $1 AND lida_em IS NULL
	`, idUsuario, agora)
	return mapDatabaseError(err)
}

func (s *Store) RemoverNotificacao(ctx context.Context, idUsuario, idNotificacao string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM notificacao WHERE id = $1 AND id_usuario = $2
	`, idNotificacao, idUsuario)
	return mapDatabaseError(err)
}

func (s *Store) LimparNotificacoes(ctx context.Context, idUsuario string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM notificacao WHERE id_usuario = $1`, idUsuario)
	return mapDatabaseError(err)
}
