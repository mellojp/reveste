# Plano de Implementacao Inicial - ReVeste

## 1. Objetivo deste documento

Este plano consolida os requisitos, casos de uso, descricao arquitetural e diagramas
UML existentes no repositorio. O objetivo e transformar os artefatos conceituais em uma
sequencia inicial de implementacao, deixando explicitas as inconsistencias que precisam
ser resolvidas antes de codificar os fluxos financeiros e logisticos.

Documentos analisados:

- `docs/requisitos.pdf`
- `docs/plano.pdf`
- `docs/casos de uso.pdf`
- `docs/descrição arquitetural.pdf`
- `docs/diagrama de classes nível de projeto.jpeg`
- `docs/diagrama de sequência - publicar anuncio.jpeg`
- `docs/diagrama de sequência - adicionar ao carrinho.jpeg`
- `docs/diagrama de sequência - comprar item.jpeg`

## 2. Sintese do produto

O ReVeste e um marketplace web C2C de pecas de vestuario usadas. Uma mesma conta
pode comprar e vender. O sistema cobre:

1. cadastro, autenticacao e perfil;
2. publicacao e consulta de anuncios;
3. carrinho sem reserva;
4. checkout unico com pedidos separados por vendedor;
5. pagamento, retencao, reembolso e repasse;
6. envio, rastreio e confirmacao de recebimento;
7. chat por pedido;
8. notificacoes;
9. historicos e avaliacoes.

Os atributos de qualidade priorizados sao desempenho, escalabilidade, seguranca,
disponibilidade e manutenibilidade.

## 3. Leitura consolidada dos artefatos

### 3.1 Requisitos

O documento `docs/requisitos.pdf` possui a lista mais extensa, com RF1-RF31 e RD1-RD19.
O arquivo `docs/plano.pdf` parece ser outra revisao da mesma especificacao, com RF1-RF28
e RD1-RD20. O segundo introduz estados de pedido importantes:
`AguardandoPagamento` e `Expirado`, alem da expiracao em uma hora.

Como os identificadores mudam entre as versoes, referencias como "RF10" nao sao
confiaveis sem indicar tambem a versao do documento. Deve existir uma unica matriz de
rastreabilidade versionada no futuro.

### 3.2 Casos de uso

Foram especificados 14 casos de uso:

- UC01 Cadastrar Usuario
- UC02 Autenticar Usuario
- UC03 Gerenciar Anuncios
- UC04 Buscar e Filtrar Anuncios
- UC05 Gerenciar Itens do Carrinho
- UC06 Finalizar Compra
- UC07 Informar Codigo de Rastreio
- UC08 Confirmar Recebimento
- UC09 Processar Reembolso
- UC10 Reativar Item Suspenso
- UC11 Comunicar via Chat
- UC12 Avaliar Vendedor
- UC13 Consultar Pedidos
- UC14 Consultar Historico

Os casos de uso sao uma boa base para testes de aceitacao, mas suas referencias de RF
estao desalinhadas com as duas versoes de requisitos. Alguns fluxos tambem omitem
transacoes, concorrencia, idempotencia e falhas parciais.

### 3.3 Arquitetura

A arquitetura proposta define:

- frontend React como SPA;
- backend Go;
- PostgreSQL 15 ou superior;
- arquitetura em camadas inspirada em Clean Architecture;
- Data Mapper para persistencia;
- API HTTP stateless;
- WebSocket dedicado para chat;
- armazenamento de imagens no Amazon S3;
- Stripe para pagamentos;
- gateway de logistica ainda nao definido;
- SMTP para notificacoes;
- Vercel para frontend/API e Render para conexoes WebSocket;
- Neon como opcao de PostgreSQL gerenciado.

As regras arquiteturais mais importantes sao corretas:

- dominio nao importa infraestrutura;
- controladores chamam casos de uso, nao o banco diretamente;
- integracoes externas ficam atras de interfaces;
- dados completos de cartao nao entram no backend nem no banco;
- listagens sao paginadas;
- operacoes de negocio criticas precisam ser transacionais.

### 3.4 Diagrama de classes

O diagrama representa as entidades:

`Usuario`, `Endereco`, `DadosBancarios`, `PerfilVendedor`, `Anuncio`, `Foto`,
`Carrinho`, `Compra`, `Pedido`, `ItemPedido`, `Pagamento`, `Reembolso`,
`Entrega`, `Conversa`, `Mensagem`, `Notificacao` e `Avaliacao`.

