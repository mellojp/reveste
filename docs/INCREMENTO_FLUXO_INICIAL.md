# Incremento do fluxo inicial

Este documento registra as alteracoes implementadas na conclusao do fluxo inicial
da ReVeste. O incremento cobre correcoes estruturais no backend, gerenciamento de
perfil e anuncios, exposicao segura do perfil publico de vendedores, refinamentos
de navegacao e uma camada consistente de feedback visual no frontend.

## Resumo funcional

O usuario autenticado agora pode:

- editar seus dados pessoais e endereco principal;
- visualizar, editar e excluir logicamente seus anuncios;
- manter, remover e reordenar fotos durante a criacao ou edicao de um anuncio;
- consultar a sacola mesmo quando um item se torna indisponivel;
- receber feedback visual durante operacoes assincronas.

Visitantes e compradores agora podem:

- visualizar os dados publicos do vendedor no detalhe do anuncio;
- acessar uma pagina publica com o perfil e os anuncios disponiveis do vendedor;
- filtrar o catalogo por texto, categoria, conservacao, tamanho e faixa de preco;
- carregar resultados adicionais sem substituir os itens ja exibidos.

## Backend

### Novos contratos HTTP

| Metodo e rota | Autenticacao | Comportamento |
|---|---|---|
| `PATCH /v1/me` | obrigatoria | Atualiza nome, e-mail, telefone e endereco principal |
| `PATCH /v1/me/anuncios/{id}` | obrigatoria | Atualiza um anuncio disponivel pertencente ao usuario |
| `DELETE /v1/me/anuncios/{id}` | obrigatoria | Faz exclusao logica de um anuncio disponivel |
| `GET /v1/vendedores/{id}` | publica | Retorna dados publicos e anuncios disponiveis do vendedor |
| `GET /saude/prontidao` | publica | Verifica a disponibilidade da conexao PostgreSQL |

O retorno de `GET /v1/anuncios/{id}` passou a incluir o objeto `vendedor` com:

- identificador;
- nome;
- cidade e estado;
- data de entrada na plataforma.

E-mail, telefone, CPF, logradouro, numero e demais dados privados nao fazem parte
desse contrato publico.

### Perfil

`ControladorCadastro.AtualizarPerfil` carrega o usuario atual, altera apenas os
campos editaveis, normaliza os valores e executa novamente as validacoes de
dominio. CPF, hash de senha, estado da conta e demais dados internos sao
preservados.

A persistencia de usuario e endereco ocorre em uma unica transacao PostgreSQL.
Conflitos de e-mail continuam sendo traduzidos pelo adaptador de banco para o
erro de aplicacao correspondente.

### Gerenciamento de anuncios

O dominio ganhou a regra `PodeSerGerenciadoPor`, que exige:

1. que o usuario autenticado seja o proprietario do anuncio;
2. que o anuncio esteja no estado `disponivel`.

A edicao atualiza os dados comerciais, recria a ordenacao das fotos e valida o
anuncio completo antes de persistir. A operacao PostgreSQL e transacional: dados
do anuncio e fotos sao confirmados ou revertidos juntos.

A exclusao e logica. O registro passa para o estado `excluido`, recebe
`excluido_em` e deixa de aparecer em consultas de catalogo, perfil publico e
painel do vendedor.

Um vendedor com `bloqueado_para_vendas` nao pode publicar novos anuncios. Essa
regra e aplicada no caso de uso, antes da criacao da entidade.

### Validacao e autorizacao

- URLs de fotos devem ser URLs HTTPS absolutas e validas.
- Tentativas de editar ou excluir anuncios de terceiros retornam `403`.
- Tentativas de gerenciar anuncios fora do estado permitido retornam conflito de
  transicao.
- O erro de vendedor bloqueado tambem e traduzido para `403`.
- A API diferencia operacao nao autorizada, nao permitida, conflito e recurso
  indisponivel.

### Carrinho

O detalhamento do carrinho deixou de buscar cada anuncio individualmente. Os
itens sao carregados em lote por seus identificadores, preservando a ordem do
carrinho e removendo uma consulta por item.

Anuncios vendidos ou indisponiveis permanecem na resposta para explicar a
mudanca ao usuario, mas nao sao somados em `total_centavos`. Anuncios excluidos
logicamente nao sao retornados.

### Persistencia e desempenho

- as fotos de uma lista de anuncios sao carregadas em uma unica consulta;
- o filtro `IDsAnuncios` permite obter os itens do carrinho em lote;
- a consulta de listagem elimina registros com `excluido_em`;
- atualizacao de perfil, atualizacao de anuncio e substituicao de fotos usam
  transacoes;
- `Store.Ping` implementa a porta usada pela verificacao de prontidao.

Essas mudancas removem os principais cenarios N+1 do catalogo e do carrinho.

### Operacao e seguranca HTTP

As respostas passaram a incluir:

- `Content-Security-Policy`;
- `Referrer-Policy`;
- `X-Content-Type-Options`;
- `X-Frame-Options`;
- `Permissions-Policy`.

Arquivos estaticos usam `no-cache`, permitindo revalidacao pelo navegador. O HTML
e renderizado por rota no servidor e nao depende mais de um shell SPA.

O limitador de tentativas de login remove entradas expiradas quando o mapa
interno cresce, reduzindo o risco de crescimento indefinido em processos longos.

