package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"reveste/apps/back/internal/common"
)

func mapDatabaseError(err error) error {
	if err == nil {
		return nil
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "23505":
			switch pgError.ConstraintName {
			case "usuario_cpf_key":
				return common.NovoConflitoCampo("cpf", "Já existe uma conta com este CPF.")
			case "uq_usuario_email_normalizado":
				return common.NovoConflitoCampo("email", "Já existe uma conta com este e-mail.")
			}
			return common.ErrConflito
		case "23503":
			return common.ErrNaoEncontrado
		case "23514", "23502", "22P02":
			return common.ErrDadosInvalidos
		}
	}
	return err
}
