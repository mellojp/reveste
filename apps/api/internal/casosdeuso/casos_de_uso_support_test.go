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
	"reveste/apps/api/internal/dominio/interacao"
)

type Store struct {
	mu sync.RWMutex

	usuarios            map[string]cadastros.Usuario
	usuarioPorEmail     map[string]string
	usuarioPorCPF       map[string]string
	anuncios            map[string]anuncios.Anuncio
	carrinhoPorUsuario  map[string]compras.Carrinho
	sessoes             map[string]sessao
	comprasPorChave     map[string]compras.Compra
	pedidosPorComprador map[string][]compras.Pedido
	enderecosPorUsuario map[string][]cadastros.Endereco
	notificacoes        []interacao.Notificacao
}

type sessao struct {
	IDUsuario string
	ExpiraEm  time.Time
}

func newTestStore() *Store {
	return &Store{
		usuarios:            make(map[string]cadastros.Usuario),
		usuarioPorEmail:     make(map[string]string),
		usuarioPorCPF:       make(map[string]string),
		anuncios:            make(map[string]anuncios.Anuncio),
		carrinhoPorUsuario:  make(map[string]compras.Carrinho),
		sessoes:             make(map[string]sessao),
		comprasPorChave:     make(map[string]compras.Compra),
		pedidosPorComprador: make(map[string][]compras.Pedido),
		enderecosPorUsuario: make(map[string][]cadastros.Endereco),
	}
}

func (r *Store) ListarEnderecos(ctx context.Context, idUsuario string) ([]cadastros.Endereco, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]cadastros.Endereco(nil), r.enderecosPorUsuario[idUsuario]...), nil
}

