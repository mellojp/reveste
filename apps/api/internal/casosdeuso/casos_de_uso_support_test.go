package casosdeuso_test

import (
	"context"
	"strings"
	"sync"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/anuncios"
	"reveste/apps/api/internal/dominio/cadastros"
	"reveste/apps/api/internal/dominio/compras"
)

type Store struct {
	mu sync.RWMutex

	usuarios           map[string]cadastros.Usuario
	usuarioPorEmail    map[string]string
	usuarioPorCPF      map[string]string
	anuncios           map[string]anuncios.Anuncio
	carrinhoPorUsuario map[string]compras.Carrinho
	sessoes            map[string]sessao
}

type sessao struct {
	IDUsuario string
	ExpiraEm  time.Time
}

func newTestStore() *Store {
	return &Store{
		usuarios:           make(map[string]cadastros.Usuario),
		usuarioPorEmail:    make(map[string]string),
		usuarioPorCPF:      make(map[string]string),
		anuncios:           make(map[string]anuncios.Anuncio),
		carrinhoPorUsuario: make(map[string]compras.Carrinho),
		sessoes:            make(map[string]sessao),
	}
}

func (r *Store) CriarUsuario(ctx context.Context, usuario cadastros.Usuario) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, existe := r.usuarioPorEmail[usuario.Email]; existe {
		return common.ErrConflito
	}
	if _, existe := r.usuarioPorCPF[usuario.CPF]; existe {
		return common.ErrConflito
	}
	r.usuarios[usuario.ID] = usuario
	r.usuarioPorEmail[usuario.Email] = usuario.ID
	r.usuarioPorCPF[usuario.CPF] = usuario.ID
	return nil
}

