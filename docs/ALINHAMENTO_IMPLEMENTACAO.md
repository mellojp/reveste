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
- criar anuncio;
- listar e filtrar anuncios;
- adicionar e remover anuncio do carrinho;
- validar CPF, anuncio, quantidade de fotos e disponibilidade.

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