Ele captura os principais substantivos do dominio, mas nao deve ser convertido
literalmente em classes Go ou tabelas. Os seguintes problemas precisam ser corrigidos:

- `Anuncio` acumula atributos e estados do produto, venda e entrega.
- Os diagramas de sequencia usam `Item`, enquanto o diagrama de classes usa `Anuncio`.
- `ItemPedido` tem poucos dados e nao preserva o snapshot da venda.
- `Pagamento` esta ligado a `Compra`, mas o fluxo de reembolso ocorre por item.
- `Entrega` aparece associada de forma ambigua a anuncio, pedido e vendedor.
- CPF, senha e dados bancarios aparecem como strings comuns, sem indicar protecao.
- valores monetarios usam `float`, o que e inadequado para dinheiro.
- metodos como `processarPagamento()` em entidades misturam dominio com integracao.

### 3.5 Diagramas de sequencia

Os tres diagramas sao uteis como narrativa, mas insuficientes como contrato tecnico.

Publicar anuncio:

- cria o item e as fotos antes de explicitar validacao e persistencia atomica;
- nao mostra upload para storage, rollback ou remocao de arquivos orfaos;
- consulta `PerfilVendedor`, embora todo usuario possa vender pela mesma conta;
- nao mostra limite maximo de fotos.

Adicionar ao carrinho:

- valida disponibilidade, mas nao impede adicionar anuncio proprio;
- nao mostra deteccao de item duplicado;
- nao mostra persistencia, autenticacao ou autorizacao;
- corretamente nao reserva o item.

Comprar item:

- processa pagamento antes de demonstrar reserva transacional dos itens;
- cria um unico `Pedido(items)`, apesar da regra de um pedido por vendedor;
- nao mostra calculo de fretes, taxa, expiracao, falha ou webhook;
- envia `alterarStatus(Vendido)` ao carrinho, nao aos itens;
- nao explicita uma transacao unica nem protecao contra duas compras simultaneas.

Esses diagramas devem ser revisados depois da consolidacao do modelo, nao usados como
fonte direta para nomes de metodos.

## 4. Inconsistencias que bloqueiam uma implementacao segura

| Tema | Divergencia | Decisao recomendada |
|---|---|---|
| Fotos por anuncio | Maximo de 5 na arquitetura e 10 em `docs/requisitos.pdf`; `docs/plano.pdf` nao fixa maximo | Adotar 2 a 5 no MVP e atualizar todos os artefatos |
| Exclusao | Exclusao permanente, logica e remocao do catalogo aparecem como equivalentes | Usar exclusao logica (`deleted_at`) para auditoria e LGPD; nunca apagar venda historica |
| Estados do pedido | Uma versao omite `AguardandoPagamento` e `Expirado` | Adotar os 7 estados de `plano.pdf` |
| Momento do pagamento | UC06 sugere processamento imediato, mas tambem espera de ate 1 hora | Modelar checkout assincrono por intencao de pagamento e webhook idempotente |
| Entidade vendida | `Item` e `Anuncio` sao usados de forma intercambiavel | No MVP, tratar cada `Anuncio` como uma peca unica; separar seu estado comercial do estado do item comprado |
| Envio | Codigo aparece por pedido e por item | Um pedido pertence a um vendedor; adotar uma entrega por pedido no MVP |
| Chat | Escopo geral nos requisitos e por pedido nos casos de uso | Adotar conversa por pedido, entre comprador e vendedor |
| Avaliacao | Avalia vendedor por item ou por compra | Adotar uma avaliacao por pedido finalizado, salvo decisao contraria |
| Bloqueio do vendedor | "apos, no maximo, tres" e "ultrapassar tres" | Bloquear ao atingir 3 nao envios confirmados |
| CPF | Documento fala em formato e autenticidade | Validar digitos verificadores; autenticidade real exige servico externo e base legal |
| Endereco | Cadastro exige endereco completo; diagrama permite varios | MVP com um endereco principal e snapshot no pedido |
| Dados bancarios | Sistema armazena dados bancarios | Preferir identificador/token de conta conectada do gateway; nao armazenar dados desnecessarios |
| Status `Indisponivel` | Usado como reserva temporaria e misturado ao ciclo do anuncio | Renomear conceitualmente para `Reservado` ou separar reserva de status |
| Valor monetario | Diagramas usam `float` | Persistir centavos em inteiro (`BIGINT`) ou `NUMERIC`, nunca ponto flutuante |

## 5. Modelo de dominio recomendado

### 5.1 Agregados principais

**Usuario**

