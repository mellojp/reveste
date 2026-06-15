package common

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL     string
	HTTPAddress     string
	VercelBlobToken string
	BlobPublicHost  string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	cfg := Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		HTTPAddress:     os.Getenv("HTTP_ADDRESS"),
		VercelBlobToken: os.Getenv("BLOB_READ_WRITE_TOKEN"),
		BlobPublicHost:  strings.ToLower(strings.TrimSpace(os.Getenv("BLOB_PUBLIC_HOST"))),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL nao foi definida no arquivo .env ou no ambiente")
	}
	if cfg.HTTPAddress == "" {
		cfg.HTTPAddress = ":8080"
	}
	if cfg.BlobPublicHost == "" {
		cfg.BlobPublicHost = hostBlobDoToken(cfg.VercelBlobToken)
	}
	if cfg.VercelBlobToken != "" && !hostBlobValido(cfg.BlobPublicHost) {
		return Config{}, fmt.Errorf(
			"BLOB_PUBLIC_HOST deve ser o hostname do store publico da Vercel Blob",
		)
	}
	return cfg, nil
}

func hostBlobDoToken(token string) string {
	partes := strings.Split(strings.TrimSpace(token), "_")
	if len(partes) < 4 || partes[3] == "" {
		return ""
	}
	return strings.ToLower(partes[3]) + ".public.blob.vercel-storage.com"
}

func hostBlobValido(host string) bool {
	return host != "" && net.ParseIP(host) == nil &&
		strings.HasSuffix(host, ".public.blob.vercel-storage.com") &&
		!strings.Contains(host, "..") &&
		!strings.ContainsAny(host, "/:@?#* \t\r\n")
}
