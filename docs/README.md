# Estrutura da aplicacao

## Documentos relacionados

- `MODELO_CANONICO.md`: vocabulario e regras canonicas do dominio;
- `ALINHAMENTO_IMPLEMENTACAO.md`: correspondencia entre a modelagem e o codigo;
- `PLANO_IMPLEMENTACAO_INICIAL.md`: plano que orientou o primeiro incremento;
- `INCREMENTO_FLUXO_INICIAL.md`: registro consolidado das correcoes de backend,
  conclusao dos fluxos de perfil e anuncios e refinamentos do frontend.
- `MIGRACAO_HTMX_SSR.md`: arquitetura, fronteiras e regras da migracao SSR.

Os PDFs em `docs/base` registram etapas anteriores da modelagem acadêmica. Quando
divergirem da implementacao atual, prevalecem `MODELO_CANONICO.md`,
`MIGRACAO_HTMX_SSR.md` e `ALINHAMENTO_IMPLEMENTACAO.md`. Em particular, a arquitetura
atual usa SSR/HTMX e Vercel Blob, nao React SPA e Amazon S3.

O fluxo principal da API e:

```text
HTTP (adaptador de entrada) -> controladores de casos de uso -> dominio
                                      |
                                      v
                         portas -> storage (adaptadores de saida)
```

Os casos de uso ficam no pacote unico `casosdeuso`, separados por arquivo:

- `cadastro.go`: cadastro, autenticacao e sessoes;
- `anuncios.go`: publicacao e consulta de anuncios;
- `carrinho.go`: carrinho e futuros fluxos de checkout e pedidos.

## Papel dos controladores

Os tipos `ControladorCadastro`, `ControladorAnuncio` e `ControladorCarrinho`
seguem o padrao Controller do GRASP: representam uma funcionalidade ou grupo
coeso de casos de uso, coordenam objetos de dominio e acessam infraestrutura por
interfaces definidas em `casosdeuso/contratos.go`.

Eles nao sao controllers HTTP/MVC. O pacote `internal/http` e o adaptador de
entrada: decodifica a requisicao, chama um controlador de aplicacao e converte o
resultado em resposta HTTP. Os pacotes `storage/postgres` e `storage/vercel` sao
adaptadores de saida que implementam as interfaces exigidas pelos controladores.

O adaptador PostgreSQL permanece em um unico pacote e usa o tipo compartilhado
`Store`, mas suas operacoes sao separadas por arquivo: `usuarios.go`,
`anuncios.go`, `carrinhos.go` e `sessoes.go`. Conexao e configuracao ficam em
`store.go`, enquanto a traducao de erros do driver fica em `erros.go`.

O adaptador Vercel Blob fica em `storage/vercel` e implementa o armazenamento
externo dos arquivos de imagem.

Os erros compartilhados pela aplicacao ficam em `internal/common/erros.go`.

Os testes de persistencia real usam um schema PostgreSQL isolado e sao ativados
quando `TEST_DATABASE_URL` esta definida:

```text
TEST_DATABASE_URL=postgres://reveste:reveste@localhost:5432/reveste?sslmode=disable
```

O `compose.yaml` aplica as migracoes `up` em ordem apenas na criacao inicial do
volume PostgreSQL. Em ambiente ja inicializado, novas migracoes precisam ser
executadas explicitamente.

## Execucao local

Crie um arquivo `.env` na raiz:

```text
DATABASE_URL=postgres://reveste:reveste@localhost:5432/reveste?sslmode=disable
HTTP_ADDRESS=:8080
BLOB_READ_WRITE_TOKEN=vercel_blob_rw_SEU_STORE_ID_SEU_TOKEN
BLOB_PUBLIC_HOST=SEU_STORE_ID.public.blob.vercel-storage.com
JOBS_INTERVAL=1m
```

O token e obtido ao criar/conectar um Blob store **publico** no projeto da Vercel.
Stores privados nao podem ser usados diretamente nas imagens do catalogo.
`BLOB_PUBLIC_HOST` restringe imagens e a CSP ao store oficial; quando omitida, a API
deriva o hostname do identificador presente em `BLOB_READ_WRITE_TOKEN`.
Sem o token, a aplicacao inicia normalmente, mas o endpoint de upload retorna `503`.

O store publico deve ser exclusivo para fotos publicas dos anuncios. Conteudo
restrito ou sensivel deve usar outro store privado. A politica completa e os
controles pendentes estao em `docs/ALINHAMENTO_IMPLEMENTACAO.md`.

Inicie o banco e a API:

```text
docker compose up -d
go run ./apps/api/cmd/api
```

O frontend fica disponivel em `http://localhost:8080`.

O mesmo processo executa periodicamente os jobs de expiracao de reservas de checkout e
de vencimento do prazo de envio. `JOBS_INTERVAL` controla o intervalo e aceita duracoes
Go, como `30s`, `1m` ou `5m`.

As paginas sao renderizadas no servidor pelo adaptador `internal/web` e
aprimoradas com uma copia local do HTMX, sem bundler ou runtime SPA. O CSS e os
assets continuam em `apps/front`; JavaScript proprio e usado apenas para galeria,
edicao de fotos e upload. Os fluxos permitem editar o perfil, editar ou excluir
anuncios disponiveis e consultar vendedores sem expor dados privados.

Endpoints de monitoramento:

- `GET /saude`: liveness do processo;
- `GET /saude/prontidao`: readiness com verificacao da conexao PostgreSQL.

Para aplicar a migracao de categorias em um volume criado antes dela:

```text
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 \
  -U reveste -d reveste -f /dev/stdin < db/migrations/003_categorias_anuncio.up.sql
```

Para executar todos os testes, incluindo a integracao PostgreSQL:

```text
TEST_DATABASE_URL=postgres://reveste:reveste@localhost:5432/reveste?sslmode=disable \
  go test ./...
node --test apps/front/tests/*.test.mjs
```

## Organizacao dos testes

Os testes seguem uma estrutura hibrida:

```text
apps/api/
|-- internal/
|   |-- dominio/.../*_test.go
|   |-- casosdeuso/*_test.go
|   `-- storage/vercel/*_test.go
`-- tests/
    `-- integration/
        |-- http_support_test.go
        |-- http_routes_test.go
        |-- http_validation_test.go
        `-- postgres_test.go
```

- testes unitarios permanecem junto ao pacote testado, conforme a convencao Go;
- testes do adaptador HTTP e de persistencia PostgreSQL ficam em
  `apps/api/tests/integration`;
- fixtures compartilhadas por um conjunto de integracao ficam em arquivos
  `*_support_test.go` no mesmo pacote de testes;
- futuros testes de navegador e fluxos completos devem ficar em
  `apps/api/tests/e2e`.

Para executar apenas as integracoes:

```text
TEST_DATABASE_URL=postgres://reveste:reveste@localhost:5432/reveste?sslmode=disable \
  go test ./apps/api/tests/integration
```
