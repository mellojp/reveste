package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/compras"
)

// IniciarCompra cria a intencao e reserva os anuncios antes da chamada ao provedor.
// A insercao da chave de idempotencia e a reserva condicional ficam na mesma transacao.
func (s *Store) IniciarCompra(
	ctx context.Context,
	compra compras.Compra,
	pagamento compras.Pagamento,
	idComprador string,
) (compras.Compra, bool, error) {
	idsAnuncios := compra.IDsAnuncios()
	if len(idsAnuncios) == 0 {
		return compras.Compra{}, false, common.ErrSemItensDisponiveis
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return compras.Compra{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	insercao, err := tx.Exec(ctx, `
		INSERT INTO compra (
			id, id_comprador, status, valor_itens_centavos, valor_fretes_centavos,
			valor_taxa_servico_centavos, valor_total_centavos, chave_idempotencia,
			expira_em, criado_em, atualizado_em
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		ON CONFLICT (chave_idempotencia) DO NOTHING
	`, compra.ID, idComprador, compra.Status, compra.ValorTotalItensCentavos,
		compra.ValorTotalFretesCentavos, compra.ValorTaxaServicoCentavos,
		compra.ValorFinalPagoCentavos, compra.ChaveIdempotencia, compra.ExpiraEm,
		compra.CriadaEm)
	if err != nil {
		return compras.Compra{}, false, mapDatabaseError(err)
	}
	if insercao.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return compras.Compra{}, false, err
		}
		existente, err := s.BuscarCompraPorChave(ctx, compra.ChaveIdempotencia)
		return existente, false, err
	}

	reserva, err := tx.Exec(ctx, `
		UPDATE anuncio
		SET status = 'reservado', atualizado_em = $2
		WHERE id::text = ANY($1::text[]) AND status = 'disponivel' AND excluido_em IS NULL
	`, idsAnuncios, compra.CriadaEm)
	if err != nil {
		return compras.Compra{}, false, mapDatabaseError(err)
	}
	if reserva.RowsAffected() != int64(len(idsAnuncios)) {
		return compras.Compra{}, false, common.ErrAnuncioIndisponivel
	}

	for _, pedido := range compra.Pedidos {
		endereco := pedido.EnderecoEntrega
		if _, err := tx.Exec(ctx, `
			INSERT INTO pedido (
				id, id_compra, id_comprador, id_vendedor, status,
				valor_itens_centavos, valor_frete_centavos, valor_taxa_servico_centavos,
				valor_liquido_vendedor_centavos, nome_destinatario, cep_destino,
				logradouro_destino, numero_destino, complemento_destino, bairro_destino,
				cidade_destino, estado_destino, criado_em, atualizado_em
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
				NULLIF($14, ''), $15, $16, $17, $18, $18
			)
		`, pedido.ID, compra.ID, idComprador, pedido.IDVendedor, pedido.Status,
			pedido.ValorTotalItensCentavos, pedido.ValorFreteCentavos,
			pedido.TaxaServicoCentavos, pedido.ValorLiquidoVendedorCentavos,
			pedido.NomeDestinatario, endereco.CEP, endereco.Logradouro, endereco.Numero,
			endereco.Complemento, endereco.Bairro, endereco.Cidade, endereco.Estado,
			compra.CriadaEm); err != nil {
			return compras.Compra{}, false, mapDatabaseError(err)
		}

		for _, item := range pedido.Itens {
			if _, err := tx.Exec(ctx, `
				INSERT INTO item_pedido (
					id, id_pedido, id_anuncio, status, titulo, categoria, tamanho, cor,
					estado_conservacao, valor_unitario_centavos, taxa_servico_centavos,
					prazo_envio_em, criado_em
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			`, item.ID, pedido.ID, item.IDAnuncio, item.Status, item.Titulo, item.Categoria,
				item.Tamanho, item.Cor, item.EstadoConservacao, item.ValorUnitarioCentavos,
				item.TaxaServicoCentavos, item.PrazoEnvioEm, compra.CriadaEm); err != nil {
				return compras.Compra{}, false, mapDatabaseError(err)
			}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO entrega (id_pedido, status, valor_frete_centavos, criado_em, atualizado_em)
			VALUES ($1, 'aguardando_postagem', $2, $3, $3)
		`, pedido.ID, pedido.ValorFreteCentavos, compra.CriadaEm); err != nil {
			return compras.Compra{}, false, mapDatabaseError(err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO pagamento (
			id, id_compra, provedor, identificador_externo, status, valor_centavos,
			chave_idempotencia, pago_em, criado_em, atualizado_em
		) VALUES ($1, $2, $3, NULL, $4, $5, $6, NULL, $7, $7)
	`, pagamento.ID, compra.ID, pagamento.Provedor, pagamento.Status,
		pagamento.ValorCentavos, pagamento.ChaveIdempotencia, compra.CriadaEm); err != nil {
		return compras.Compra{}, false, mapDatabaseError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return compras.Compra{}, false, err
	}
	return compra, true, nil
}

// ConfirmarCompraAprovada conclui uma intencao pendente de forma idempotente.
func (s *Store) ConfirmarCompraAprovada(
	ctx context.Context,
	chave, provedor, identificadorExterno string,
	agora time.Time,
) (compras.Compra, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return compras.Compra{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var idCompra, idComprador string
	var status compras.StatusCompra
	if err := tx.QueryRow(ctx, `
		SELECT id, id_comprador, status
		FROM compra
		WHERE chave_idempotencia = $1
		FOR UPDATE
	`, chave).Scan(&idCompra, &idComprador, &status); errors.Is(err, pgx.ErrNoRows) {
		return compras.Compra{}, common.ErrNaoEncontrado
	} else if err != nil {
		return compras.Compra{}, mapDatabaseError(err)
	}
	if status == compras.StatusCompraAprovada {
		if err := tx.Commit(ctx); err != nil {
			return compras.Compra{}, err
		}
		return s.BuscarCompraPorChave(ctx, chave)
	}
	if status != compras.StatusCompraAguardandoPagamento {
		return compras.Compra{}, common.ErrTransicaoInvalida
	}

	resultado, err := tx.Exec(ctx, `
		UPDATE anuncio
		SET status = 'vendido', atualizado_em = $2
		WHERE id IN (
			SELECT ip.id_anuncio
			FROM item_pedido ip
			JOIN pedido p ON p.id = ip.id_pedido
			WHERE p.id_compra = $1
		) AND status = 'reservado'
	`, idCompra, agora)
	if err != nil {
		return compras.Compra{}, mapDatabaseError(err)
	}
	var quantidadeItens int64
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM item_pedido ip
		JOIN pedido p ON p.id = ip.id_pedido
		WHERE p.id_compra = $1
	`, idCompra).Scan(&quantidadeItens); err != nil {
		return compras.Compra{}, mapDatabaseError(err)
	}
	if resultado.RowsAffected() != quantidadeItens {
		return compras.Compra{}, common.ErrTransicaoInvalida
	}
	if _, err := tx.Exec(ctx, `
		UPDATE compra SET status = 'aprovada', atualizado_em = $2 WHERE id = $1
	`, idCompra, agora); err != nil {
		return compras.Compra{}, mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pedido SET status = 'aguardando_envio', atualizado_em = $2
		WHERE id_compra = $1 AND status = 'aguardando_pagamento'
	`, idCompra, agora); err != nil {
		return compras.Compra{}, mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pagamento
		SET provedor = $2, identificador_externo = NULLIF($3, ''), status = 'aprovado',
		    pago_em = $4, atualizado_em = $4
		WHERE id_compra = $1 AND status = 'pendente'
	`, idCompra, provedor, identificadorExterno, agora); err != nil {
		return compras.Compra{}, mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM carrinho_anuncio
		WHERE id_carrinho = (SELECT id FROM carrinho WHERE id_usuario = $1)
		  AND id_anuncio IN (
			SELECT ip.id_anuncio FROM item_pedido ip
			JOIN pedido p ON p.id = ip.id_pedido
			WHERE p.id_compra = $2
		  )
	`, idComprador, idCompra); err != nil {
		return compras.Compra{}, mapDatabaseError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return compras.Compra{}, err
	}
	return s.BuscarCompraPorChave(ctx, chave)
}

// RecusarCompra registra a recusa e devolve os anuncios reservados ao catalogo.
func (s *Store) RecusarCompra(
	ctx context.Context,
	chave, provedor, identificadorExterno string,
	agora time.Time,
) error {
	return s.finalizarCompraPendente(ctx, chave, provedor, identificadorExterno, agora, false)
}

func (s *Store) finalizarCompraPendente(
	ctx context.Context,
	chave, provedor, identificadorExterno string,
	agora time.Time,
	expirada bool,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	statusCompra := "recusada"
	statusPedido := "cancelado"
	statusPagamento := "recusado"
	if expirada {
		statusCompra = "expirada"
		statusPedido = "expirado"
	}
	var idCompra, idComprador string
	var atual compras.StatusCompra
	if err := tx.QueryRow(ctx, `
		SELECT id, id_comprador, status FROM compra WHERE chave_idempotencia = $1 FOR UPDATE
	`, chave).Scan(&idCompra, &idComprador, &atual); errors.Is(err, pgx.ErrNoRows) {
		return common.ErrNaoEncontrado
	} else if err != nil {
		return mapDatabaseError(err)
	}
	if atual != compras.StatusCompraAguardandoPagamento {
		if atual == compras.StatusCompraRecusada || atual == compras.StatusCompraExpirada {
			return tx.Commit(ctx)
		}
		return common.ErrTransicaoInvalida
	}
	if _, err := tx.Exec(ctx, `
		UPDATE anuncio SET status = 'disponivel', atualizado_em = $2
		WHERE status = 'reservado' AND id IN (
			SELECT ip.id_anuncio FROM item_pedido ip
			JOIN pedido p ON p.id = ip.id_pedido WHERE p.id_compra = $1
		)
	`, idCompra, agora); err != nil {
		return mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE compra SET status = $2, atualizado_em = $3 WHERE id = $1
	`, idCompra, statusCompra, agora); err != nil {
		return mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pedido SET status = $2, atualizado_em = $3
		WHERE id_compra = $1 AND status = 'aguardando_pagamento'
	`, idCompra, statusPedido, agora); err != nil {
		return mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pagamento
		SET provedor = CASE WHEN $2 = '' THEN provedor ELSE $2 END,
		    identificador_externo = NULLIF($3, ''), status = $4, atualizado_em = $5
		WHERE id_compra = $1 AND status = 'pendente'
	`, idCompra, provedor, identificadorExterno, statusPagamento, agora); err != nil {
		return mapDatabaseError(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE carrinho SET atualizado_em = $2 WHERE id_usuario = $1
	`, idComprador, agora); err != nil {
		return mapDatabaseError(err)
	}
	return tx.Commit(ctx)
}

// ExpirarComprasPendentes libera reservas abandonadas sem depender de interacao do usuario.
func (s *Store) ExpirarComprasPendentes(ctx context.Context, agora time.Time) (int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT chave_idempotencia
		FROM compra
		WHERE status = 'aguardando_pagamento' AND expira_em <= $1
		ORDER BY expira_em
	`, agora)
	if err != nil {
		return 0, mapDatabaseError(err)
	}
	var chaves []string
	for rows.Next() {
		var chave string
		if err := rows.Scan(&chave); err != nil {
			rows.Close()
			return 0, err
		}
		chaves = append(chaves, chave)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	afetadas := 0
	for _, chave := range chaves {
		if err := s.finalizarCompraPendente(ctx, chave, "", "", agora, true); err != nil &&
			!errors.Is(err, common.ErrTransicaoInvalida) {
			return afetadas, err
		}
		afetadas++
	}
	return afetadas, nil
}

func (s *Store) BuscarCompraPorChave(ctx context.Context, chave string) (compras.Compra, error) {
	var compra compras.Compra
	err := s.pool.QueryRow(ctx, `
		SELECT id, id_comprador, status, valor_itens_centavos, valor_fretes_centavos,
		       valor_taxa_servico_centavos, valor_total_centavos, chave_idempotencia,
		       expira_em, criado_em
		FROM compra
		WHERE chave_idempotencia = $1
	`, chave).Scan(
		&compra.ID, &compra.IDComprador, &compra.Status, &compra.ValorTotalItensCentavos,
		&compra.ValorTotalFretesCentavos, &compra.ValorTaxaServicoCentavos,
		&compra.ValorFinalPagoCentavos, &compra.ChaveIdempotencia, &compra.ExpiraEm,
		&compra.CriadaEm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return compras.Compra{}, common.ErrNaoEncontrado
	}
	if err != nil {
		return compras.Compra{}, mapDatabaseError(err)
	}
	compra.Pedidos, err = s.buscarPedidos(ctx, `p.id_compra = $1`, compra.ID)
	return compra, err
}

func (s *Store) ListarPedidosDoComprador(ctx context.Context, idComprador string) ([]compras.Pedido, error) {
	return s.buscarPedidos(ctx, `p.id_comprador = $1`, idComprador)
}

func (s *Store) buscarPedidos(ctx context.Context, condicao string, args ...any) ([]compras.Pedido, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.id_compra, p.id_comprador, p.id_vendedor, p.status,
		       p.valor_itens_centavos, p.valor_frete_centavos, p.valor_taxa_servico_centavos,
		       p.valor_liquido_vendedor_centavos, p.nome_destinatario, p.cep_destino,
		       p.logradouro_destino, p.numero_destino, COALESCE(p.complemento_destino, ''),
		       p.bairro_destino, p.cidade_destino, p.estado_destino, p.criado_em, p.finalizado_em,
		       e.id, COALESCE(e.provedor, ''), COALESCE(e.codigo_rastreio, ''), e.status,
		       e.valor_frete_centavos, e.postado_em, e.entregue_em
		FROM pedido p
		LEFT JOIN entrega e ON e.id_pedido = p.id
		WHERE `+condicao+`
		ORDER BY p.criado_em DESC, p.id
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pedidos []compras.Pedido
	indicePorID := make(map[string]int)
	for rows.Next() {
		var p compras.Pedido
		var endereco = &p.EnderecoEntrega
		var (
			idEntrega       *string
			provedorEntrega string
			rastreioEntrega string
			statusEntrega   *compras.StatusEntrega
			freteEntrega    int64
			postadoEm       *time.Time
			entregueEm      *time.Time
		)
		if err := rows.Scan(
			&p.ID, &p.IDCompra, &p.IDComprador, &p.IDVendedor, &p.Status,
			&p.ValorTotalItensCentavos, &p.ValorFreteCentavos, &p.TaxaServicoCentavos,
			&p.ValorLiquidoVendedorCentavos, &p.NomeDestinatario, &endereco.CEP,
			&endereco.Logradouro, &endereco.Numero, &endereco.Complemento,
			&endereco.Bairro, &endereco.Cidade, &endereco.Estado, &p.CriadoEm, &p.FinalizadoEm,
			&idEntrega, &provedorEntrega, &rastreioEntrega, &statusEntrega,
			&freteEntrega, &postadoEm, &entregueEm,
		); err != nil {
			return nil, err
		}
		if idEntrega != nil && statusEntrega != nil {
			p.Entrega = &compras.Entrega{
				ID:                 *idEntrega,
				IDPedido:           p.ID,
				Provedor:           provedorEntrega,
				CodigoRastreio:     rastreioEntrega,
				Status:             *statusEntrega,
				ValorFreteCentavos: freteEntrega,
				PostadoEm:          postadoEm,
				EntregueEm:         entregueEm,
			}
		}
		indicePorID[p.ID] = len(pedidos)
		pedidos = append(pedidos, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if len(pedidos) == 0 {
		return pedidos, nil
	}
	idsPedidos := make([]string, 0, len(pedidos))
	for _, p := range pedidos {
		idsPedidos = append(idsPedidos, p.ID)
	}
	if err := s.carregarItensPedidos(ctx, pedidos, indicePorID, idsPedidos); err != nil {
		return nil, err
	}
	return pedidos, nil
}

func (s *Store) carregarItensPedidos(
	ctx context.Context,
	pedidos []compras.Pedido,
	indicePorID map[string]int,
	idsPedidos []string,
) error {
	rows, err := s.pool.Query(ctx, `
		SELECT id_pedido::text, id, id_anuncio::text, status, titulo, categoria, tamanho, cor,
		       estado_conservacao, valor_unitario_centavos, taxa_servico_centavos,
		       prazo_envio_em, enviado_em, recebido_em
		FROM item_pedido
		WHERE id_pedido::text = ANY($1::text[])
		ORDER BY id_pedido, criado_em, id
	`, idsPedidos)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var idPedido string
		var item compras.ItemPedido
		if err := rows.Scan(
			&idPedido, &item.ID, &item.IDAnuncio, &item.Status, &item.Titulo, &item.Categoria,
			&item.Tamanho, &item.Cor, &item.EstadoConservacao, &item.ValorUnitarioCentavos,
			&item.TaxaServicoCentavos, &item.PrazoEnvioEm, &item.EnviadoEm, &item.RecebidoEm,
		); err != nil {
			return err
		}
		item.IDPedido = idPedido
		if indice, ok := indicePorID[idPedido]; ok {
			pedidos[indice].Itens = append(pedidos[indice].Itens, item)
		}
	}
	return rows.Err()
}
