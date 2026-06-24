package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"reveste/apps/back/internal/common"
	"reveste/apps/back/internal/dominio/cadastros"
)

func (s *Store) CriarUsuario(ctx context.Context, usuario cadastros.Usuario) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO usuario (
			id, nome, cpf, email, hash_senha, telefone, criado_em, atualizado_em
		) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8)
	`, usuario.ID, usuario.Nome, usuario.CPF, usuario.Email, usuario.HashSenha,
		usuario.Telefone, usuario.CriadoEm, usuario.AtualizadoEm)
	if err != nil {
		return mapDatabaseError(err)
	}

	endereco := usuario.EnderecoPrincipal
	_, err = tx.Exec(ctx, `
		INSERT INTO endereco (
			id_usuario, cep, logradouro, numero, complemento, bairro, cidade,
			estado, principal, criado_em, atualizado_em
		) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, TRUE, $9, $9)
	`, usuario.ID, endereco.CEP, endereco.Logradouro, endereco.Numero,
		endereco.Complemento, endereco.Bairro, endereco.Cidade, endereco.Estado,
		usuario.CriadoEm)
	if err != nil {
		return mapDatabaseError(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO perfil_vendedor (
			id_usuario, itens_nao_enviados, bloqueado, criado_em, atualizado_em
		) VALUES ($1, $2, $3, $4, $4)
	`, usuario.ID, usuario.ItensNaoEnviados, usuario.BloqueadoParaVendas, usuario.CriadoEm)
	if err != nil {
		return mapDatabaseError(err)
	}
	return tx.Commit(ctx)
}

func (s *Store) AtualizarUsuario(ctx context.Context, usuario cadastros.Usuario) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	resultado, err := tx.Exec(ctx, `
		UPDATE usuario
		SET nome = $2, email = $3, telefone = NULLIF($4, ''), atualizado_em = $5
		WHERE id = $1 AND excluido_em IS NULL
	`, usuario.ID, usuario.Nome, usuario.Email, usuario.Telefone, usuario.AtualizadoEm)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoEncontrado
	}
	endereco := usuario.EnderecoPrincipal
	resultado, err = tx.Exec(ctx, `
		UPDATE endereco
		SET cep = $2, logradouro = $3, numero = $4, complemento = NULLIF($5, ''),
		    bairro = $6, cidade = $7, estado = $8, atualizado_em = $9
		WHERE id_usuario = $1 AND principal = TRUE AND excluido_em IS NULL
	`, usuario.ID, endereco.CEP, endereco.Logradouro, endereco.Numero,
		endereco.Complemento, endereco.Bairro, endereco.Cidade, endereco.Estado,
		usuario.AtualizadoEm)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoEncontrado
	}
	return tx.Commit(ctx)
}

func (s *Store) BuscarUsuarioPorID(ctx context.Context, id string) (cadastros.Usuario, error) {
	return s.buscarUsuario(ctx, `u.id = $1`, id)
}

func (s *Store) BuscarUsuarioPorEmailOuCPF(ctx context.Context, identificador string) (cadastros.Usuario, error) {
	email := strings.ToLower(strings.TrimSpace(identificador))
	cpf := cadastros.NormalizarCPF(identificador)
	return s.buscarUsuario(ctx, `(u.email = $1 OR u.cpf = $2)`, email, cpf)
}

func (s *Store) buscarUsuario(ctx context.Context, condicao string, args ...any) (cadastros.Usuario, error) {
	var usuario cadastros.Usuario
	err := s.pool.QueryRow(ctx, `
		SELECT
			u.id, u.nome, u.cpf, u.email, u.hash_senha, COALESCE(u.telefone, ''),
			pv.itens_nao_enviados, pv.bloqueado, u.criado_em, u.atualizado_em,
			e.cep, e.logradouro, e.numero, COALESCE(e.complemento, ''),
			e.bairro, e.cidade, e.estado
		FROM usuario u
		JOIN perfil_vendedor pv ON pv.id_usuario = u.id
		JOIN endereco e
		  ON e.id_usuario = u.id AND e.principal = TRUE AND e.excluido_em IS NULL
		WHERE u.excluido_em IS NULL AND `+condicao+`
	`, args...).Scan(
		&usuario.ID, &usuario.Nome, &usuario.CPF, &usuario.Email, &usuario.HashSenha,
		&usuario.Telefone, &usuario.ItensNaoEnviados, &usuario.BloqueadoParaVendas,
		&usuario.CriadoEm, &usuario.AtualizadoEm, &usuario.EnderecoPrincipal.CEP,
		&usuario.EnderecoPrincipal.Logradouro, &usuario.EnderecoPrincipal.Numero,
		&usuario.EnderecoPrincipal.Complemento, &usuario.EnderecoPrincipal.Bairro,
		&usuario.EnderecoPrincipal.Cidade, &usuario.EnderecoPrincipal.Estado,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return cadastros.Usuario{}, common.ErrNaoEncontrado
	}
	return usuario, err
}

