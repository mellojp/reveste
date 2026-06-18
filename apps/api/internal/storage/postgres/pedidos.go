package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/compras"
	"reveste/apps/api/internal/dominio/interacao"
)

func (s *Store) ListarPedidosDoVendedor(ctx context.Context, idVendedor string) ([]compras.Pedido, error) {
	return s.buscarPedidos(ctx, `p.id_vendedor = $1`, idVendedor)
}

func (s *Store) BuscarPedidoDoComprador(ctx context.Context, idComprador, idPedido string) (compras.Pedido, error) {
	pedidos, err := s.buscarPedidos(ctx, `p.id_comprador = $1 AND p.id = $2`, idComprador, idPedido)
	if err != nil {
		return compras.Pedido{}, err
	}
	if len(pedidos) == 0 {
		return compras.Pedido{}, common.ErrNaoEncontrado
	}
	return pedidos[0], nil
}

func (s *Store) BuscarPedidoDoVendedor(ctx context.Context, idVendedor, idPedido string) (compras.Pedido, error) {
	pedidos, err := s.buscarPedidos(ctx, `p.id_vendedor = $1 AND p.id = $2`, idVendedor, idPedido)
	if err != nil {
		return compras.Pedido{}, err
	}
	if len(pedidos) == 0 {
		return compras.Pedido{}, common.ErrNaoEncontrado
	}
	return pedidos[0], nil
}

func (s *Store) BuscarAvaliacaoDoPedido(ctx context.Context, idPedido string) (interacao.Avaliacao, error) {
	var avaliacao interacao.Avaliacao
	err := s.pool.QueryRow(ctx, `
		SELECT id, id_pedido, id_usuario_autor, id_usuario_avaliado, nota, COALESCE(comentario, ''), criada_em
		FROM avaliacao
		WHERE id_pedido = $1
	`, idPedido).Scan(
		&avaliacao.ID, &avaliacao.IDPedido, &avaliacao.IDUsuarioAutor, &avaliacao.IDUsuarioAvaliado,
		&avaliacao.Nota, &avaliacao.Comentario, &avaliacao.CriadaEm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return interacao.Avaliacao{}, common.ErrNaoEncontrado
	}
	if err != nil {
		return interacao.Avaliacao{}, mapDatabaseError(err)
	}
	return avaliacao, nil
}

// MarcarPedidoEnviado executa, em uma transacao, a postagem de um pedido: itens vao para
// enviado, a entrega para postado e o pedido para aguardando_entrega. So avanca se o pedido
// pertence ao vendedor e ainda aguarda envio (idempotente via WHERE de status).
func (s *Store) MarcarPedidoEnviado(
	ctx context.Context,
	idPedido, idVendedor, provedor, rastreio string,
	agora time.Time,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	resultado, err := tx.Exec(ctx, `
		UPDATE pedido SET status = 'aguardando_entrega', atualizado_em = $3
		WHERE id = $1 AND id_vendedor = $2 AND status = 'aguardando_envio'
	`, idPedido, idVendedor, agora)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoPermitido
	}
	if _, err := tx.Exec(ctx, `
		UPDATE item_pedido SET status = 'enviado', enviado_em = $2
		WHERE id_pedido = $1 AND status = 'aguardando_envio'
	`, idPedido, agora); err != nil {
		return mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE entrega
		SET status = 'postado', provedor = $2, codigo_rastreio = $3,
		    postado_em = $4, atualizado_em = $4
		WHERE id_pedido = $1 AND status = 'aguardando_postagem'
	`, idPedido, provedor, rastreio, agora); err != nil {
		return mapDatabaseError(err)
	}
	return tx.Commit(ctx)
}

// ConfirmarRecebimentoPedido finaliza o pedido a pedido do comprador: itens para recebido,
// entrega para entregue e pedido para finalizado.
func (s *Store) ConfirmarRecebimentoPedido(
	ctx context.Context,
	idPedido, idComprador string,
	agora time.Time,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	resultado, err := tx.Exec(ctx, `
		UPDATE pedido SET status = 'finalizado', finalizado_em = $3, atualizado_em = $3
		WHERE id = $1 AND id_comprador = $2 AND status = 'aguardando_entrega'
	`, idPedido, idComprador, agora)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoPermitido
	}
	if _, err := tx.Exec(ctx, `
		UPDATE item_pedido SET status = 'recebido', recebido_em = $2
		WHERE id_pedido = $1 AND status = 'enviado'
	`, idPedido, agora); err != nil {
		return mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE entrega SET status = 'entregue', entregue_em = $2, atualizado_em = $2
		WHERE id_pedido = $1
	`, idPedido, agora); err != nil {
		return mapDatabaseError(err)
	}
	return tx.Commit(ctx)
}

func (s *Store) RegistrarAvaliacao(ctx context.Context, avaliacao interacao.Avaliacao) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO avaliacao (
			id, id_pedido, id_usuario_autor, id_usuario_avaliado, nota, comentario, criada_em
		) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)
	`, avaliacao.ID, avaliacao.IDPedido, avaliacao.IDUsuarioAutor, avaliacao.IDUsuarioAvaliado,
		avaliacao.Nota, avaliacao.Comentario, avaliacao.CriadaEm)
	return mapDatabaseError(err)
}

func (s *Store) MediaAvaliacoesVendedor(ctx context.Context, idVendedor string) (casosdeuso.MediaAvaliacoes, error) {
	var media casosdeuso.MediaAvaliacoes
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(nota), 0)::float8, COUNT(*)
		FROM avaliacao
		WHERE id_usuario_avaliado = $1
	`, idVendedor).Scan(&media.Media, &media.Quantidade)
	if err != nil {
		return casosdeuso.MediaAvaliacoes{}, mapDatabaseError(err)
	}
	return media, nil
}

