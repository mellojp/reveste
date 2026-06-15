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
	httptransport "reveste/apps/api/internal/http"
	"reveste/apps/api/internal/storage/postgres"
	"reveste/apps/api/internal/storage/vercel"
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
	paginasHTML, err := web.NovoAdaptadorPaginas(
		controladorCadastros,
		controladorAnuncios,
		controladorCarrinho,
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
			database,
			logger,
			cfg.BlobPublicHost,
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
