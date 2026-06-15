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
- editar dados pessoais e endereco principal;
- editar e excluir logicamente anuncios disponiveis do proprio vendedor;
- consultar dados publicos e anuncios disponiveis de outros vendedores;
- adicionar e remover anuncio do carrinho;
- validar CPF, anuncio, quantidade de fotos, categoria e disponibilidade;
- apresentar um fluxo web navegavel de conta, catalogo, publicacao, perfil e carrinho.

## Frontend SSR com HTMX

O frontend continua sem etapa de build e e servido pela propria API. O pacote
`internal/web` e o adaptador de paginas, separado do adaptador JSON: consultas
GET e comandos POST chamam os mesmos controladores de casos de uso, preenchem um
`contextoDocumento` e respondem templates Go com escape contextual de HTML.

Os templates ficam em `apps/api/internal/web/templates`, os estilos e assets em
`apps/front`, e o HTMX 2.0.8 e versionado localmente em
`apps/front/js/htmx.min.js`. Navegacao, filtros, autenticacao, perfil, anuncios e
carrinho usam HTML do servidor. O modulo `apps/front/js/web.js` cobre somente
galeria, controles de fotos, confirmacao de exclusao e feedback visual; o
processamento seguro e upload de imagens fica em `js/uploads.js`.

Telas e fluxos disponiveis:

- landing page em `/` e catalogo responsivo em `/catalogo`;
- detalhe publico de anuncio em `/anuncios/:id`, com galeria e inclusao na sacola;
- busca por texto, categoria, tamanho, faixa de preco e estado de conservacao;
- carregamento progressivo do catalogo em paginas de 24 anuncios;
- cadastro em `/cadastro`, com mensagens de validacao junto aos campos;
- login em `/entrar`, logout e sessao mantida em cookie `HttpOnly`;
- publicacao em `/vender`, com upload de 2 a 5 fotos;
- perfil em `/perfil`, com dados pessoais e endereco;
- edicao de dados pessoais e endereco em `/perfil`;
- painel do vendedor em `/meus-anuncios`, com edicao e exclusao logica;
- perfil publico de vendedor em `/vendedores/:id`;
- carrinho em `/carrinho`, com inclusao e remocao de pecas.

As alteracoes deste incremento, incluindo contratos, regras, decisoes de
frontend, verificacoes e limitacoes, estao detalhadas em
`docs/INCREMENTO_FLUXO_INICIAL.md`.

O carrinho nao reserva estoque. Anuncios que se tornam indisponiveis continuam visiveis
para que o usuario entenda a alteracao, mas deixam de compor o total.

Cada rota de tela e atendida diretamente pelo backend e funciona com navegacao
HTML convencional. O `hx-boost` melhora progressivamente a navegacao e os
formularios, sem alterar URLs ou depender de um estado global no navegador.
Rotas autenticadas redirecionam para `/entrar` e preservam o destino.

Checkout ainda nao e simulado na interface. Edicao de perfil, edicao de anuncios
disponiveis e exclusao logica ja possuem contratos HTTP e fluxos web.

O navegador autentica por cookie `HttpOnly`, `SameSite=Lax` e `Secure` em HTTPS.
Todos os formularios web mutaveis e operacoes autenticadas por cookie exigem uma
origem da propria aplicacao, reduzindo o risco de CSRF. O token de sessao nao e
exposto ao JavaScript nem persistido em Web Storage. Clientes de API podem solicitar explicitamente a resposta Bearer com
`X-Reveste-Session-Transport: bearer`.

## Contratos HTTP adicionados

- `GET /v1/me`: retorna o usuario da sessao atual;
- `GET /v1/anuncios/{idAnuncio}`: retorna os detalhes publicos de um anuncio;
- `GET /v1/me/anuncios`: retorna os anuncios publicados pelo usuario;
- `PATCH /v1/me`: atualiza dados pessoais e endereco principal;
- `PATCH /v1/me/anuncios/{idAnuncio}`: edita anuncio disponivel do usuario;
- `DELETE /v1/me/anuncios/{idAnuncio}`: exclui logicamente anuncio disponivel;
- `GET /v1/vendedores/{idVendedor}`: retorna perfil publico e anuncios disponiveis;
- `GET /v1/anuncios`: quando recebe um Bearer valido, omite anuncios do proprio
  usuario; sem autenticacao, continua publico;
- `GET /saude/prontidao`: verifica a conexao PostgreSQL com timeout;
- erros de cadastro podem preencher `campos` com mensagens especificas por input.