`GET /saude` permanece como liveness. `GET /saude/prontidao` usa timeout de dois
segundos e retorna `503` quando o banco nao esta disponivel.

## Frontend

### Rotas e navegacao SSR

Foram adicionadas as rotas:

- `/meus-anuncios/:id/editar`, autenticada;
- `/vendedores/:id`, publica.

As rotas de tela sao registradas no adaptador `internal/web`, que aplica
autenticacao antes de renderizar paginas privadas. O HTMX melhora
progressivamente navegacao, formularios e carregamento adicional do catalogo,
mantendo a navegacao HTML convencional como base.

O cabecalho indica a secao atual com `aria-current="page"` e fornece feedback
visual imediato ao clicar em links.

### Perfil do usuario

A pagina `/perfil` possui modos de visualizacao e edicao. O formulario permite
alterar dados pessoais e endereco, mostra erros junto aos campos e atualiza o
HTML completo da pagina depois do salvamento.

### Painel e edicao de anuncios

O painel `/meus-anuncios` apresenta acoes separadas para visualizar, editar e
excluir. A exclusao exige um segundo clique de confirmacao, com expiracao em
cinco segundos.

A pagina de edicao:

- valida propriedade e disponibilidade antes de exibir o formulario;
- reutiliza os campos e regras da publicacao;
- permite manter fotos existentes e adicionar novas;
- permite remover e reordenar as imagens;
- exige entre duas e cinco imagens JPEG, PNG ou WebP de ate 5 MB;
- envia apenas novas imagens ao Blob e reutiliza as URLs existentes;
- redireciona para o detalhe do anuncio apos salvar.

O upload direto para o Vercel Blob fica em `js/uploads.js`, evitando
duplicacao entre publicacao e edicao.

### Vendedores

O detalhe do anuncio exibe nome, iniciais, localidade e ano de entrada do
vendedor. Para anuncios de terceiros, ha um link para `/vendedores/:id`.

A pagina publica do vendedor mostra apenas dados publicos e os anuncios
disponiveis. A inclusao na sacola continua disponivel nessa pagina e exige login.

### Catalogo

O catalogo passou a oferecer:

- filtro por tamanho;
- preco minimo e maximo, convertidos para centavos;
- painel de filtros adaptado para telas menores;
- paginacao incremental de 24 itens;
- preservacao do scroll ao aplicar ou limpar filtros;
- estado vazio com acao clara;
- indicacao de que a ordenacao usa os anuncios mais recentes.

Os cards reconhecem itens que ja estao na sacola e desabilitam uma nova inclusao.

### Sacola

A interface deixou de tratar inclusao na sacola como reserva. Ela informa que a
disponibilidade sera confirmada no checkout, sinaliza itens indisponiveis e
calcula o resumo somente com itens disponiveis.

### Feedback visual e acessibilidade

O HTMX aplica estado de requisicao aos formularios e botoes. O modulo `web.js`
mantem toasts, confirmacao de exclusao, galeria e feedback do upload sem montar
HTML recebido de dados da aplicacao.

Outros refinamentos:

- resposta visual de hover, clique e pressao;
- animacao de entrada e saida dos toasts;
- confirmacao visual de item adicionado;
- transicao na troca da foto principal;
- animacao na abertura dos filtros mobile;
- foco visivel para navegacao por teclado;
- estados vazios mais identificaveis;
- suporte a `prefers-reduced-motion`.

As animacoes sao curtas e preservam a identidade visual existente, baseada em
tons naturais, tipografia editorial, superficies claras e cantos arredondados.

## Testes e verificacoes

Foram adicionados testes para:

- bloqueio de publicacao por vendedor impedido de vender;
- atualizacao de perfil com normalizacao e preservacao de dados privados;
- autorizacao e estado permitido no gerenciamento de anuncios;
- exclusao logica;
- perfil publico sem exposicao de contato;
- rejeicao de URL de foto insegura;
- permanencia de item indisponivel no carrinho sem soma ao total;
- detalhe de anuncio com dados publicos do vendedor;
- endpoint de prontidao;
- headers de seguranca.

Verificacoes executadas no fechamento do incremento:

```text
find apps/front/js -name '*.js' -print0 | xargs -0 -n1 node --check
git diff --check
GOCACHE=/tmp/reveste-go-cache go test ./...
GOCACHE=/tmp/reveste-go-cache go vet ./...
```

Todas passaram.

## Limitacoes conhecidas

- checkout, pagamento, entrega e pedidos ainda nao estao implementados;
- a sessao do navegador usa cookie `HttpOnly`, `SameSite=Lax` e `Secure` em HTTPS;
- formularios web e operacoes mutaveis autenticadas por cookie validam `Origin`
  e `Sec-Fetch-Site`;
- o frontend nao persiste token ou dados do perfil em Web Storage;
- imagens sao reencodadas antes do upload para remover EXIF e rejeitar arquivos
  disfarçados, corrompidos ou com dimensoes excessivas;
- a CSP permite imagens apenas do Blob store exato configurado e nao depende mais de
  fontes remotas;
- ainda nao ha testes automatizados de navegador para as transicoes e fluxos web;
- imagens removidas de um anuncio ou abandonadas durante upload ainda podem ficar
  orfas no Blob;
- validacao autoritativa dos bytes no backend e moderacao de imagens continuam pendentes;
- a paginacao do catalogo informa a quantidade carregada, nao o total global;
