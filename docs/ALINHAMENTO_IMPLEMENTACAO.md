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

O `index.html` contem somente o shell da aplicacao. As telas ficam em
`apps/front/js/pages`, componentes reutilizaveis em `apps/front/js/components`,
estado, chamadas HTTP e roteamento em `apps/front/js/core`, e os estilos sao
separados por responsabilidade em `apps/front/css`.

Telas e fluxos disponiveis:

- landing page em `/` e catalogo responsivo em `/catalogo`;
- detalhe publico de anuncio em `/anuncios/:id`, com galeria e inclusao na sacola;
- busca por texto, categoria e estado de conservacao;
- cadastro em `/cadastro`, com mensagens de validacao junto aos campos;
- login em `/entrar`, logout e sessao mantida em `sessionStorage`;
- publicacao em `/vender`, com upload de 2 a 5 fotos;
- perfil em `/perfil`, com dados pessoais e endereco;
- painel do vendedor em `/meus-anuncios`;
- carrinho em `/carrinho`, com inclusao e remocao de pecas.

O frontend usa a History API para navegacao sem recarregamento. O servidor
estatico entrega `index.html` como fallback para rotas de tela, permitindo abrir
ou atualizar diretamente URLs como `/perfil`. Rotas autenticadas redirecionam
para `/entrar` e preservam o destino para retorno depois do login.

Checkout, edicao de perfil e edicao ou exclusao de anuncios nao sao simulados na
interface: as telas indicam essas limitacoes ate os respectivos contratos HTTP
serem implementados.

O uso de `sessionStorage` e provisório e acompanha o contrato Bearer atual. A decisao
de autenticacao por cookie `HttpOnly`, protecao CSRF e deploy continua pendente antes
de producao.

## Contratos HTTP adicionados

- `GET /v1/me`: retorna o usuario da sessao atual;
- `GET /v1/anuncios/{idAnuncio}`: retorna os detalhes publicos de um anuncio;
- `GET /v1/me/anuncios`: retorna os anuncios publicados pelo usuario;
- `GET /v1/anuncios`: quando recebe um Bearer valido, omite anuncios do proprio
  usuario; sem autenticacao, continua publico;
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
sao verificados. Validacao binaria, remocao de EXIF, moderacao, rate limiting e coleta
de orfaos permanecem no backlog de seguranca do upload.

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

- editar e excluir logicamente anuncios pela interface;
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
