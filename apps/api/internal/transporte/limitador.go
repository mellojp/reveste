package transporte

import (
	"context"
	"sync"
	"time"
)

const (
	maxTentativasLogin = 5
	janelaLogin        = time.Minute
)

// RegistroTentativas persiste as tentativas de login por chave (IP), permitindo que o
// limite sobreviva a reinicios e seja compartilhado entre instancias.
type RegistroTentativas interface {
	ContarTentativas(ctx context.Context, chave string, desde time.Time) (int, error)
	RegistrarTentativa(ctx context.Context, chave string, em time.Time) error
	LimparTentativas(ctx context.Context, chave string) error
}

// LimitadorLogin aplica a janela e o teto de tentativas sobre um RegistroTentativas.
// Em caso de falha de armazenamento, libera a tentativa (fail-open) para nao trancar
// usuarios legitimos por uma indisponibilidade do backing store.
type LimitadorLogin struct {
	registro RegistroTentativas
	max      int
	janela   time.Duration
}

func NovoLimitadorLogin(registro RegistroTentativas) *LimitadorLogin {
	return &LimitadorLogin{registro: registro, max: maxTentativasLogin, janela: janelaLogin}
}

func (l *LimitadorLogin) Permitido(ctx context.Context, chave string) bool {
	desde := time.Now().UTC().Add(-l.janela)
	quantidade, err := l.registro.ContarTentativas(ctx, chave, desde)
	if err != nil {
		return true
	}
	return quantidade < l.max
}

func (l *LimitadorLogin) RegistrarFalha(ctx context.Context, chave string) {
	_ = l.registro.RegistrarTentativa(ctx, chave, time.Now().UTC())
}

func (l *LimitadorLogin) Limpar(ctx context.Context, chave string) {
	_ = l.registro.LimparTentativas(ctx, chave)
}

// RegistroMemoria e uma implementacao em memoria de RegistroTentativas, util para testes
// e para execucao sem banco. Nao sobrevive a reinicios nem e compartilhada entre instancias.
type RegistroMemoria struct {
	mu         sync.Mutex
	tentativas map[string][]time.Time
}

func NovoRegistroMemoria() *RegistroMemoria {
	return &RegistroMemoria{tentativas: make(map[string][]time.Time)}
}

func (r *RegistroMemoria) ContarTentativas(_ context.Context, chave string, desde time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	quantidade := 0
	for _, em := range r.tentativas[chave] {
		if em.After(desde) {
			quantidade++
		}
	}
	return quantidade, nil
}

func (r *RegistroMemoria) RegistrarTentativa(_ context.Context, chave string, em time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.tentativas) >= 10_000 {
		r.tentativas = make(map[string][]time.Time)
	}
	r.tentativas[chave] = append(r.tentativas[chave], em)
	return nil
}

func (r *RegistroMemoria) LimparTentativas(_ context.Context, chave string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tentativas, chave)
	return nil
}
