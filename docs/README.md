# Estrutura da aplicacao

O fluxo principal da API e:

```text
HTTP (adaptador de entrada) -> controladores de casos de uso -> dominio
                                      |
                                      v
                         portas -> database (adaptador de saida)
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
resultado em resposta HTTP. O pacote `database/postgres` e um adaptador de saida
que implementa as interfaces exigidas pelos controladores.

O adaptador PostgreSQL permanece em um unico pacote e usa o tipo compartilhado
`Store`, mas suas operacoes sao separadas por arquivo: `usuarios.go`,
`anuncios.go`, `carrinhos.go` e `sessoes.go`. Conexao e configuracao ficam em
`store.go`, enquanto a traducao de erros do driver fica em `erros.go`.

Os erros compartilhados pela aplicacao ficam em `internal/common/erros.go`.

Os testes de persistencia real usam um schema PostgreSQL isolado e sao ativados
quando `TEST_DATABASE_URL` esta definida:

```text
TEST_DATABASE_URL=postgres://reveste:reveste@localhost:5432/reveste?sslmode=disable
```

O `compose.yaml` aplica as migracoes `up` em ordem apenas na criacao inicial do
volume PostgreSQL. Em ambiente ja inicializado, novas migracoes precisam ser
executadas explicitamente.
