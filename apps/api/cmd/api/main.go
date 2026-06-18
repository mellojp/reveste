package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reveste/apps/api/internal/casosdeuso"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/dominio/compras"
	httptransport "reveste/apps/api/internal/http"
	"reveste/apps/api/internal/storage/pagamentos"
	"reveste/apps/api/internal/storage/postgres"
	"reveste/apps/api/internal/storage/vercel"
	"reveste/apps/api/internal/transporte"
	"reveste/apps/api/internal/web"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := executar(logger); err != nil {
		logger.Error("api encerrada com erro", "erro", err)
		os.Exit(1)
	}
}

func executar(logger *slog.Logger) error {
	cfg, err := common.Load()
	if err != nil {
		return fmt.Errorf("carregar configuracao: %w", err)
	}

	ctxInicializacao, cancelarInicializacao := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelarInicializacao()
	database, err := postgres.Open(ctxInicializacao, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("iniciar banco de dados: %w", err)
	}
	defer database.Close()

	controladorCadastros := casosdeuso.NovoControladorCadastro(
		database,
		database,
		common.GeradorIDCriptografico{},
		common.ProcessadorPBKDF2{Iteracoes: 210_000},
		common.RelogioSistema{},
	)
	controladorAnuncios := casosdeuso.NovoControladorAnuncio(
		database,
		database,
		common.GeradorIDCriptografico{},
		common.RelogioSistema{},
		cfg.BlobPublicHost,
	)
	controladorCarrinho := casosdeuso.NovoControladorCarrinho(
		database,
		database,
		common.GeradorIDCriptografico{},
		common.RelogioSistema{},
	)
	controladorUpload := casosdeuso.NovoControladorUpload(
		vercel.Novo(cfg.VercelBlobToken),
		common.GeradorIDCriptografico{},
		common.RelogioSistema{},
	)
	controladorCheckout := casosdeuso.NovoControladorCheckout(
		database,
		database,
		database,
		database,
		pagamentos.NovoSimulado(),
		common.GeradorIDCriptografico{},
		common.RelogioSistema{},
		compras.PoliticaCobranca{TaxaServicoPercentual: 10, FretePorPedidoCentavos: 1990},
	)
	controladorPedidos := casosdeuso.NovoControladorPedidos(
		database,
		common.GeradorIDCriptografico{},
		common.RelogioSistema{},
	)
	limitadorLogin := transporte.NovoLimitadorLogin(database)
	paginasHTML, err := web.NovoAdaptadorPaginas(
		controladorCadastros,
		controladorAnuncios,
		controladorCarrinho,
		controladorCheckout,
		controladorPedidos,
		limitadorLogin,
		cfg.ConfiarProxy,
		logger,
	)
	if err != nil {
		return fmt.Errorf("iniciar frontend: %w", err)
	}

	servidor := &http.Server{
		Addr: cfg.HTTPAddress,
		Handler: httptransport.NovaAPI(
			controladorCadastros,
			controladorAnuncios,
			controladorCarrinho,
			controladorUpload,
			controladorCheckout,
			controladorPedidos,
			database,
			logger,
			cfg.BlobPublicHost,
			limitadorLogin,
			cfg.ConfiarProxy,
			paginasHTML,
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("api iniciada", "endereco", cfg.HTTPAddress)
	errosServidor := make(chan error, 1)
	go func() {
		errosServidor <- servidor.ListenAndServe()
	}()

	ctxEncerramento, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()
	go executarJobsPeriodicos(
		ctxEncerramento, logger, cfg.IntervaloJobs, controladorCheckout, controladorPedidos,
	)
	select {
	case err := <-errosServidor:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctxEncerramento.Done():
	}

	ctxShutdown, cancelarShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelarShutdown()
	if err := servidor.Shutdown(ctxShutdown); err != nil {
		return fmt.Errorf("encerrar servidor HTTP: %w", err)
	}
	return nil
}

func executarJobsPeriodicos(
	ctx context.Context,
	logger *slog.Logger,
	intervalo time.Duration,
	checkout *casosdeuso.ControladorCheckout,
	pedidos *casosdeuso.ControladorPedidos,
) {
	executar := func() {
		ctxJob, cancelar := context.WithTimeout(ctx, 30*time.Second)
		defer cancelar()

		expiradas, err := checkout.ProcessarExpiracoes(ctxJob)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("falha ao expirar compras pendentes", "erro", err)
		} else if expiradas > 0 {
			logger.Info("compras pendentes expiradas", "quantidade", expiradas)
		}

		naoEnviados, err := pedidos.ProcessarPrazosEnvio(ctxJob)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("falha ao processar prazos de envio", "erro", err)
		} else if naoEnviados > 0 {
			logger.Info("itens marcados como nao enviados", "quantidade", naoEnviados)
		}
	}

	executar()
	ticker := time.NewTicker(intervalo)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			executar()
		}
	}
}
