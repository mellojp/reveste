// Package db expõe as migrações SQL versionadas, embarcadas no binário.
//
// Os arquivos seguem o formato do golang-migrate: NNN_nome.up.sql aplica a
// migração e NNN_nome.down.sql a reverte. Embarcá-los garante que a ferramenta
// de migração leve exatamente o mesmo conteúdo versionado no repositório, sem
// depender de arquivos soltos no ambiente de deploy.
package db

import "embed"

// Migracoes contém os arquivos de migração em db/migrations.
//
//go:embed migrations/*.sql
var Migracoes embed.FS
