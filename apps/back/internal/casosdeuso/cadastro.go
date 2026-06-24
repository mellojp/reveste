package casosdeuso

import (
	"context"
	"strings"
	"time"

	"reveste/apps/back/internal/common"
	dominiocadastros "reveste/apps/back/internal/dominio/cadastros"
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

type EntradaAtualizacaoPerfil struct {
	Nome     string
	Email    string
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

// ListarEnderecos devolve todos os enderecos ativos do usuario (principal primeiro).
func (c *ControladorCadastro) ListarEnderecos(
	ctx context.Context,
	idUsuario string,
) ([]dominiocadastros.Endereco, error) {
	enderecos, err := c.usuarios.ListarEnderecos(ctx, idUsuario)
	if err != nil {
		return nil, err
	}
	if enderecos == nil {
		enderecos = []dominiocadastros.Endereco{}
	}
	return enderecos, nil
}

// BuscarEndereco devolve um endereco especifico do usuario.
func (c *ControladorCadastro) BuscarEndereco(
	ctx context.Context,
	idUsuario, idEndereco string,
) (dominiocadastros.Endereco, error) {
	return c.usuarios.BuscarEndereco(ctx, idUsuario, idEndereco)
}

// AdicionarEndereco valida e cria um novo endereco (nao principal) para o usuario.
func (c *ControladorCadastro) AdicionarEndereco(
	ctx context.Context,
	idUsuario string,
	entrada dominiocadastros.Endereco,
) (dominiocadastros.Endereco, error) {
	entrada.Normalizar()
	if campos := entrada.Validar(); len(campos) > 0 {
		return dominiocadastros.Endereco{}, common.NovaValidacao(campos)
	}
	entrada.ID = c.ids.Novo()
	if err := c.usuarios.AdicionarEndereco(ctx, idUsuario, entrada, c.relogio.Agora()); err != nil {
		return dominiocadastros.Endereco{}, err
	}
	return entrada, nil
}

// AtualizarEndereco valida e atualiza um endereco existente do usuario.
func (c *ControladorCadastro) AtualizarEndereco(
	ctx context.Context,
	idUsuario, idEndereco string,
	entrada dominiocadastros.Endereco,
) error {
	entrada.Normalizar()
	if campos := entrada.Validar(); len(campos) > 0 {
		return common.NovaValidacao(campos)
	}
	return c.usuarios.AtualizarEndereco(ctx, idUsuario, idEndereco, entrada, c.relogio.Agora())
}

// DefinirEnderecoPrincipal marca o endereco escolhido como principal do usuario.
func (c *ControladorCadastro) DefinirEnderecoPrincipal(
	ctx context.Context,
	idUsuario, idEndereco string,
) error {
	return c.usuarios.DefinirEnderecoPrincipal(ctx, idUsuario, idEndereco, c.relogio.Agora())
}

// RemoverEndereco exclui logicamente um endereco. O endereco principal nao pode ser removido
// antes de outro ser definido como principal.
func (c *ControladorCadastro) RemoverEndereco(
	ctx context.Context,
	idUsuario, idEndereco string,
) error {
	endereco, err := c.usuarios.BuscarEndereco(ctx, idUsuario, idEndereco)
	if err != nil {
		return err
	}
	if endereco.Principal {
		return common.ErrNaoPermitido
	}
	return c.usuarios.RemoverEndereco(ctx, idUsuario, idEndereco, c.relogio.Agora())
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

func (c *ControladorCadastro) AtualizarPerfil(
	ctx context.Context,
	idUsuario string,
	entrada EntradaAtualizacaoPerfil,
) (dominiocadastros.Usuario, error) {
	usuario, err := c.usuarios.BuscarUsuarioPorID(ctx, idUsuario)
	if err != nil {
		return dominiocadastros.Usuario{}, err
	}
	usuario.Nome = entrada.Nome
	usuario.Email = entrada.Email
	usuario.Telefone = entrada.Telefone
	usuario.EnderecoPrincipal = entrada.Endereco
	usuario.AtualizadoEm = c.relogio.Agora()
	usuario.Normalizar()
	if err := usuario.Validar(); err != nil {
		return dominiocadastros.Usuario{}, err
	}
	if err := c.usuarios.AtualizarUsuario(ctx, usuario); err != nil {
		return dominiocadastros.Usuario{}, err
	}
	return usuario, nil
}
