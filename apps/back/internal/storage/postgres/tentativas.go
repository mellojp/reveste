package postgres

import (
	"context"
	"time"
)

// ContarTentativas conta as tentativas de login da chave dentro da janela (>= desde).
func (s *Store) ContarTentativas(ctx context.Context, chave string, desde time.Time) (int, error) {
	var quantidade int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tentativa_login WHERE chave = $1 AND criada_em >= $2
	`, chave, desde).Scan(&quantidade)
	return quantidade, err
}

// RegistrarTentativa grava uma falha de login e remove registros antigos da mesma chave,
// limitando o crescimento da tabela sem depender de um job externo.
func (s *Store) RegistrarTentativa(ctx context.Context, chave string, em time.Time) error {
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM tentativa_login WHERE chave = $1 AND criada_em < $2
	`, chave, em.Add(-time.Hour)); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tentativa_login (chave, criada_em) VALUES ($1, $2)
	`, chave, em)
	return err
}

func (s *Store) LimparTentativas(ctx context.Context, chave string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM tentativa_login WHERE chave = $1`, chave)
	return err
}
