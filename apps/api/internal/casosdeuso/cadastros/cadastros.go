package cadastros

import (
	"context"
	"strings"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	dominiocadastros "reveste/apps/api/internal/dominio/cadastros"
	errosdominio "reveste/apps/api/internal/dominio/erros"
)

// controller
type FluxoCadastro struct {
	usuarios casosdeuso.OperacoesUsuarios
	sessoes  casosdeuso.OperacoesSessoes
	ids      casosdeuso.GeradorID
	senhas   casosdeuso.GerenciadorSenhas
	relogio  casosdeuso.Relogio
}

func NovoFluxoCadastro(
	usuarios casosdeuso.OperacoesUsuarios,
	sessoes casosdeuso.OperacoesSessoes,
	ids casosdeuso.GeradorID,
	senhas casosdeuso.GerenciadorSenhas,
	relogio casosdeuso.Relogio,
) *FluxoCadastro {
	return &FluxoCadastro{
		usuarios: usuarios,
		sessoes:  sessoes,
		ids:      ids,
		senhas:   senhas,
		relogio:  relogio,
	}
}

type EntradaCadastro struct {
	Nome     string
	CPF      string
	Email    string
	Senha    string
	Telefone string
	Endereco dominiocadastros.Endereco
}

func (c *FluxoCadastro) CadastrarUsuario(
	ctx context.Context,
	entrada EntradaCadastro,
) (dominiocadastros.Usuario, error) {
	if len(entrada.Senha) < 8 {
		return dominiocadastros.Usuario{}, errosdominio.ErrDadosInvalidos
	}
	hash, err := c.senhas.Gerar(entrada.Senha)
	if err != nil {
		return dominiocadastros.Usuario{}, err
	}
	agora := c.relogio.Agora()
	usuario := dominiocadastros.Usuario{
		ID: c.ids.Novo(), Nome: entrada.Nome, CPF: entrada.CPF, Email: entrada.Email,
		HashSenha: hash, Telefone: entrada.Telefone, EnderecoPrincipal: entrada.Endereco,
		CriadoEm: agora, AtualizadoEm: agora,
	}
	usuario.Normalizar()
	if err := usuario.Validar(); err != nil {
		return dominiocadastros.Usuario{}, err
	}
	if err := c.usuarios.CriarUsuario(ctx, usuario); err != nil {
		return dominiocadastros.Usuario{}, err
	}
	return usuario, nil
}

type Sessao struct {
	Token    string                   `json:"token"`
	ExpiraEm time.Time                `json:"expira_em"`
	Usuario  dominiocadastros.Usuario `json:"usuario"`
}

func (c *FluxoCadastro) Autenticar(ctx context.Context, identificador, senha string) (Sessao, error) {
	usuario, err := c.usuarios.BuscarUsuarioPorEmailOuCPF(ctx, strings.TrimSpace(identificador))
	if err != nil || !c.senhas.Comparar(usuario.HashSenha, senha) {
		return Sessao{}, errosdominio.ErrNaoAutorizado
	}
	token := c.ids.Novo() + c.ids.Novo()
	expiraEm := c.relogio.Agora().Add(24 * time.Hour)
	if err := c.sessoes.CriarSessao(ctx, token, usuario.ID, expiraEm); err != nil {
		return Sessao{}, err
	}
	return Sessao{Token: token, ExpiraEm: expiraEm, Usuario: usuario}, nil
}

func (c *FluxoCadastro) IdentificarUsuario(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", errosdominio.ErrNaoAutorizado
	}
	return c.sessoes.BuscarUsuarioDaSessao(ctx, token, c.relogio.Agora())
}

func (c *FluxoCadastro) EncerrarSessao(ctx context.Context, token string) error {
	return c.sessoes.RemoverSessao(ctx, token)
}