## Armazenamento de imagens

Os arquivos de imagem sao armazenados em um Vercel Blob store configurado com
acesso publico. O PostgreSQL persiste apenas as URLs publicas e a ordem das fotos
em `foto_anuncio`.

O fluxo adotado e:

1. o frontend seleciona e valida de 2 a 5 imagens JPEG, PNG ou WebP;
2. para cada arquivo, solicita autorizacao em
   `POST /v1/uploads/imagens/autorizacoes`;
3. a API autentica o usuario e emite um token Vercel Blob restrito ao pathname,
   tipos permitidos, limite de 5 MB e validade de 10 minutos;
4. o navegador envia o arquivo diretamente ao Vercel Blob;
5. as URLs retornadas sao enviadas na criacao do anuncio.

Os adaptadores ficam em `internal/storage/vercel` e
`internal/storage/postgres`: o primeiro persiste objetos externos; o segundo
persiste dados relacionais. A porta `ArmazenamentoArquivos` mantem o caso de uso
independente do provedor e permite trocar Vercel Blob por S3 no futuro.

Ainda falta implementar limpeza de imagens orfas quando o upload conclui, mas a
criacao do anuncio falha ou e abandonada.

### Politica de acesso aos arquivos

Fotos de anuncios usam um Blob store publico porque o proprio catalogo e publico.
Nesse contexto, "publico" significa que qualquer pessoa que possua a URL consegue
ler o arquivo. Isso nao torna a listagem ou descoberta de objetos automatica, mas
a URL nao deve ser tratada como segredo.

O store publico deve conter somente midia destinada a exposicao no catalogo.
Documentos, comprovantes, anexos de conversas e qualquer dado pessoal ou sensivel
devem usar um store privado separado, com URLs assinadas de curta duracao ou entrega
autorizada pela API.

Controles obrigatorios para as fotos publicas:

- pathnames gerados pelo servidor com identificadores imprevisiveis;
- autorizacao de upload vinculada ao usuario autenticado;
- token temporario restrito ao pathname, tipo e tamanho;
- limite de 2 a 5 imagens e 5 MB por arquivo;
- tipos permitidos: JPEG, PNG e WebP;
- validacao do conteudo real do arquivo, nao apenas do MIME informado pelo navegador;
- remocao de metadados EXIF, principalmente coordenadas GPS;
- moderacao de conteudo e limites de requisicao;
- remocao de imagens orfas e das imagens de anuncios excluidos quando permitido
  pelas regras de auditoria.

No estado atual, pathname, autenticacao, token temporario, quantidade, tamanho e MIME
sao verificados. Antes do upload, o navegador valida a assinatura binaria, decodifica
e reencoda a imagem como WebP, removendo EXIF, GPS, animacoes e conteudo extra. A API
aceita somente URLs do hostname exato do Blob store configurado, e a CSP aplica a mesma
restricao. Moderacao, rate limiting e coleta de orfaos permanecem no backlog.

Essa reencodificacao melhora privacidade e robustez, mas JavaScript nao e uma fronteira
de seguranca contra clientes modificados. Uma validacao autoritativa de bytes exigiria
processamento no backend ou uma etapa confiavel pos-upload.

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
- testes de autorizacao para edicao e exclusao de anuncios;
- testes de privacidade do perfil publico do vendedor;
- testes de carrinho com itens indisponiveis;
- testes do endpoint de prontidao e dos headers de seguranca;
- teste de integracao PostgreSQL com migracoes em schema isolado;
- fluxo manual contra PostgreSQL real: cadastro, login, publicacao, perfil,
  exclusao de anuncio proprio do catalogo e listagem em "Meus anuncios".

Os testes unitarios ficam junto aos pacotes. Testes HTTP e PostgreSQL que exercitam
adaptadores completos ficam centralizados em `apps/api/tests/integration`. Futuros
testes de navegador e fluxos completos devem usar `apps/api/tests/e2e`.

## Correspondencia dos controladores

| Responsabilidade arquitetural | Implementacao |
|---|---|
| Controller de aplicacao (GRASP) | `casosdeuso.Controlador*` |
| Adaptador/controller HTTP | `internal/http.API` e seus handlers |
| Entidades e regras de dominio | `internal/dominio/*` |
| Portas de saida | interfaces em `casosdeuso/contratos.go` |
| Adaptadores de persistencia | `storage/postgres.Store` e `storage/vercel.Storage` |

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