- identidade, nome, CPF, email e hash de senha;
- endereco principal;
- estado da conta e contador de nao envios;
- identificador da conta de recebimento no gateway;
- comprador e vendedor sao papeis contextuais, nao subclasses.

**Anuncio**

- pertence a um vendedor;
- representa uma unica peca;
- titulo, descricao, categoria, tamanho, cor, conservacao e preco;
- fotos ordenadas;
- estados comerciais recomendados: `Disponivel`, `Reservado`, `Vendido`,
  `Suspenso`, `Excluido`.

**Carrinho**

- pertence a um comprador;
- contem referencias a anuncios;
- nao reserva estoque;
- impede duplicata e, por regra recomendada, anuncio do proprio usuario.

**Compra**

- representa a tentativa unica de checkout do comprador;
- consolida subtotal, fretes, taxa e total;
- possui uma referencia de pagamento;
- agrupa um ou mais pedidos, um por vendedor.

**Pedido**

- pertence a uma compra, comprador e vendedor;
- guarda snapshots do endereco, precos e dados relevantes;
- estados: `Criado`, `AguardandoPagamento`, `Cancelado`, `Expirado`,
  `AguardandoEnvio`, `AguardandoEntrega`, `Finalizado`;
- possui itens, entrega e conversa.

**ItemPedido**

- aponta para o anuncio original, mas preserva titulo, preco, taxa e demais dados da
  venda;
- estados logisticos: `AguardandoEnvio`, `Enviado`, `NaoEnviado`, `Recebido`,
  `Suspenso`;
- suporta reembolso e repasse individual.

**Pagamento**

- guarda somente identificadores do gateway, valores, estado e chave de idempotencia;
- estados: `Pendente`, `Aprovado`, `Recusado`, `ReembolsadoParcial`,
  `Reembolsado`;
- atualizacoes confirmadas por webhook assinado.

**Entrega**

- uma por pedido no MVP;
- frete cotado, transportadora, codigo de rastreio e datas;
- o codigo deve ser validado antes de alterar os itens para `Enviado`.

**Conversa**

- uma por pedido;
- mensagens persistidas e paginadas;
- WebSocket distribui eventos, mas o banco continua sendo a fonte da verdade.

### 5.2 Invariantes essenciais

1. Um anuncio so pode ser reservado se estiver `Disponivel`.
2. A reserva de todos os anuncios do checkout ocorre na mesma transacao do banco.
3. Duas compras concorrentes nao podem reservar o mesmo anuncio.
4. Um pedido contem itens de exatamente um vendedor.
5. O total da compra e a soma de itens, fretes e taxa, calculada no servidor.
6. O cliente nunca informa valores finais confiaveis.
7. Webhooks e comandos financeiros sao idempotentes.
8. Falha externa nao pode deixar pagamento aprovado com itens novamente disponiveis.
9. Historico financeiro e de pedidos nao e apagado por exclusao de anuncio ou conta.
10. Toda transicao de estado e validada por uma maquina de estados explicita.

## 6. Escopo recomendado

### 6.1 MVP demonstravel

O primeiro incremento deve provar o ciclo central sem depender de dinheiro real:

- cadastro e login;
- perfil e endereco principal;
- criar, editar, excluir logicamente e listar anuncios;
- upload de 2 a 5 imagens por anuncio;
- busca e filtros paginados;
- carrinho;
- checkout transacional agrupado por vendedor;
- pagamento simulado por uma interface de gateway;
- consulta de compras, vendas e pedidos;
- atualizacao manual/simulada de envio e recebimento;
- testes das transicoes de estado.

Esse recorte entrega o valor central do marketplace e permite validar o modelo antes de
integrar servicos externos de alta complexidade.

### 6.2 Segundo incremento

- integracao real com storage S3 compativel;
- gateway de frete;
- gateway de pagamento em ambiente de testes;
- webhooks, reembolso e repasse;
- jobs de expiracao em 1 hora, nao envio em 7 dias e recebimento em 15 dias;
- notificacoes persistidas e email.

### 6.3 Terceiro incremento

- chat em tempo real;
- avaliacoes;
- reativacao paga de item suspenso;
- bloqueio automatico de vendedor;
- termos, privacidade, exportacao e exclusao de dados;
- observabilidade, testes de carga e endurecimento de seguranca.

## 7. Arquitetura de implementacao

### 7.1 Organizacao sugerida do repositorio

