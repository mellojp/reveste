package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	casosdeusoanuncios "reveste/apps/api/internal/casosdeuso/anuncios"
	casosdeusocadastros "reveste/apps/api/internal/casosdeuso/cadastros"
	casosdeusocompras "reveste/apps/api/internal/casosdeuso/compras"
	"reveste/apps/api/internal/common"
	"reveste/apps/api/internal/database/postgres"
	httptransport "reveste/apps/api/internal/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := common.Load()
	if err != nil {
		logger.Error("configuracao invalida", "erro", err)
		os.Exit(1)
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	database, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("falha ao iniciar banco de dados", "erro", err)
		os.Exit(1)
	}
	defer database.Close()

	//controller de cadstros
	cadastros := casosdeusocadastros.NovoFluxoCadastro(
		database,
		database,
		common.GeradorIDCriptografico{},
		common.ProcessadorPBKDF2{Iteracoes: 210_000},
		common.RelogioSistema{},
	)
	//controller de anuncios
	anuncios := casosdeusoanuncios.NovoFluxoAnuncio(
		database,
		database,
		common.GeradorIDCriptografico{},
		common.RelogioSistema{},
	)
	//controller de compras
	compras := casosdeusocompras.NovoFluxoCarrinho(
		database,
		database,
		common.GeradorIDCriptografico{},
		common.RelogioSistema{},
	)

	servidor := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           httptransport.NovaAPI(cadastros, anuncios, compras, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("api iniciada", "endereco", cfg.HTTPAddress)
	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("api encerrada com erro", "erro", err)
		os.Exit(1)
	}
}
