package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
	errosdominio "reveste/apps/api/internal/dominio/erros"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("configurar PostgreSQL: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("criar pool PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("conectar ao PostgreSQL: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

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
	return s.findUser(ctx, `u.id = $1`, id)
}

func (s *Store) BuscarUsuarioPorEmailOuCPF(ctx context.Context, identificador string) (cadastros.Usuario, error) {
	email := strings.ToLower(strings.TrimSpace(identificador))
	cpf := cadastros.NormalizarCPF(identificador)
	return s.findUser(ctx, `(u.email = $1 OR u.cpf = $2)`, email, cpf)
}

func (s *Store) findUser(ctx context.Context, condition string, args ...any) (cadastros.Usuario, error) {
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
		WHERE u.excluido_em IS NULL AND `+condition+`
	`, args...).Scan(
		&usuario.ID, &usuario.Nome, &usuario.CPF, &usuario.Email, &usuario.HashSenha,
		&usuario.Telefone, &usuario.ItensNaoEnviados, &usuario.BloqueadoParaVendas,
		&usuario.CriadoEm, &usuario.AtualizadoEm, &usuario.EnderecoPrincipal.CEP,
		&usuario.EnderecoPrincipal.Logradouro, &usuario.EnderecoPrincipal.Numero,
		&usuario.EnderecoPrincipal.Complemento, &usuario.EnderecoPrincipal.Bairro,
		&usuario.EnderecoPrincipal.Cidade, &usuario.EnderecoPrincipal.Estado,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return cadastros.Usuario{}, errosdominio.ErrNaoEncontrado
	}
	return usuario, err
}

func (s *Store) CriarAnuncio(ctx context.Context, anuncio anuncios.Anuncio) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO anuncio (
			id, id_perfil_vendedor, titulo, descricao, categoria, tamanho, cor,
			estado_conservacao, preco_centavos, status, criado_em, atualizado_em
		) SELECT $1, pv.id, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		  FROM perfil_vendedor pv
		 WHERE pv.id_usuario = $2
	`, anuncio.ID, anuncio.IDVendedor, anuncio.Titulo, anuncio.Descricao,
		anuncio.Categoria, anuncio.Tamanho, anuncio.Cor, anuncio.EstadoConservacao,
		anuncio.PrecoCentavos, anuncio.Status, anuncio.CriadoEm, anuncio.AtualizadoEm)
	if err != nil {
		return mapDatabaseError(err)
	}
	for _, foto := range anuncio.Fotos {
		_, err = tx.Exec(ctx, `
			INSERT INTO foto_anuncio (id, id_anuncio, url, ordem, legenda, criado_em)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
		`, foto.ID, anuncio.ID, foto.URL, foto.Ordem, foto.Legenda, anuncio.CriadoEm)
		if err != nil {
			return mapDatabaseError(err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) BuscarAnuncioPorID(ctx context.Context, id string) (anuncios.Anuncio, error) {
	var anuncio anuncios.Anuncio
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, pv.id_usuario, a.titulo, a.descricao, a.categoria, a.tamanho, a.cor,
		       a.estado_conservacao, a.preco_centavos, a.status, a.criado_em,
		       a.atualizado_em, a.excluido_em
		FROM anuncio a
		JOIN perfil_vendedor pv ON pv.id = a.id_perfil_vendedor
		WHERE a.id = $1 AND a.excluido_em IS NULL
	`, id).Scan(
		&anuncio.ID, &anuncio.IDVendedor, &anuncio.Titulo, &anuncio.Descricao,
		&anuncio.Categoria, &anuncio.Tamanho, &anuncio.Cor, &anuncio.EstadoConservacao,
		&anuncio.PrecoCentavos, &anuncio.Status, &anuncio.CriadoEm,
		&anuncio.AtualizadoEm, &anuncio.ExcluidoEm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return anuncios.Anuncio{}, errosdominio.ErrNaoEncontrado
	}
	if err != nil {
		return anuncios.Anuncio{}, err
	}
	anuncio.Fotos, err = s.findAdPhotos(ctx, anuncio.ID)
	return anuncio, err
}

func (s *Store) ListarAnuncios(
	ctx context.Context,
	filtro casosdeuso.FiltroAnuncios,
) ([]anuncios.Anuncio, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, pv.id_usuario, a.titulo, a.descricao, a.categoria, a.tamanho, a.cor,
		       a.estado_conservacao, a.preco_centavos, a.status, a.criado_em, a.atualizado_em
		FROM anuncio a
		JOIN perfil_vendedor pv ON pv.id = a.id_perfil_vendedor
		WHERE a.status = 'disponivel'
		  AND a.excluido_em IS NULL
		  AND ($1 = '' OR a.titulo ILIKE '%' || $1 || '%' OR a.descricao ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR a.categoria = LOWER($2))
		  AND ($3 = '' OR a.tamanho = UPPER($3))
		  AND ($4 = '' OR a.estado_conservacao = $4)
		  AND ($5 = 0 OR a.preco_centavos >= $5)
		  AND ($6 = 0 OR a.preco_centavos <= $6)
		ORDER BY a.criado_em DESC
		LIMIT $7 OFFSET $8
	`, strings.TrimSpace(filtro.Palavra), filtro.Categoria, filtro.Tamanho,
		filtro.EstadoConservacao, filtro.PrecoMinCentavos, filtro.PrecoMaxCentavos,
		filtro.Limite, filtro.Deslocamento)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultado []anuncios.Anuncio
	for rows.Next() {
		var anuncio anuncios.Anuncio
		if err := rows.Scan(
			&anuncio.ID, &anuncio.IDVendedor, &anuncio.Titulo, &anuncio.Descricao,
			&anuncio.Categoria, &anuncio.Tamanho, &anuncio.Cor, &anuncio.EstadoConservacao,
			&anuncio.PrecoCentavos, &anuncio.Status, &anuncio.CriadoEm, &anuncio.AtualizadoEm,
		); err != nil {
			return nil, err
		}
		anuncio.Fotos, err = s.findAdPhotos(ctx, anuncio.ID)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, anuncio)
	}
	return resultado, rows.Err()
}

func (s *Store) findAdPhotos(ctx context.Context, idAnuncio string) ([]anuncios.Foto, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, url, ordem, COALESCE(legenda, '')
		FROM foto_anuncio
		WHERE id_anuncio = $1
		ORDER BY ordem
	`, idAnuncio)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fotos []anuncios.Foto
	for rows.Next() {
		var foto anuncios.Foto
		if err := rows.Scan(&foto.ID, &foto.URL, &foto.Ordem, &foto.Legenda); err != nil {
			return nil, err
		}
		fotos = append(fotos, foto)
	}
	return fotos, rows.Err()
}

func (s *Store) ObterOuCriarCarrinho(
	ctx context.Context,
	novoID string,
	idUsuario string,
	agora time.Time,
) (compras.Carrinho, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO carrinho (id, id_usuario, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (id_usuario) DO NOTHING
	`, novoID, idUsuario, agora)
	if err != nil {
		return compras.Carrinho{}, mapDatabaseError(err)
	}

	var carrinho compras.Carrinho
	err = s.pool.QueryRow(ctx, `
		SELECT id, id_usuario, criado_em, atualizado_em
		FROM carrinho
		WHERE id_usuario = $1
	`, idUsuario).Scan(&carrinho.ID, &carrinho.IDUsuario, &carrinho.CriadoEm, &carrinho.AtualizadoEm)
	if err != nil {
		return compras.Carrinho{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id_anuncio
		FROM carrinho_anuncio
		WHERE id_carrinho = $1
		ORDER BY adicionado_em
	`, carrinho.ID)
	if err != nil {
		return compras.Carrinho{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var idAnuncio string
		if err := rows.Scan(&idAnuncio); err != nil {
			return compras.Carrinho{}, err
		}
		carrinho.IDsAnuncios = append(carrinho.IDsAnuncios, idAnuncio)
	}
	return carrinho, rows.Err()
}

func (s *Store) SalvarCarrinho(ctx context.Context, carrinho compras.Carrinho) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE carrinho SET atualizado_em = $2 WHERE id = $1
	`, carrinho.ID, carrinho.AtualizadoEm); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM carrinho_anuncio WHERE id_carrinho = $1`, carrinho.ID); err != nil {
		return err
	}
	for _, idAnuncio := range carrinho.IDsAnuncios {
		if _, err := tx.Exec(ctx, `
			INSERT INTO carrinho_anuncio (id_carrinho, id_anuncio, adicionado_em)
			VALUES ($1, $2, $3)
		`, carrinho.ID, idAnuncio, carrinho.AtualizadoEm); err != nil {
			return mapDatabaseError(err)
		}
	}
	return tx.Commit(ctx)
}

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
		SELECT id_usuario
		FROM sessao
		WHERE token_hash = $1 AND expira_em > $2
	`, hashToken(token), agora).Scan(&idUsuario)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errosdominio.ErrNaoAutorizado
	}
	return idUsuario, err
}

func (s *Store) RemoverSessao(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessao WHERE token_hash = $1`, hashToken(token))
	return err
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func mapDatabaseError(err error) error {
	if err == nil {
		return nil
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "23505":
			return errosdominio.ErrConflito
		case "23503":
			return errosdominio.ErrNaoEncontrado
		case "23514", "23502", "22P02":
			return errosdominio.ErrDadosInvalidos
		}
	}
	return err
}
