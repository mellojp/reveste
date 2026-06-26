package common

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	HTTPAddress      string
	VercelBlobToken  string
	BlobPublicHost   string
	ConfiarProxy     bool
	IntervaloJobs    time.Duration
	MelhorEnvioToken string
	MelhorEnvioURL   string
	MelhorEnvioUA    string

	MercadoPagoToken          string
	MercadoPagoWebhookSecret  string
	MercadoPagoURL            string
	MercadoPagoNotificacaoURL string
	MercadoPagoPublicKey      string
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
		ConfiarProxy:    proxyConfiavel(os.Getenv("TRUST_PROXY")),
		IntervaloJobs:   time.Minute,
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
	if valor := strings.TrimSpace(os.Getenv("JOBS_INTERVAL")); valor != "" {
		intervalo, err := time.ParseDuration(valor)
		if err != nil || intervalo <= 0 {
			return Config{}, fmt.Errorf("JOBS_INTERVAL deve ser uma duracao positiva")
		}
		cfg.IntervaloJobs = intervalo
	}

	// Cotacao de frete via Melhor Envio. Sem token, o checkout usa o frete de contingencia.
	cfg.MelhorEnvioToken = strings.TrimSpace(os.Getenv("MELHORENVIO_TOKEN"))
	cfg.MelhorEnvioURL = strings.TrimSpace(os.Getenv("MELHORENVIO_URL"))
	if cfg.MelhorEnvioURL == "" {
		cfg.MelhorEnvioURL = "https://sandbox.melhorenvio.com.br"
	}
	cfg.MelhorEnvioUA = strings.TrimSpace(os.Getenv("MELHORENVIO_USER_AGENT"))
	if cfg.MelhorEnvioUA == "" {
		cfg.MelhorEnvioUA = "ReVeste (contato@reveste.com.br)"
	}

	// Pagamento via Mercado Pago. Sem MERCADOPAGO_ACCESS_TOKEN, o checkout usa o provedor
	// simulado (sincrono) e o webhook nao e exposto.
	cfg.MercadoPagoToken = strings.TrimSpace(os.Getenv("MERCADOPAGO_ACCESS_TOKEN"))
	cfg.MercadoPagoWebhookSecret = strings.TrimSpace(os.Getenv("MERCADOPAGO_WEBHOOK_SECRET"))
	cfg.MercadoPagoURL = strings.TrimSpace(os.Getenv("MERCADOPAGO_URL"))
	if cfg.MercadoPagoURL == "" {
		cfg.MercadoPagoURL = "https://api.mercadopago.com"
	}
	cfg.MercadoPagoNotificacaoURL = strings.TrimSpace(os.Getenv("MERCADOPAGO_NOTIFICATION_URL"))
	// Chave publica usada no frontend (SDK MercadoPago.js) para tokenizar o cartao com seguranca
	// (PCI). E publica por natureza; nao confundir com o Access Token.
	cfg.MercadoPagoPublicKey = strings.TrimSpace(os.Getenv("MERCADOPAGO_PUBLIC_KEY"))
	return cfg, nil
}

func proxyConfiavel(valor string) bool {
	switch strings.ToLower(strings.TrimSpace(valor)) {
	case "1", "true", "sim", "yes":
		return true
	default:
		return false
	}
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
