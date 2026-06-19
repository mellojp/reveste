package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/interacao"
)

func (s *Store) BuscarParticipantesPedido(ctx context.Context, idPedido string) (string, string, error) {
	var idComprador, idVendedor string
	err := s.pool.QueryRow(ctx, `
		SELECT id_comprador::text, id_vendedor::text FROM pedido WHERE id = $1
	`, idPedido).Scan(&idComprador, &idVendedor)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", common.ErrNaoEncontrado
	}
	if err != nil {
		return "", "", mapDatabaseError(err)
	}
	return idComprador, idVendedor, nil
}

// ObterOuCriarConversa garante uma unica conversa por pedido (uq_conversa id_pedido). O
// DO UPDATE e um no-op que permite RETURNING tanto na criacao quanto quando ja existe.
func (s *Store) ObterOuCriarConversa(ctx context.Context, novoID, idPedido string, agora time.Time) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO conversa (id, id_pedido, criado_em)
		VALUES ($1, $2, $3)
		ON CONFLICT (id_pedido) DO UPDATE SET id_pedido = EXCLUDED.id_pedido
		RETURNING id::text
	`, novoID, idPedido, agora).Scan(&id)
	if err != nil {
		return "", mapDatabaseError(err)
	}
	return id, nil
}

func (s *Store) ListarMensagens(ctx context.Context, idConversa string) ([]interacao.Mensagem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, id_conversa::text, id_usuario_remetente::text, conteudo, lida_em, criada_em
		FROM mensagem
		WHERE id_conversa = $1
		ORDER BY criada_em ASC
	`, idConversa)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()

	mensagens := make([]interacao.Mensagem, 0)
	for rows.Next() {
		var m interacao.Mensagem
		if err := rows.Scan(
			&m.ID, &m.IDConversa, &m.IDUsuarioRemetente, &m.Conteudo, &m.LidaEm, &m.CriadaEm,
		); err != nil {
			return nil, err
		}
		mensagens = append(mensagens, m)
	}
	return mensagens, rows.Err()
}

func (s *Store) CriarMensagem(ctx context.Context, m interacao.Mensagem) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO mensagem (id, id_conversa, id_usuario_remetente, conteudo, criada_em)
		VALUES ($1, $2, $3, $4, $5)
	`, m.ID, m.IDConversa, m.IDUsuarioRemetente, m.Conteudo, m.CriadaEm)
	return mapDatabaseError(err)
}