func (r *Store) BuscarEndereco(ctx context.Context, idUsuario, idEndereco string) (cadastros.Endereco, error) {
	if err := ctx.Err(); err != nil {
		return cadastros.Endereco{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.enderecosPorUsuario[idUsuario] {
		if e.ID == idEndereco {
			return e, nil
		}
	}
	return cadastros.Endereco{}, common.ErrNaoEncontrado
}

func (r *Store) AdicionarEndereco(ctx context.Context, idUsuario string, endereco cadastros.Endereco, _ time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	endereco.Principal = false
	r.enderecosPorUsuario[idUsuario] = append(r.enderecosPorUsuario[idUsuario], endereco)
	return nil
}

func (r *Store) AtualizarEndereco(ctx context.Context, idUsuario, idEndereco string, endereco cadastros.Endereco, _ time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	lista := r.enderecosPorUsuario[idUsuario]
	for i, e := range lista {
		if e.ID == idEndereco {
			endereco.ID = idEndereco
			endereco.Principal = e.Principal
			lista[i] = endereco
			return nil
		}
	}
	return common.ErrNaoEncontrado
}

func (r *Store) RemoverEndereco(ctx context.Context, idUsuario, idEndereco string, _ time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	lista := r.enderecosPorUsuario[idUsuario]
	for i, e := range lista {
		if e.ID == idEndereco {
			r.enderecosPorUsuario[idUsuario] = append(lista[:i], lista[i+1:]...)
			return nil
		}
	}
	return common.ErrNaoEncontrado
}

func (r *Store) DefinirEnderecoPrincipal(ctx context.Context, idUsuario, idEndereco string, _ time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	lista := r.enderecosPorUsuario[idUsuario]
	encontrado := false
	for i := range lista {
		lista[i].Principal = lista[i].ID == idEndereco
		if lista[i].ID == idEndereco {
			encontrado = true
		}
	}
	if !encontrado {
		return common.ErrNaoEncontrado
	}
	return nil
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

func (r *Store) AtualizarUsuario(ctx context.Context, usuario cadastros.Usuario) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	anterior, existe := r.usuarios[usuario.ID]
	if !existe {
		return common.ErrNaoEncontrado
	}
	delete(r.usuarioPorEmail, anterior.Email)
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

func (r *Store) CriarNotificacao(ctx context.Context, n interacao.Notificacao) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notificacoes = append(r.notificacoes, n)
	return nil
}

func (r *Store) ListarNotificacoes(ctx context.Context, idUsuario string, limite int) ([]interacao.Notificacao, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var lista []interacao.Notificacao
	for i := len(r.notificacoes) - 1; i >= 0 && len(lista) < limite; i-- {
		if r.notificacoes[i].IDUsuario == idUsuario {
			lista = append(lista, r.notificacoes[i])
		}
	}
	return lista, nil
}

func (r *Store) ContarNotificacoesNaoLidas(ctx context.Context, idUsuario string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	total := 0
	for _, n := range r.notificacoes {
		if n.IDUsuario == idUsuario && n.LidaEm == nil {
			total++
		}
	}
	return total, nil
}

func (r *Store) MarcarNotificacoesLidas(ctx context.Context, idUsuario string, agora time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.notificacoes {
		if r.notificacoes[i].IDUsuario == idUsuario && r.notificacoes[i].LidaEm == nil {
			lida := agora
			r.notificacoes[i].LidaEm = &lida
		}
	}
	return nil
}

func (r *Store) RemoverNotificacao(ctx context.Context, idUsuario, idNotificacao string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, n := range r.notificacoes {
		if n.ID == idNotificacao && n.IDUsuario == idUsuario {
			r.notificacoes = append(r.notificacoes[:i], r.notificacoes[i+1:]...)
			break
		}
	}
	return nil
}

func (r *Store) LimparNotificacoes(ctx context.Context, idUsuario string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	filtradas := r.notificacoes[:0]
	for _, n := range r.notificacoes {
		if n.IDUsuario != idUsuario {
			filtradas = append(filtradas, n)
		}
	}
	r.notificacoes = filtradas
	return nil
}

func (r *Store) ReativarVendedor(ctx context.Context, idVendedor string, agora time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	usuario, existe := r.usuarios[idVendedor]
	if !existe || !usuario.BloqueadoParaVendas {
		return false, nil
	}
	usuario.BloqueadoParaVendas = false
	usuario.ItensNaoEnviados = 0
	usuario.AtualizadoEm = agora
	r.usuarios[idVendedor] = usuario
	return true, nil
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

func (r *Store) AtualizarAnuncio(ctx context.Context, anuncio anuncios.Anuncio) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, existe := r.anuncios[anuncio.ID]; !existe {
		return common.ErrNaoEncontrado
	}
	r.anuncios[anuncio.ID] = copiarAnuncio(anuncio)
	return nil
}

func (r *Store) ExcluirAnuncio(
	ctx context.Context,
	idAnuncio,
	idVendedor string,
	agora time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	anuncio, existe := r.anuncios[idAnuncio]
	if !existe {
		return common.ErrNaoEncontrado
	}
	if anuncio.IDVendedor != idVendedor {
		return common.ErrNaoPermitido
	}
	anuncio.Status = anuncios.StatusAnuncioExcluido
	anuncio.ExcluidoEm = &agora
	anuncio.AtualizadoEm = agora
	r.anuncios[idAnuncio] = anuncio
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
	if anuncio.ExcluidoEm != nil {
		return false
	}
	if len(filtro.IDsAnuncios) > 0 && !contemID(filtro.IDsAnuncios, anuncio.ID) {
		return false
	}
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

func contemID(ids []string, procurado string) bool {
	for _, id := range ids {
		if id == procurado {
			return true
		}
	}
	return false
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

func (r *Store) BuscarCompraPorChave(ctx context.Context, chave string) (compras.Compra, error) {
	if err := ctx.Err(); err != nil {
		return compras.Compra{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	compra, existe := r.comprasPorChave[chave]
	if !existe {
		return compras.Compra{}, common.ErrNaoEncontrado
	}
	return compra, nil
}

func (r *Store) IniciarCompra(
	ctx context.Context,
	compra compras.Compra,
	pagamento compras.Pagamento,
	idComprador string,
) (compras.Compra, bool, error) {
	if err := ctx.Err(); err != nil {
		return compras.Compra{}, false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existente, ok := r.comprasPorChave[compra.ChaveIdempotencia]; ok {
		return existente, false, nil
	}
	ids := compra.IDsAnuncios()
	for _, id := range ids {
		anuncio, existe := r.anuncios[id]
		if !existe || anuncio.Status != anuncios.StatusAnuncioDisponivel {
			return compras.Compra{}, false, common.ErrAnuncioIndisponivel
		}
	}
	for _, id := range ids {
		anuncio := r.anuncios[id]
		anuncio.Status = anuncios.StatusAnuncioReservado
		anuncio.AtualizadoEm = compra.CriadaEm
		r.anuncios[id] = anuncio
	}
	r.comprasPorChave[compra.ChaveIdempotencia] = compra
	r.pedidosPorComprador[idComprador] = append(r.pedidosPorComprador[idComprador], compra.Pedidos...)
	return compra, true, nil
}

func (r *Store) ConfirmarCompraAprovada(
	ctx context.Context,
	chave, provedor, identificadorExterno string,
	agora time.Time,
) (compras.Compra, error) {
	if err := ctx.Err(); err != nil {
		return compras.Compra{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	compra, existe := r.comprasPorChave[chave]
	if !existe {
		return compras.Compra{}, common.ErrNaoEncontrado
	}
	if compra.Status == compras.StatusCompraAprovada {
		return compra, nil
	}
	if compra.Status != compras.StatusCompraAguardandoPagamento {
		return compras.Compra{}, common.ErrTransicaoInvalida
	}
	compra.Status = compras.StatusCompraAprovada
	for indice := range compra.Pedidos {
		compra.Pedidos[indice].Status = compras.StatusPedidoAguardandoEnvio
		for _, item := range compra.Pedidos[indice].Itens {
			anuncio := r.anuncios[item.IDAnuncio]
			if anuncio.Status != anuncios.StatusAnuncioReservado {
				return compras.Compra{}, common.ErrTransicaoInvalida
			}
			anuncio.Status = anuncios.StatusAnuncioVendido
			anuncio.AtualizadoEm = agora
			r.anuncios[item.IDAnuncio] = anuncio
		}
	}
	r.comprasPorChave[chave] = compra
	r.pedidosPorComprador[compra.IDComprador] = append([]compras.Pedido(nil), compra.Pedidos...)
	if carrinho, existe := r.carrinhoPorUsuario[compra.IDComprador]; existe {
		for _, id := range compra.IDsAnuncios() {
			carrinho.Remover(id)
		}
		carrinho.AtualizadoEm = agora
		r.carrinhoPorUsuario[compra.IDComprador] = copiarCarrinho(carrinho)
	}
	return compra, nil
}

func (r *Store) RecusarCompra(
	ctx context.Context,
	chave, provedor, identificadorExterno string,
	agora time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	compra, existe := r.comprasPorChave[chave]
	if !existe {
		return common.ErrNaoEncontrado
	}
	if compra.Status != compras.StatusCompraAguardandoPagamento {
		return nil
	}
	compra.Status = compras.StatusCompraRecusada
	for indice := range compra.Pedidos {
		compra.Pedidos[indice].Status = compras.StatusPedidoCancelado
		for _, item := range compra.Pedidos[indice].Itens {
			anuncio := r.anuncios[item.IDAnuncio]
			if anuncio.Status == anuncios.StatusAnuncioReservado {
				anuncio.Status = anuncios.StatusAnuncioDisponivel
				anuncio.AtualizadoEm = agora
				r.anuncios[item.IDAnuncio] = anuncio
			}
		}
	}
	r.comprasPorChave[chave] = compra
	r.pedidosPorComprador[compra.IDComprador] = append([]compras.Pedido(nil), compra.Pedidos...)
	if carrinho, ok := r.carrinhoPorUsuario[compra.IDComprador]; ok {
		carrinho.AtualizadoEm = agora
		r.carrinhoPorUsuario[compra.IDComprador] = carrinho
	}
	return nil
}

func (r *Store) ExpirarComprasPendentes(ctx context.Context, agora time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	afetadas := 0
	for chave, compra := range r.comprasPorChave {
		if compra.Status != compras.StatusCompraAguardandoPagamento || compra.ExpiraEm.After(agora) {
			continue
		}
		compra.Status = compras.StatusCompraExpirada
		for indice := range compra.Pedidos {
			compra.Pedidos[indice].Status = compras.StatusPedidoExpirado
			for _, item := range compra.Pedidos[indice].Itens {
				anuncio := r.anuncios[item.IDAnuncio]
				if anuncio.Status == anuncios.StatusAnuncioReservado {
					anuncio.Status = anuncios.StatusAnuncioDisponivel
					anuncio.AtualizadoEm = agora
					r.anuncios[item.IDAnuncio] = anuncio
				}
			}
		}
		r.comprasPorChave[chave] = compra
		r.pedidosPorComprador[compra.IDComprador] = append([]compras.Pedido(nil), compra.Pedidos...)
		afetadas++
	}
	return afetadas, nil
}

func (r *Store) ListarPedidosDoComprador(ctx context.Context, idComprador string) ([]compras.Pedido, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]compras.Pedido(nil), r.pedidosPorComprador[idComprador]...), nil
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
