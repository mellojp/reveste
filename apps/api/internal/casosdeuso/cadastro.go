package casosdeuso

import (
	"context"
	"strings"
	"time"

	"reveste/apps/api/internal/common"
	dominiocadastros "reveste/apps/api/internal/dominio/cadastros"
)

const hashSenhaInexistente = "pbkdf2_sha256$210000$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// Controla operacoes de cadastro e sessao
type ControladorCadastro struct {
	usuarios OperacoesUsuarios
	sessoes  OperacoesSessoes
	ids      GeradorID
	senhas   GerenciadorSenhas
	relogio  Relogio
}

func NovoControladorCadastro(
	usuarios OperacoesUsuarios,
	sessoes OperacoesSessoes,
	ids GeradorID,
	senhas GerenciadorSenhas,
	relogio Relogio,
) *ControladorCadastro {
	return &ControladorCadastro{
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

func (c *ControladorCadastro) CadastrarUsuario(
	ctx context.Context,
	entrada EntradaCadastro,
) (dominiocadastros.Usuario, error) {
	if len(entrada.Senha) < 8 {
		return dominiocadastros.Usuario{}, common.NovaValidacao(map[string]string{
			"senha": "A senha deve conter pelo menos 8 caracteres.",
		})
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

func (c *ControladorCadastro) Autenticar(ctx context.Context, identificador, senha string) (Sessao, error) {
	usuario, err := c.usuarios.BuscarUsuarioPorEmailOuCPF(ctx, strings.TrimSpace(identificador))
	hash := usuario.HashSenha
	if err != nil {
		hash = hashSenhaInexistente
	}
	senhaValida := c.senhas.Comparar(hash, senha)
	if err != nil || !senhaValida {
		return Sessao{}, common.ErrNaoAutorizado
	}
	token := c.ids.Novo() + c.ids.Novo()
	expiraEm := c.relogio.Agora().Add(24 * time.Hour)
	if err := c.sessoes.CriarSessao(ctx, token, usuario.ID, expiraEm); err != nil {
		return Sessao{}, err
	}
	return Sessao{Token: token, ExpiraEm: expiraEm, Usuario: usuario}, nil
}

func (c *ControladorCadastro) IdentificarUsuario(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", common.ErrNaoAutorizado
	}
	return c.sessoes.BuscarUsuarioDaSessao(ctx, token, c.relogio.Agora())
}

func (c *ControladorCadastro) EncerrarSessao(ctx context.Context, token string) error {
	return c.sessoes.RemoverSessao(ctx, token)
}

func (c *ControladorCadastro) ObterPerfil(
	ctx context.Context,
	idUsuario string,
) (dominiocadastros.Usuario, error) {
	return c.usuarios.BuscarUsuarioPorID(ctx, idUsuario)
}