func (r *Store) BuscarUsuarioPorID(ctx context.Context, id string) (cadastros.Usuario, error) {
	if err := ctx.Err(); err != nil {
		return cadastros.Usuario{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	usuario, existe := r.usuarios[id]
	if !existe {
		return cadastros.Usuario{}, common.ErrNaoEncontrado
	}
	return usuario, nil
}

func (r *Store) BuscarUsuarioPorEmailOuCPF(ctx context.Context, identificador string) (cadastros.Usuario, error) {
	if err := ctx.Err(); err != nil {
		return cadastros.Usuario{}, err
	}
	normalizado := strings.ToLower(strings.TrimSpace(identificador))
	cpf := cadastros.NormalizarCPF(identificador)
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, existe := r.usuarioPorEmail[normalizado]
	if !existe {
		id, existe = r.usuarioPorCPF[cpf]
	}
	if !existe {
		return cadastros.Usuario{}, common.ErrNaoEncontrado
	}
	return r.usuarios[id], nil
}

func (r *Store) CriarAnuncio(ctx context.Context, anuncio anuncios.Anuncio) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, existe := r.anuncios[anuncio.ID]; existe {
		return common.ErrConflito
	}
	r.anuncios[anuncio.ID] = copiarAnuncio(anuncio)
	return nil
}

func (r *Store) BuscarAnuncioPorID(ctx context.Context, id string) (anuncios.Anuncio, error) {
	if err := ctx.Err(); err != nil {
		return anuncios.Anuncio{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	anuncio, existe := r.anuncios[id]
	if !existe {
		return anuncios.Anuncio{}, common.ErrNaoEncontrado
	}
	return copiarAnuncio(anuncio), nil
}

func (r *Store) ListarAnuncios(
	ctx context.Context,
	filtro casosdeuso.FiltroAnuncios,
) ([]anuncios.Anuncio, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	resultado := make([]anuncios.Anuncio, 0, filtro.Limite)
	pular := filtro.Deslocamento
	for _, anuncio := range r.anuncios {
		if !correspondeAoFiltro(anuncio, filtro) {
			continue
		}
		if pular > 0 {
			pular--
			continue
		}
		resultado = append(resultado, copiarAnuncio(anuncio))
		if len(resultado) == filtro.Limite {
			break
		}
	}
	return resultado, nil
}

func correspondeAoFiltro(anuncio anuncios.Anuncio, filtro casosdeuso.FiltroAnuncios) bool {
	if !filtro.IncluirTodosStatus && anuncio.Status != anuncios.StatusAnuncioDisponivel {
		return false
	}
	if filtro.IDVendedor != "" && anuncio.IDVendedor != filtro.IDVendedor {
		return false
	}
	if filtro.ExcluirVendedor != "" && anuncio.IDVendedor == filtro.ExcluirVendedor {
		return false
	}
	palavra := strings.ToLower(strings.TrimSpace(filtro.Palavra))
	if palavra != "" && !strings.Contains(strings.ToLower(anuncio.Titulo+" "+anuncio.Descricao), palavra) {
		return false
	}
	if filtro.Categoria != "" && anuncio.Categoria != strings.ToLower(filtro.Categoria) {
		return false
	}
	if filtro.Tamanho != "" && anuncio.Tamanho != strings.ToUpper(filtro.Tamanho) {
		return false
	}
	if filtro.EstadoConservacao != "" && anuncio.EstadoConservacao != filtro.EstadoConservacao {
		return false
	}
	if filtro.PrecoMinCentavos > 0 && anuncio.PrecoCentavos < filtro.PrecoMinCentavos {
		return false
	}
	return filtro.PrecoMaxCentavos <= 0 || anuncio.PrecoCentavos <= filtro.PrecoMaxCentavos
}

func (r *Store) ObterOuCriarCarrinho(ctx context.Context, novoID, idUsuario string, agora time.Time) (compras.Carrinho, error) {
	if err := ctx.Err(); err != nil {
		return compras.Carrinho{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if carrinho, existe := r.carrinhoPorUsuario[idUsuario]; existe {
		return copiarCarrinho(carrinho), nil
	}
	carrinho := compras.Carrinho{
		ID: novoID, IDUsuario: idUsuario, IDsAnuncios: []string{},
		CriadoEm: agora, AtualizadoEm: agora,
	}
	r.carrinhoPorUsuario[idUsuario] = carrinho
	return copiarCarrinho(carrinho), nil
}

func (r *Store) AdicionarAnuncioAoCarrinho(
	ctx context.Context,
	novoID,
	idUsuario,
	idAnuncio string,
	agora time.Time,
) (compras.Carrinho, error) {
	if err := ctx.Err(); err != nil {
		return compras.Carrinho{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	carrinho, existe := r.carrinhoPorUsuario[idUsuario]
	if !existe {
		carrinho = compras.Carrinho{
			ID: novoID, IDUsuario: idUsuario, IDsAnuncios: []string{},
			CriadoEm: agora, AtualizadoEm: agora,
		}
	}
	carrinho.Adicionar(idAnuncio)
	carrinho.AtualizadoEm = agora
	r.carrinhoPorUsuario[carrinho.IDUsuario] = copiarCarrinho(carrinho)
	return copiarCarrinho(carrinho), nil
}

func (r *Store) RemoverAnuncioDoCarrinho(
	ctx context.Context,
	novoID,
	idUsuario,
	idAnuncio string,
	agora time.Time,
) (compras.Carrinho, error) {
	if err := ctx.Err(); err != nil {
		return compras.Carrinho{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	carrinho, existe := r.carrinhoPorUsuario[idUsuario]
	if !existe {
		carrinho = compras.Carrinho{
			ID: novoID, IDUsuario: idUsuario, IDsAnuncios: []string{},
			CriadoEm: agora, AtualizadoEm: agora,
		}
	}
	carrinho.Remover(idAnuncio)
	carrinho.AtualizadoEm = agora
	r.carrinhoPorUsuario[idUsuario] = copiarCarrinho(carrinho)
	return copiarCarrinho(carrinho), nil
}

func (r *Store) CriarSessao(ctx context.Context, token, idUsuario string, expiraEm time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessoes[token] = sessao{IDUsuario: idUsuario, ExpiraEm: expiraEm}
	return nil
}

func (r *Store) BuscarUsuarioDaSessao(ctx context.Context, token string, agora time.Time) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	sessaoAtual, existe := r.sessoes[token]
	if !existe || !agora.Before(sessaoAtual.ExpiraEm) {
		return "", common.ErrNaoAutorizado
	}
	return sessaoAtual.IDUsuario, nil
}

func (r *Store) RemoverSessao(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessoes, token)
	return nil
}

func copiarAnuncio(anuncio anuncios.Anuncio) anuncios.Anuncio {
	anuncio.Fotos = append([]anuncios.Foto(nil), anuncio.Fotos...)
	return anuncio
}

func copiarCarrinho(carrinho compras.Carrinho) compras.Carrinho {
	carrinho.IDsAnuncios = append([]string(nil), carrinho.IDsAnuncios...)
	return carrinho
}
