# Pendencias tecnicas

Este documento registra o backlog tecnico posterior ao fechamento do fluxo
transacional inicial. A ordem abaixo considera risco operacional, dependencia e
impacto sobre uma futura implantacao real.

## P0 - antes de operar com dinheiro real

- integrar um gateway de pagamento com chave de idempotencia, webhook assinado e
  reconciliacao de respostas desconhecidas ou timeouts;
- substituir o reembolso simulado por reembolso real e implementar o repasse ao
  vendedor;
- executar migracoes por uma ferramenta versionada no deploy, sem depender de
  `docker-entrypoint-initdb.d`;
- mover os jobs temporais para um executor unico ou adicionar coordenacao
  distribuida. Hoje cada instancia da API executa os jobs;
- adicionar observabilidade para checkout, pagamentos, jobs, reembolsos e
  transicoes de pedido: logs correlacionados, metricas e alertas;
- implementar CI com testes Go, `go vet`, JavaScript e integracao PostgreSQL.

## P1 - robustez operacional e escala

- implementar outbox e processamento assincrono para eventos financeiros,
  notificacoes e integracoes externas;
- paginar pedidos e vendas por cursor;
- integrar gateway de frete, cotacao e validacao de rastreio;
- adicionar testes E2E de navegador para conta, anuncio, carrinho, checkout,
  envio, recebimento e avaliacao;
- criar rotinas de limpeza de sessoes, tentativas antigas e imagens orfas;
- processar e validar os bytes das imagens em fronteira confiavel, com moderacao
  e limites de requisicao;
- definir recuperacao e reconciliacao quando um provedor externo fica
  indisponivel;
- validar o modelo de deploy escolhido para conexoes PostgreSQL, jobs e processos
  persistentes.

## P2 - funcionalidades e governanca

- evoluir as notificacoes para entrega assincrona via outbox e cobrir os eventos do job de
  prazos (item nao enviado, reembolso) alem dos eventos sincronos ja implementados;
- evoluir o chat por pedido: marcacao de leitura de mensagens, atualizacao em tempo real e
  anexos por store privado;
- concluir politica de cancelamento, contestacao, reembolso parcial e tratamento
  do frete;
- implementar exportacao, retencao e exclusao de dados conforme LGPD;
- publicar Termos de Uso e Politica de Privacidade;
- atualizar os PDFs arquiteturais antigos para SSR/HTMX, Vercel Blob e o modelo
  transacional atual;
- realizar testes de carga, auditoria de autorizacao horizontal, verificacao de
  dependencias e varredura de segredos.

## Evolucao visual em andamento

Lote concluido (consolidacao do design system):

- escala de tokens de raio (`--radius-sm/md/lg`) e elevacao (`--shadow-rest`)
  aplicada em todos os cards, paineis e controles, eliminando valores
  hardcoded divergentes (12/16/18/22px e sombras quase-iguais);
- primitiva `.card` / `.card-interactive` como base unica de superficie,
  ja adotada pelos cartoes de pedido e venda;
- escala unica de titulo para paginas utilitarias (perfil, sacola, checkout,
  pedidos, formularios) e normalizacao dos titulos de card;
- badges de status semanticos (`classeStatus` -> sucesso/andamento/negativo/
  neutro) em anuncios, pedidos e vendas;
- piso comum dos campos de formulario (altura, fundo e hover do cadastro)
  estendido a perfil, endereco, anuncio e login;
- pagina de perfil mais densa; endereco principal destacado e formulario de
  novo endereco em disclosure; `meus-anuncios` integrado ao layout de conta
  com a navegacao lateral.

Os proximos lotes devem padronizar os demais cards (produto, anuncio,
carrinho) sobre a primitiva `.card`, criar uma variante compacta do rodape e
adicionar testes visuais responsivos.
