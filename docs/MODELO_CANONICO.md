# Modelo Canonico do MVP

Este documento registra as correcoes aplicadas antes da implementacao.

## Decisoes consolidadas

- Cada `anuncio` representa uma unica peca de roupa.
- O limite do MVP e de 2 a 5 fotos por anuncio, com ate 5 MB por arquivo.
- A exclusao de anuncio e logica. Dados relacionados a vendas permanecem para auditoria.
- Valores monetarios sao armazenados em centavos inteiros.
- Comprador e vendedor sao papeis exercidos pelo mesmo `usuario`.
- Um carrinho nao reserva anuncios.
- Um usuario nao pode adicionar o proprio anuncio ao carrinho.
- A reserva ocorre apenas no checkout, dentro de transacao no PostgreSQL.
- Uma `compra` representa o checkout unico; ela gera um `pedido` por vendedor.
- `item_pedido` preserva um snapshot dos dados da peca no momento da compra.
- Uma `entrega` pertence a um pedido, portanto existe um codigo de rastreio por vendedor.
- Uma `conversa` pertence a um pedido.
- Uma `avaliacao` e registrada por pedido finalizado.
- O vendedor e bloqueado ao atingir 3 itens nao enviados confirmados.
- Dados completos de cartao ou conta bancaria nao sao armazenados. O sistema guarda
  apenas identificadores opacos do provedor financeiro.

## Estados

### Anuncio

`disponivel -> reservado -> vendido`

Transicoes adicionais:

- `disponivel -> excluido`
- `suspenso -> disponivel`, apos pagamento da taxa de reativacao
- `reservado -> disponivel`, quando pagamento falha ou expira

### Compra

`aguardando_pagamento -> aprovada | recusada | expirada | cancelada`

### Pedido

`criado -> aguardando_pagamento -> aguardando_envio -> aguardando_entrega -> finalizado`

Saidas excepcionais: `cancelado` e `expirado`.

### Item do pedido

`aguardando_envio -> enviado -> recebido`

Fluxo de falha:

`aguardando_envio -> nao_enviado -> suspenso`

### Pagamento

`pendente -> aprovado | recusado`

Depois de aprovado:

`aprovado -> reembolsado_parcial -> reembolsado`

## Regra de precedencia

Enquanto os PDFs originais nao forem atualizados, este documento e
`PLANO_IMPLEMENTACAO_INICIAL.md` prevalecem para a implementacao do MVP quando houver
divergencia de nomenclatura ou comportamento.
