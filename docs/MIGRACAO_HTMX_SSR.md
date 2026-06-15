# Migracao do frontend para SSR e HTMX

## Arquitetura resultante

O backend possui dois adaptadores de entrada que compartilham os mesmos
controladores de casos de uso:

```text
HTTP /v1 -> internal/http -> casosdeuso -> dominio/storage
Paginas  -> internal/web  -> casosdeuso -> dominio/storage
```

`internal/http` mantem os contratos JSON existentes. `internal/web` implementa o
adaptador de paginas: registra consultas e comandos HTTP, chama os controladores,
prepara um contexto de apresentacao e responde documentos ou fragmentos HTML.

## Organizacao

- `adaptador.go`: dependencias, criacao do mux e registro das rotas;
- `consultas_paginas.go`: consultas GET e montagem das paginas;
- `comandos_formularios.go`: comandos POST recebidos por formularios;
- `contexto_documento.go`: contrato de apresentacao entregue aos templates;
- `sessao_navegador.go`: identificacao da sessao, cookie e retorno ao login;
- `respostas_html.go`: documentos, fragmentos HTMX e redirecionamentos;
- `apresentacao.go`: formatacao, funcoes de template e traducao de erros;
- `limite_autenticacao.go`: limite de tentativas do formulario de login;
- `templates/estrutura_documento.html`: documento, cabecalho e componentes comuns;
- `templates/catalogo_publico.html`: inicio, catalogo e perfil publico;
- `templates/conta_usuario.html`: autenticacao, perfil, anuncios e carrinho;
- `templates/anuncios_usuario.html`: detalhe e formularios de anuncio;
- `apps/front/css`: estilos preservados da interface anterior;
- `apps/front/assets`: logos e marca;
- `apps/front/js/web.js`: interacoes que dependem do navegador;
- `apps/front/js/uploads.js`: validacao, reencodificacao e upload de imagens;
- `apps/front/js/htmx.min.js`: HTMX 2.0.8 versionado localmente.

O arquivo HTMX possui SHA-256
`22283ef68cb7545914f0a88a1bdedc7256a703d1d580c1d255217d0a50d31313`.

## Fluxo de comunicacao

Uma navegacao convencional recebe um documento HTML completo. Com JavaScript
disponivel, `hx-boost` envia a mesma requisicao por XHR e substitui o `body`,
preservando URL, historico e comportamento dos formularios.

O catalogo usa um fragmento apenas para "Carregar mais". As demais operacoes
retornam documentos completos ou redirecionamentos. Por isso, todos os fluxos
continuam funcionais sem HTMX, exceto selecao, ordenacao e upload de fotos, que
dependem de APIs do navegador.

## Fronteira de JavaScript

JavaScript proprio nao consulta dados para montar paginas e nao mantem estado de
sessao, usuario, carrinho ou catalogo. Ele e limitado a:

- trocar a foto principal da galeria;
- abrir filtros em telas pequenas;
- confirmar exclusao;
- selecionar, ordenar e remover pre-visualizacoes;
- validar, decodificar, reencodificar e enviar fotos;
- remover toasts depois da exibicao.

Novos fluxos devem ser implementados primeiro como rota e formulario SSR.
JavaScript deve ser adicionado somente quando a capacidade nao puder ser
expressa por HTML, CSS ou HTMX declarativo.

## Seguranca

- templates usam escape contextual de `html/template`;
- sessao permanece em cookie `HttpOnly`, `SameSite=Lax` e `Secure` em HTTPS;
- todos os formularios web mutaveis validam `Origin` e `Sec-Fetch-Site`;
- HTMX esta configurado com `allowEval=false` e `allowScriptTags=false`;
- CSP permite scripts somente da propria origem;
- assets estaticos sao expostos apenas em `/styles.css`, `/css`, `/assets` e `/js`;
- nenhuma pagina depende de HTML dinamico construido com `innerHTML`;
- URLs de fotos continuam restritas ao Blob store oficial.

## Evolucao

Para adicionar uma pagina:

1. registrar a consulta GET ou o comando POST em `adaptador.go`;
2. chamar o controlador em `consultas_paginas.go` ou `comandos_formularios.go`;
3. preencher campos explicitamente nomeados de `contextoDocumento`;
4. responder por `respostas_html.go`;
5. adicionar o template e testes de renderizacao, autorizacao e origem.
