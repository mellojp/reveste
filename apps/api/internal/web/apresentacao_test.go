package web

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/interacao"
)

// renderizarDocumento monta o conjunto de templates exatamente como o adaptador e executa o
// documento completo, exercitando cabecalho, rodape e o conteudo selecionado. Falhas de
// execucao (campos/funcoes ausentes) que o ParseFS nao pega aparecem aqui.
func renderizarDocumento(t *testing.T, contexto contextoDocumento) string {
	t.Helper()
	tmpl, err := template.New("web").
		Funcs(funcoesApresentacaoTemplates()).
		ParseFS(arquivosTemplates, "templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "documento", contexto); err != nil {
		t.Fatalf("executar documento (%s): %v", contexto.Conteudo, err)
	}
	return buf.String()
}

func TestRenderizarPaginaNotificacoes(t *testing.T) {
	usuario := cadastros.Usuario{ID: "u1", Nome: "Ana Vendedora"}
	saida := renderizarDocumento(t, contextoDocumento{
		Conteudo: conteudoNotificacoes, Titulo: "Notificações",
		UsuarioAutenticado:   &usuario,
		NotificacoesNaoLidas: 1,
		NotificacoesListadas: []interacao.Notificacao{
			{ID: "n1", IDUsuario: "u1", Tipo: interacao.NotificacaoPedidoEnviado,
				Conteudo: "Seu pedido foi enviado.", IDPedido: "p1", CriadaEm: time.Now()},
		},
	})
	if !strings.Contains(saida, "Seu pedido foi enviado.") {
		t.Fatal("conteúdo da notificação não renderizado")
	}
	if !strings.Contains(saida, "/meus-pedidos/p1") {
		t.Fatal("deep-link da notificação ausente")
	}
	if !strings.Contains(saida, `/notificacoes/n1/remover`) ||
		!strings.Contains(saida, `/notificacoes/limpar`) {
		t.Fatal("ações de remoção das notificações ausentes")
	}
}

func TestNotificacaoVendaDirecionaParaVenda(t *testing.T) {
	if destino := linkNotificacao(interacao.Notificacao{
		Tipo: interacao.NotificacaoVendaRealizada, IDPedido: "p1",
	}); destino != "/minhas-vendas/p1" {
		t.Fatalf("link de nova venda = %q", destino)
	}
}

func TestRenderizarPaginaConversa(t *testing.T) {
	usuario := cadastros.Usuario{ID: "c1", Nome: "Caio Comprador"}
	saida := renderizarDocumento(t, contextoDocumento{
		Conteudo: conteudoConversa, Titulo: "Conversa",
		UsuarioAutenticado:    &usuario,
		InterlocutorNome:      "Vera Vendedora",
		URLPerfilInterlocutor: "/usuarios/v1",
		URLPedidoOrigem:       "/meus-pedidos/p1",
		ConversaDetalhe: &casosdeuso.ConversaDetalhada{
			IDPedido: "p1", IDConversa: "conv1", IDComprador: "c1", IDVendedor: "v1",
			Mensagens: []interacao.Mensagem{
				{ID: "m1", IDUsuarioRemetente: "c1", Conteudo: "Olá!", CriadaEm: time.Now()},
				{ID: "m2", IDUsuarioRemetente: "v1", Conteudo: "Oi, tudo bem?", CriadaEm: time.Now()},
			},
		},
	})
	if !strings.Contains(saida, "Vera Vendedora") || !strings.Contains(saida, "Oi, tudo bem?") {
		t.Fatal("conversa não renderizada como esperado")
	}
	if !strings.Contains(saida, `action="/pedidos/p1/mensagens"`) {
		t.Fatal("formulário de envio de mensagem ausente")
	}
	if !strings.Contains(saida, `href="/usuarios/v1"`) ||
		!strings.Contains(saida, `href="/meus-pedidos/p1"`) {
		t.Fatal("chat sem acesso ao perfil ou retorno aos detalhes")
	}
	if !strings.Contains(saida, `hx-get="/pedidos/p1/conversa/mensagens"`) || !strings.Contains(saida, `hx-trigger="every 5s"`) {
		t.Fatal("thread da conversa sem polling HTMX")
	}
	// Sem hx-target proprio, o thread herdaria hx-target="body" do <body> e o polling
	// substituiria a pagina inteira pelo fragmento. Travamos isso aqui.
	if !strings.Contains(saida, `hx-target="this"`) {
		t.Fatal("thread sem hx-target=this: o polling trocaria o body inteiro")
	}
}

func TestRenderizarFragmentoConversaThread(t *testing.T) {
	tmpl, err := template.New("web").
		Funcs(funcoesApresentacaoTemplates()).
		ParseFS(arquivosTemplates, "templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	usuario := cadastros.Usuario{ID: "c1", Nome: "Caio"}
	var buf bytes.Buffer
	contexto := contextoDocumento{
		UsuarioAutenticado: &usuario,
		ConversaDetalhe: &casosdeuso.ConversaDetalhada{
			IDPedido: "p1", IDComprador: "c1", IDVendedor: "v1",
			Mensagens: []interacao.Mensagem{
				{ID: "m1", IDUsuarioRemetente: "v1", Conteudo: "chegou?", CriadaEm: time.Now()},
			},
		},
	}
	if err := tmpl.ExecuteTemplate(&buf, "conversa-thread", contexto); err != nil {
		t.Fatalf("executar fragmento: %v", err)
	}
	saida := buf.String()
	if !strings.Contains(saida, `id="chat-thread"`) || !strings.Contains(saida, "chegou?") {
		t.Fatalf("fragmento do thread incompleto: %s", saida)
	}
	if !strings.Contains(saida, `hx-target="this"`) || !strings.Contains(saida, `hx-push-url="false"`) {
		t.Fatal("fragmento do thread deve mirar nele mesmo e não empurrar URL no polling")
	}
	if !strings.Contains(saida, "is-theirs") {
		t.Fatal("mensagem do interlocutor deveria ter a classe is-theirs")
	}
}