```text
/
|-- apps/
|   |-- web/                  # React + TypeScript
|   `-- api/                  # Go
|       |-- cmd/api/
|       `-- internal/
|           |-- dominio/
|           |   |-- cadastros/
|           |   |-- anuncios/
|           |   |-- compras/
|           |   |-- interacao/
|           |   `-- erros/
|           |-- casosdeuso/
|           |   |-- cadastros/
|           |   |-- anuncios/
|           |   `-- compras/
|           |-- database/
|           |-- http/
|           `-- common/
|-- contracts/
|   `-- openapi.yaml
|-- db/
|   |-- migrations/
|   `-- seeds/
|-- docs/
|   |-- adr/
|   |-- requirements/
|   `-- diagrams/
|-- compose.yaml
|-- Makefile
`-- README.md
```

No backend:

- `dominio/cadastros`: conta, endereco, perfil vendedor e dados bancarios;
- `dominio/anuncios`: anuncio, foto, conservacao e estados de catalogo;
- `dominio/compras`: carrinho, compra, pedido, pagamento, reembolso e entrega;
- `dominio/interacao`: conversa, mensagem, notificacao e avaliacao;
- `dominio/erros`: erros compartilhados entre os modulos;
- `casosdeuso/cadastros`: cadastro, autenticacao e gerenciamento de sessao;
- `casosdeuso/anuncios`: publicacao e consulta do catalogo;
- `casosdeuso/compras`: carrinho e, futuramente, checkout e pedidos;
- `casosdeuso/contratos.go`: contratos externos compartilhados pelo modulo;
- `database`: implementacao de persistencia PostgreSQL;
- `http`: transporte separado em arquivos por cadastros, anuncios, carrinho,
  autenticacao, respostas e middlewares;
- `common`: leitura de ambiente, `.env`, seguranca, IDs e tempo.

Esta divisao deve ser aplicada pragmaticamente. Nao e necessario criar uma interface
para cada funcao; interfaces devem existir nos limites externos e nos pontos que exigem
substituicao em testes.

### 7.2 Contratos

Definir OpenAPI antes das telas principais reduz divergencia entre Go e TypeScript.
Os erros devem ter formato estavel, por exemplo:

```json
{
  "code": "AD_NOT_AVAILABLE",
  "message": "O anuncio nao esta mais disponivel.",
  "fields": {}
}
```

IDs devem ser opacos. Datas devem trafegar em ISO 8601 e valores monetarios em
centavos.

### 7.3 Persistencia inicial

Tabelas minimas:

- `users`
- `addresses`
- `seller_accounts`
- `ads`
- `ad_photos`
- `carts`
- `cart_items`
- `purchases`
- `orders`
- `order_items`
- `payments`
- `refunds`
- `shipments`
- `conversations`
- `messages`
- `notifications`
- `reviews`
- `outbox_events`

Indices iniciais:

- unicidade em email e CPF normalizados;
- anuncios por status e data;
- filtros por categoria, tamanho, conservacao e preco;
- pedidos por comprador, vendedor e status;
- mensagens por conversa e data;
- IDs externos e chaves de idempotencia unicos.

### 7.4 Consistencia e eventos

O checkout e o ponto de maior risco. A sequencia recomendada e:

1. iniciar transacao;
2. reler e bloquear os anuncios selecionados;
3. rejeitar os indisponiveis;
4. calcular valores no servidor;
5. criar compra e pedidos por vendedor;
6. marcar anuncios como reservados;
7. registrar evento de solicitacao de pagamento na outbox;
8. confirmar transacao;
9. criar/processar a intencao no gateway;
10. receber webhook idempotente;
11. aprovar e marcar anuncios vendidos, ou expirar e liberar reservas.

O padrao outbox evita perder eventos entre o commit do PostgreSQL e chamadas externas.
Para o MVP simulado, a mesma interface deve ser mantida, mesmo que o processamento
ocorra no proprio processo.

### 7.5 Autenticacao e seguranca

- hash de senha com algoritmo apropriado e custo configuravel;
- sessao preferencialmente em cookie `HttpOnly`, `Secure` e `SameSite`;
- protecao CSRF caso cookies autentiquem operacoes mutaveis;
- autorizacao por recurso em todos os casos de uso;
- rate limiting em login, cadastro, upload e chat;
- validacao de tipo real, tamanho e dimensoes de imagem;
- URLs assinadas para upload/download quando aplicavel;
- criptografia ou tokenizacao de CPF e dados sensiveis conforme necessidade;
- logs sem senha, token, CPF completo, endereco ou payload financeiro;
- trilha de auditoria para transicoes financeiras e de pedido.

Armazenar JWT diretamente em `localStorage`, como sugerido na descricao arquitetural,
aumenta o impacto de XSS. Essa decisao deve ser revista antes da implementacao.

## 8. API inicial sugerida

### Conta

- `POST /v1/users`
- `POST /v1/sessions`
- `DELETE /v1/sessions/current`
- `GET /v1/me`
- `PATCH /v1/me`
- `PUT /v1/me/address`

### Anuncios

- `POST /v1/ads`
- `GET /v1/ads`
- `GET /v1/ads/{adId}`
- `PATCH /v1/ads/{adId}`
- `DELETE /v1/ads/{adId}`
- `POST /v1/ads/{adId}/photos`

### Carrinho e checkout

- `GET /v1/cart`
- `POST /v1/cart/items`
- `DELETE /v1/cart/items/{adId}`
- `POST /v1/checkouts`
- `GET /v1/purchases/{purchaseId}`

### Pedidos

- `GET /v1/orders?role=buyer|seller&status=...`
- `GET /v1/orders/{orderId}`
- `POST /v1/orders/{orderId}/shipment`
- `POST /v1/orders/{orderId}/receipt-confirmation`

### Integracoes

- `POST /v1/webhooks/payments`
- `POST /v1/webhooks/shipping`

Chat, reembolso, reativacao e avaliacao entram nos incrementos posteriores.

## 9. Plano de execucao por fases

### Fase 0 - Consolidacao e fundacao

Entregas:

- decidir as divergencias da secao 4;
- criar glossario canonico;
- definir maquinas de estado de anuncio, pedido, item e pagamento;
- criar ADRs para autenticacao, deploy, pagamento e jobs;
- criar monorepo, linters, formatadores e pipeline de CI;
- subir PostgreSQL local e primeira migracao;
- publicar contrato OpenAPI inicial.

Criterio de saida: projeto compila, testes executam no CI e o modelo canonico foi
aprovado pela equipe.

### Fase 1 - Conta e autenticacao

Entregas:

- cadastro com unicidade e validacao de CPF;
- login e logout;
- perfil autenticado e endereco;
- autorizacao basica;
- testes unitarios e de integracao.

Criterio de saida: usuario cria conta, autentica e acessa somente os proprios dados.

### Fase 2 - Anuncios e catalogo

Entregas:

- CRUD de anuncio;
- upload de imagens;
- regras de 2 a 5 fotos e 5 MB;
- catalogo paginado;
- busca e filtros;
- painel de anuncios do vendedor.

Criterio de saida: um usuario publica uma peca e outro a encontra no catalogo.

### Fase 3 - Carrinho e checkout simulado

Entregas:

- carrinho persistente;
- validacao de disponibilidade;
- agrupamento por vendedor;
- reserva concorrente no PostgreSQL;
- compra, pedidos e itens com snapshots;
- gateway de pagamento falso;
- expiracao de reservas.

Criterio de saida: duas requisicoes concorrentes nao conseguem comprar a mesma peca,
e uma compra com vendedores diferentes gera pedidos separados.

### Fase 4 - Pedidos e logistica simulada

Entregas:

- paineis de compra e venda;
- transicoes de envio e recebimento;
- historico;
- jobs de 7 e 15 dias usando relogio injetavel;
- reembolso e repasse simulados;
- notificacoes internas.

Criterio de saida: o fluxo completo da venda chega a finalizado e os fluxos de
nao envio chegam a suspenso/cancelado.

### Fase 5 - Integracoes externas

Entregas:

- spike tecnico de compatibilidade de deploy;
- storage real;
- frete real;
- pagamento em sandbox;
- verificacao de assinatura de webhook;
- idempotencia, retry e reconciliacao;
- email.

Criterio de saida: testes de contrato e cenarios de falha demonstram recuperacao sem
duplicar cobranca, reembolso ou repasse.

### Fase 6 - Chat, avaliacao e endurecimento

Entregas:

- historico de mensagens via HTTP;
- entrega em tempo real via WebSocket;
- reconexao sem duplicacao;
- avaliacao;
- reativacao de item e bloqueio de vendedor;
- testes E2E, carga e seguranca;
- observabilidade e documentacao operacional.

Criterio de saida: requisitos restantes rastreados para testes e operacao demonstravel.

## 10. Estrategia de testes

### Testes unitarios

Priorizar regras puras:

- transicoes de estado;
- calculo de subtotal, frete, taxa, reembolso e repasse;
- agrupamento por vendedor;
- bloqueio por nao envio;
- validacao de permissao;
- expiracao por relogio injetavel.

### Testes de integracao

- operacoes PostgreSQL reais;
- constraints e migracoes;
- bloqueio concorrente no checkout;
- handlers HTTP;
- outbox e idempotencia;
- implementacoes falsas dos gateways.

### Testes de contrato

- OpenAPI contra backend;
- webhooks e gateways externos;
- schemas de eventos do chat.

### Testes E2E

Fluxos prioritarios:

1. cadastrar, autenticar e publicar;
2. buscar, adicionar ao carrinho e comprar;
3. comprar itens de dois vendedores;
4. conflito de compra da mesma peca;
5. pagamento recusado ou expirado;
6. envio e recebimento;
7. nao envio, reembolso e suspensao.

### Testes nao funcionais

- carga no feed e busca;
- latencia do chat;
- limites de upload;
- verificacao de autorizacao horizontal;
- varredura de dependencias e segredos;
- recuperacao diante de timeout dos gateways.

## 11. Backlog inicial priorizado

| Prioridade | Epic | Dependencia |
|---|---|---|
| P0 | Modelo canonico e maquinas de estado | Nenhuma |
| P0 | Fundacao do repositorio e CI | Modelo minimo |
| P0 | Persistencia e migracoes | Fundacao |
| P0 | Conta, sessao e autorizacao | Persistencia |
| P0 | Anuncios e catalogo | Conta |
| P0 | Carrinho | Anuncios |
| P0 | Checkout transacional simulado | Carrinho |
| P0 | Pedidos e historico | Checkout |
| P1 | Storage externo | Anuncios |
| P1 | Pagamento, webhook e outbox | Checkout |
| P1 | Frete e rastreio | Pedidos |
| P1 | Jobs e notificacoes | Pedidos/pagamento |
| P2 | Chat WebSocket | Pedidos |
| P2 | Avaliacoes | Pedido finalizado |
| P2 | Penalidades e reativacao | Reembolso |
| P2 | LGPD operacional e observabilidade | Fundacao completa |

## 12. Divisao sugerida para seis integrantes

Depois da Fase 0, a equipe pode trabalhar em tres trilhas com revisao cruzada:

- Trilha A: frontend, design system e fluxos de conta/catalogo;
- Trilha B: dominio, casos de uso e API;
- Trilha C: PostgreSQL, integracoes, CI e observabilidade.

Cada trilha pode ter duas pessoas, mas a propriedade de codigo nao deve ser exclusiva.
Checkout, autenticacao e migracoes devem exigir revisao de alguem de outra trilha.

## 13. Decisoes pendentes antes da Fase 3

1. Qual documento passa a ser a fonte oficial de requisitos?
2. O limite final de fotos e 5 ou 10?
3. A exclusao de anuncio e apenas logica?
4. O usuario pode adicionar o proprio anuncio ao carrinho?
5. A avaliacao ocorre por item ou por pedido?
6. Um pedido usa exatamente um codigo de rastreio?
7. Qual gateway de frete sera usado e ele realmente valida CEP de destino por rastreio?
8. Qual modelo do gateway de pagamento atende marketplace, repasse e reembolso parcial?
9. "Reter pagamento" significa captura tardia, saldo de marketplace ou outra operacao?
10. Qual e o percentual de taxa e qual e a taxa de reativacao?
11. Quem paga e como e tratado o frete em reembolso parcial?
12. Como ocorre contestacao quando o comprador nao reconhece o recebimento?
13. Qual politica de exclusao, retencao e exportacao de dados atende ao projeto?
14. Onde os jobs temporais serao executados?
15. A combinacao Go, funcoes serverless e conexao PostgreSQL atende aos limites reais
    do ambiente escolhido? Isso deve ser provado por um spike, nao assumido.

## 14. Primeira milestone recomendada

**Milestone: "Da conta ao carrinho"**

Escopo:

- fundacao do repositorio;
- cadastro, login e endereco;
- publicacao e listagem de anuncios;
- imagens usando adaptador local no desenvolvimento;
- filtros;
- carrinho persistente sem reserva;
- OpenAPI e testes dos fluxos.

Essa milestone evita iniciar pelo trecho mais arriscado antes de validar a estrutura,
mas ja produz uma demonstracao vertical navegavel. Em paralelo, uma dupla deve
prototipar o checkout concorrente e a integracao de pagamento em sandbox para reduzir
os riscos da milestone seguinte.
