# Alinhamento entre UML e implementacao

## Correspondencia estrutural

| Diagrama de classes | Dominio Go | PostgreSQL |
|---|---|---|
| Usuario | `dominio/cadastros.Usuario` | `usuario` |
| Endereco | `dominio/cadastros.Endereco` | `endereco` |
| DadosBancarios | `dominio/cadastros.DadosBancarios` | `dados_bancarios` |
| PerfilVendedor | `dominio/cadastros.PerfilVendedor` | `perfil_vendedor` |
| Anuncio | `dominio/anuncios.Anuncio` | `anuncio` |
| Foto | `dominio/anuncios.Foto` | `foto_anuncio` |
| Carrinho | `dominio/compras.Carrinho` | `carrinho`, `carrinho_anuncio` |
| Compra | `dominio/compras.Compra` | `compra` |
| Pedido | `dominio/compras.Pedido` | `pedido` |
| ItemPedido | `dominio/compras.ItemPedido` | `item_pedido` |
| Pagamento | `dominio/compras.Pagamento` | `pagamento` |
| Reembolso | `dominio/compras.Reembolso` | `reembolso` |
| Entrega | `dominio/compras.Entrega` | `entrega` |
| Conversa | `dominio/interacao.Conversa` | `conversa` |
| Mensagem | `dominio/interacao.Mensagem` | `mensagem` |
| Notificacao | `dominio/interacao.Notificacao` | `notificacao` |
| Avaliacao | `dominio/interacao.Avaliacao` | `avaliacao` |

## Comportamentos ja executaveis

- cadastrar e autenticar usuario;
- consultar o perfil autenticado;
- criar anuncio;
- listar e filtrar anuncios;
- consultar os anuncios do usuario autenticado;
- adicionar e remover anuncio do carrinho;
- validar CPF, anuncio, quantidade de fotos, categoria e disponibilidade;
- apresentar um fluxo web navegavel de conta, catalogo, publicacao, perfil e carrinho.

## Incremento web inicial

O frontend inicial fica em `apps/front` e e servido pela propria API na rota `/`.
Ele foi mantido como HTML, CSS e JavaScript sem etapa de build enquanto a equipe
nao define a toolchain React/TypeScript prevista na arquitetura.

Telas e fluxos disponiveis:

- landing page e catalogo responsivo;
- busca por texto, categoria e estado de conservacao;
- cadastro com mensagens de validacao junto aos campos;
- login, logout e sessao mantida em `sessionStorage`;
- publicacao de anuncio com 2 a 5 URLs de fotos;
- perfil com dados pessoais, endereco e painel de anuncios;
- carrinho com inclusao e remocao de pecas.

O uso de `sessionStorage` e provisório e acompanha o contrato Bearer atual. A decisao
de autenticacao por cookie `HttpOnly`, protecao CSRF e deploy continua pendente antes
de producao.

## Contratos HTTP adicionados

- `GET /v1/me`: retorna o usuario da sessao atual;
- `GET /v1/me/anuncios`: retorna os anuncios publicados pelo usuario;
- `GET /v1/anuncios`: quando recebe um Bearer valido, omite anuncios do proprio
  usuario; sem autenticacao, continua publico;
- erros de cadastro podem preencher `campos` com mensagens especificas por input.

## Categorias canonicas

Anuncios aceitam somente:

- `vestidos`;
- `camisetas`;
- `calcas`;
- `saias_e_shorts`;
- `casacos`;
- `acessorios`;
- `calcados`;
- `outros`.

A regra existe no dominio, no transporte HTTP, no formulario web e na constraint
`ck_anuncio_categoria`. A migracao `003_categorias_anuncio` normaliza categorias
livres existentes antes de criar a constraint.

## Validacoes verificadas

- testes unitarios dos dominios e casos de uso;
- testes dos handlers HTTP;
- teste de integracao PostgreSQL com migracoes em schema isolado;
- fluxo manual contra PostgreSQL real: cadastro, login, publicacao, perfil,
  exclusao de anuncio proprio do catalogo e listagem em "Meus anuncios".

## Correspondencia dos controladores

| Responsabilidade arquitetural | Implementacao |
|---|---|
| Controller de aplicacao (GRASP) | `casosdeuso.Controlador*` |
| Adaptador/controller HTTP | `internal/http.API` e seus handlers |
| Entidades e regras de dominio | `internal/dominio/*` |
| Portas de saida | interfaces em `casosdeuso/contratos.go` |
| Adaptador de persistencia | `database/postgres.Store` |

Os controladores de aplicacao coordenam casos de uso e nao conhecem HTTP ou
PostgreSQL. Os handlers HTTP traduzem o protocolo; o store PostgreSQL implementa
as portas usadas pelos controladores.

## Comportamentos modelados, ainda nao executaveis

- editar e excluir logicamente anuncios pela interface;
- upload real de imagens;
- checkout e criacao de pedidos por vendedor;
- pagamento, reembolso e repasse;
- rastreio e confirmacao de recebimento;
- chat, notificacoes e avaliacoes;
- bloqueio e reativacao de vendedor.

Essas classes ja existem no dominio e no esquema SQL, mas seus casos de uso ainda
precisam ser implementados nas proximas fases.

## Diferencas intencionais em relacao ao diagrama

1. Valores monetarios usam centavos inteiros, nao `float`, para evitar erros de precisao.
2. Dados bancarios completos nao sao persistidos. E armazenado um identificador opaco
   do provedor financeiro.
3. `Anuncio` possui apenas estados comerciais. Envio e recebimento pertencem a
   `ItemPedido` e `Entrega`, evitando misturar catalogo com logistica.
4. Processamento de pagamento, reembolso, storage e frete fica nos submodulos
   funcionais de `casosdeuso` e nas integracoes externas, nao dentro das entidades.
5. `Compra` representa o checkout unico e gera um `Pedido` separado por vendedor,
   conforme os requisitos de dominio.