// ProcessarItensVencidos encerra de forma consistente os pedidos que perderam o prazo:
// itens ficam como nao enviados, pedido e entrega sao cancelados, o reembolso simulado e
// registrado e o vendedor recebe a penalidade. Tudo ocorre na mesma transacao.
func (s *Store) ProcessarItensVencidos(
	ctx context.Context,
	agora time.Time,
	limiteBloqueio int,
) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT ip.id::text, p.id_vendedor::text
		FROM item_pedido ip
		JOIN pedido p ON p.id = ip.id_pedido
		WHERE ip.status = 'aguardando_envio' AND ip.prazo_envio_em < $1
		FOR UPDATE OF ip
	`, agora)
	if err != nil {
		return 0, mapDatabaseError(err)
	}
	var idsItens []string
	contagemPorVendedor := make(map[string]int)
	for rows.Next() {
		var idItem, idVendedor string
		if err := rows.Scan(&idItem, &idVendedor); err != nil {
			rows.Close()
			return 0, err
		}
		idsItens = append(idsItens, idItem)
		contagemPorVendedor[idVendedor]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(idsItens) == 0 {
		return 0, tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE item_pedido SET status = 'nao_enviado'
		WHERE id::text = ANY($1::text[]) AND status = 'aguardando_envio'
	`, idsItens); err != nil {
		return 0, mapDatabaseError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE pedido p
		SET status = 'cancelado', atualizado_em = $2
		WHERE p.status = 'aguardando_envio'
		  AND EXISTS (
			SELECT 1 FROM item_pedido ip
			WHERE ip.id_pedido = p.id AND ip.id::text = ANY($1::text[])
		  )
	`, idsItens, agora); err != nil {
		return 0, mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE entrega e
		SET status = 'falhou', atualizado_em = $2
		WHERE EXISTS (
			SELECT 1
			FROM pedido p
			JOIN item_pedido ip ON ip.id_pedido = p.id
			WHERE p.id = e.id_pedido AND ip.id::text = ANY($1::text[])
		)
	`, idsItens, agora); err != nil {
		return 0, mapDatabaseError(err)
	}

	// O adaptador financeiro do MVP e simulado. Cada item gera um reembolso processado;
	// o frete do pedido e somado ao primeiro item para que o valor devolvido seja exato.
	if _, err := tx.Exec(ctx, `
		WITH itens_reembolso AS (
			SELECT ip.id, ip.id_pedido, ip.valor_unitario_centavos,
			       p.id_compra, p.valor_frete_centavos,
			       ROW_NUMBER() OVER (PARTITION BY ip.id_pedido ORDER BY ip.id) AS ordem
			FROM item_pedido ip
			JOIN pedido p ON p.id = ip.id_pedido
			WHERE ip.id::text = ANY($1::text[])
		)
		INSERT INTO reembolso (
			id_pagamento, id_item_pedido, identificador_externo, status,
			valor_centavos, motivo, chave_idempotencia, processado_em,
			criado_em, atualizado_em
		)
		SELECT pg.id, ir.id, 'nao-envio-' || ir.id::text, 'processado',
		       ir.valor_unitario_centavos +
		         CASE WHEN ir.ordem = 1 THEN ir.valor_frete_centavos ELSE 0 END,
		       'item_nao_enviado', 'nao-envio-' || ir.id::text, $2, $2, $2
		FROM itens_reembolso ir
		JOIN pagamento pg ON pg.id_compra = ir.id_compra
		ON CONFLICT (chave_idempotencia) DO NOTHING
	`, idsItens, agora); err != nil {
		return 0, mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pagamento pg
		SET status = CASE
			WHEN EXISTS (
				SELECT 1 FROM pedido p
				WHERE p.id_compra = pg.id_compra AND p.status <> 'cancelado'
			) THEN 'reembolsado_parcial'
			ELSE 'reembolsado'
		END,
		atualizado_em = $2
		WHERE pg.status IN ('aprovado', 'reembolsado_parcial')
		  AND EXISTS (
			SELECT 1
			FROM pedido p
			JOIN item_pedido ip ON ip.id_pedido = p.id
			WHERE p.id_compra = pg.id_compra AND ip.id::text = ANY($1::text[])
		  )
	`, idsItens, agora); err != nil {
		return 0, mapDatabaseError(err)
	}

	for idVendedor, quantidade := range contagemPorVendedor {
		if _, err := tx.Exec(ctx, `
			UPDATE perfil_vendedor
			SET itens_nao_enviados = itens_nao_enviados + $2,
			    bloqueado = (itens_nao_enviados + $2 >= $3),
			    atualizado_em = $4
			WHERE id_usuario = $1
		`, idVendedor, quantidade, limiteBloqueio, agora); err != nil {
			return 0, mapDatabaseError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(idsItens), nil
}
