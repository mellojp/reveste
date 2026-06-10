package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/cadastros"
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