// ReativarVendedor desfaz o bloqueio do vendedor e zera o contador de itens nao enviados.
// So afeta perfis efetivamente bloqueados (idempotente via WHERE); o bool informa se havia
// um bloqueio para reverter.
func (s *Store) ReativarVendedor(ctx context.Context, idVendedor string, agora time.Time) (bool, error) {
	resultado, err := s.pool.Exec(ctx, `
		UPDATE perfil_vendedor
		SET bloqueado = FALSE, itens_nao_enviados = 0, atualizado_em = $2
		WHERE id_usuario = $1 AND bloqueado = TRUE
	`, idVendedor, agora)
	if err != nil {
		return false, mapDatabaseError(err)
	}
	return resultado.RowsAffected() > 0, nil
}

func (s *Store) ListarEnderecos(ctx context.Context, idUsuario string) ([]cadastros.Endereco, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, cep, logradouro, numero, COALESCE(complemento, ''),
		       bairro, cidade, estado, principal
		FROM endereco
		WHERE id_usuario = $1 AND excluido_em IS NULL
		ORDER BY principal DESC, criado_em
	`, idUsuario)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()

	enderecos := make([]cadastros.Endereco, 0)
	for rows.Next() {
		var e cadastros.Endereco
		if err := rows.Scan(
			&e.ID, &e.CEP, &e.Logradouro, &e.Numero, &e.Complemento,
			&e.Bairro, &e.Cidade, &e.Estado, &e.Principal,
		); err != nil {
			return nil, err
		}
		enderecos = append(enderecos, e)
	}
	return enderecos, rows.Err()
}

func (s *Store) BuscarEndereco(ctx context.Context, idUsuario, idEndereco string) (cadastros.Endereco, error) {
	var e cadastros.Endereco
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, cep, logradouro, numero, COALESCE(complemento, ''),
		       bairro, cidade, estado, principal
		FROM endereco
		WHERE id = $2 AND id_usuario = $1 AND excluido_em IS NULL
	`, idUsuario, idEndereco).Scan(
		&e.ID, &e.CEP, &e.Logradouro, &e.Numero, &e.Complemento,
		&e.Bairro, &e.Cidade, &e.Estado, &e.Principal,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return cadastros.Endereco{}, common.ErrNaoEncontrado
	}
	if err != nil {
		return cadastros.Endereco{}, mapDatabaseError(err)
	}
	return e, nil
}

func (s *Store) AdicionarEndereco(ctx context.Context, idUsuario string, endereco cadastros.Endereco, agora time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO endereco (
			id, id_usuario, cep, logradouro, numero, complemento, bairro, cidade,
			estado, principal, criado_em, atualizado_em
		) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, FALSE, $10, $10)
	`, endereco.ID, idUsuario, endereco.CEP, endereco.Logradouro, endereco.Numero,
		endereco.Complemento, endereco.Bairro, endereco.Cidade, endereco.Estado, agora)
	return mapDatabaseError(err)
}

func (s *Store) AtualizarEndereco(ctx context.Context, idUsuario, idEndereco string, endereco cadastros.Endereco, agora time.Time) error {
	resultado, err := s.pool.Exec(ctx, `
		UPDATE endereco
		SET cep = $3, logradouro = $4, numero = $5, complemento = NULLIF($6, ''),
		    bairro = $7, cidade = $8, estado = $9, atualizado_em = $10
		WHERE id = $2 AND id_usuario = $1 AND excluido_em IS NULL
	`, idUsuario, idEndereco, endereco.CEP, endereco.Logradouro, endereco.Numero,
		endereco.Complemento, endereco.Bairro, endereco.Cidade, endereco.Estado, agora)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoEncontrado
	}
	return nil
}

func (s *Store) RemoverEndereco(ctx context.Context, idUsuario, idEndereco string, agora time.Time) error {
	resultado, err := s.pool.Exec(ctx, `
		UPDATE endereco
		SET excluido_em = $3, atualizado_em = $3
		WHERE id = $2 AND id_usuario = $1 AND excluido_em IS NULL
	`, idUsuario, idEndereco, agora)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoEncontrado
	}
	return nil
}

// DefinirEnderecoPrincipal zera o principal atual e marca o escolhido, em uma transacao.
//
// A ordem importa: o indice unico parcial uq_endereco_principal_usuario garante no maximo um
// endereco principal por usuario. Por isso desmarcamos todos antes de marcar o escolhido —
// marcar primeiro deixaria dois principais simultaneos e violaria o indice.
func (s *Store) DefinirEnderecoPrincipal(ctx context.Context, idUsuario, idEndereco string, agora time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE endereco SET principal = FALSE, atualizado_em = $2
		WHERE id_usuario = $1 AND principal = TRUE AND excluido_em IS NULL
	`, idUsuario, agora); err != nil {
		return mapDatabaseError(err)
	}
	resultado, err := tx.Exec(ctx, `
		UPDATE endereco SET principal = TRUE, atualizado_em = $3
		WHERE id = $2 AND id_usuario = $1 AND excluido_em IS NULL
	`, idUsuario, idEndereco, agora)
	if err != nil {
		return mapDatabaseError(err)
	}
	if resultado.RowsAffected() == 0 {
		return common.ErrNaoEncontrado
	}
	return tx.Commit(ctx)
}
