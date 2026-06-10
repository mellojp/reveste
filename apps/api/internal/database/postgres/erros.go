package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"reveste/apps/api/internal/common"
)

func mapDatabaseError(err error) error {
	if err == nil {
		return nil
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "23505":
			return common.ErrConflito
		case "23503":
			return common.ErrNaoEncontrado
		case "23514", "23502", "22P02":
			return common.ErrDadosInvalidos
		}
	}
	return err
}
