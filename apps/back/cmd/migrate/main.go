// Command migrate aplica as migrações versionadas do banco PostgreSQL.
//
// As migrações ficam embarcadas no binário (ver pacote reveste/db) e seguem o
// formato do golang-migrate. A URL do banco vem de DATABASE_URL, lida do .env
// (quando presente) ou do ambiente.
//
// Uso:
//
//	migrate up                aplica todas as migrações pendentes
//	migrate down [N]          reverte N migrações (padrão: 1)
//	migrate goto <versao>     migra para uma versão específica
//	migrate force <versao>    fixa a versão sem executar (baseline/recuperação)
//	migrate version           mostra a versão atual e se está "dirty"
//	migrate drop              remove todos os objetos (apenas desenvolvimento)
//
// Em um banco já criado fora desta ferramenta (por exemplo, pelo antigo
// docker-entrypoint-initdb.d), use "force <versao>" uma única vez para registrar
// a versão já aplicada antes de rodar "up".
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/joho/godotenv"

	"reveste/db"
)

func main() {
	log.SetFlags(0)
	if err := executar(os.Args[1:]); err != nil {
		log.Fatalf("migrate: %v", err)
	}
}

func executar(args []string) error {
	if len(args) == 0 {
		return errors.New("informe um comando: up, down, goto, force, version ou drop")
	}

	_ = godotenv.Load()
	urlBanco := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if urlBanco == "" {
		return errors.New("DATABASE_URL não foi definida no .env ou no ambiente")
	}

	migrador, err := novoMigrador(urlBanco)
	if err != nil {
		return err
	}
	defer fechar(migrador)

	comando := args[0]
	switch comando {
	case "up":
		return aplicar(migrador.Up())
	case "down":
		passos := 1
		if len(args) > 1 {
			passos, err = inteiroPositivo(args[1])
			if err != nil {
				return fmt.Errorf("quantidade inválida para down: %w", err)
			}
		}
		return aplicar(migrador.Steps(-passos))
	case "goto":
		if len(args) < 2 {
			return errors.New("goto exige a versão de destino")
		}
		versao, err := inteiroPositivo(args[1])
		if err != nil {
			return fmt.Errorf("versão inválida para goto: %w", err)
		}
		return aplicar(migrador.Migrate(uint(versao)))
	case "force":
		if len(args) < 2 {
			return errors.New("force exige a versão a ser fixada")
		}
		versao, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("versão inválida para force: %q", args[1])
		}
		if err := migrador.Force(versao); err != nil {
			return err
		}
		log.Printf("versão fixada em %d", versao)
		return nil
	case "version":
		return mostrarVersao(migrador)
	case "drop":
		if err := migrador.Drop(); err != nil {
			return err
		}
		log.Println("banco esvaziado")
		return nil
	default:
		return fmt.Errorf("comando desconhecido: %q", comando)
	}
}

func novoMigrador(urlBanco string) (*migrate.Migrate, error) {
	origem, err := iofs.New(db.Migracoes, "migrations")
	if err != nil {
		return nil, fmt.Errorf("carregar migrações embarcadas: %w", err)
	}
	migrador, err := migrate.NewWithSourceInstance("iofs", origem, urlPgx(urlBanco))
	if err != nil {
		return nil, fmt.Errorf("conectar ao banco: %w", err)
	}
	return migrador, nil
}

// urlPgx adapta o esquema da URL para o driver pgx/v5 do golang-migrate, que
// espera "pgx5://". Mantém a URL intacta se o esquema já for outro.
func urlPgx(u string) string {
	for _, prefixo := range []string{"postgresql://", "postgres://"} {
		if rest, ok := strings.CutPrefix(u, prefixo); ok {
			return "pgx5://" + rest
		}
	}
	return u
}

// aplicar trata ErrNoChange como sucesso: rodar "up"/"down" sem migrações
// pendentes não é um erro.
func aplicar(err error) error {
	if err == nil {
		log.Println("migrações aplicadas")
		return nil
	}
	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("nenhuma migração pendente")
		return nil
	}
	return err
}

func mostrarVersao(migrador *migrate.Migrate) error {
	versao, dirty, err := migrador.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		log.Println("nenhuma migração aplicada")
		return nil
	}
	if err != nil {
		return err
	}
	estado := "ok"
	if dirty {
		estado = "dirty (requer correção com force)"
	}
	log.Printf("versão %d (%s)", versao, estado)
	return nil
}

func inteiroPositivo(valor string) (int, error) {
	n, err := strconv.Atoi(valor)
	if err != nil {
		return 0, fmt.Errorf("%q não é um número", valor)
	}
	if n <= 0 {
		return 0, fmt.Errorf("deve ser maior que zero, recebido %d", n)
	}
	return n, nil
}

func fechar(migrador *migrate.Migrate) {
	if errOrigem, errBanco := migrador.Close(); errOrigem != nil || errBanco != nil {
		log.Printf("aviso ao encerrar: origem=%v banco=%v", errOrigem, errBanco)
	}
}
