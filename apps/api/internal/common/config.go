package common

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	HTTPAddress string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddress: os.Getenv("HTTP_ADDRESS"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL nao foi definida no arquivo .env ou no ambiente")
	}
	if cfg.HTTPAddress == "" {
		cfg.HTTPAddress = ":8080"
	}
	return cfg, nil
}
