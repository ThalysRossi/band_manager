package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Environment              string
	Address                  string
	AllowedOrigins           []string
	DatabaseURL              string
	RedisURL                 string
	SupabaseJWTSecret        string
	MercadoPagoAccessToken   string
	MercadoPagoWebhookSecret string
}

func LoadFromEnvironment() (Config, error) {
	environment, err := requiredEnv("APP_ENV")
	if err != nil {
		return Config{}, err
	}

	address, err := requiredEnv("API_ADDR")
	if err != nil {
		return Config{}, err
	}

	originsValue, err := requiredEnv("API_ALLOWED_ORIGINS")
	if err != nil {
		return Config{}, err
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	redisURL, err := requiredEnv("REDIS_URL")
	if err != nil {
		return Config{}, err
	}

	supabaseJWTSecret, err := requiredEnv("SUPABASE_JWT_SECRET")
	if err != nil {
		return Config{}, err
	}

	mercadoPagoAccessToken, err := requiredEnv("MERCADOPAGO_ACCESS_TOKEN")
	if err != nil {
		return Config{}, err
	}

	mercadoPagoWebhookSecret, err := requiredEnv("MERCADOPAGO_WEBHOOK_SECRET")
	if err != nil {
		return Config{}, err
	}

	allowedOrigins, err := parseAllowedOrigins(originsValue)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Environment:              environment,
		Address:                  address,
		AllowedOrigins:           allowedOrigins,
		DatabaseURL:              databaseURL,
		RedisURL:                 redisURL,
		SupabaseJWTSecret:        supabaseJWTSecret,
		MercadoPagoAccessToken:   mercadoPagoAccessToken,
		MercadoPagoWebhookSecret: mercadoPagoWebhookSecret,
	}, nil
}

func requiredEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}

	return value, nil
}

func parseAllowedOrigins(value string) ([]string, error) {
	rawOrigins := strings.Split(value, ",")
	allowedOrigins := make([]string, 0, len(rawOrigins))

	for _, rawOrigin := range rawOrigins {
		origin := strings.TrimSpace(rawOrigin)
		if origin == "" {
			continue
		}

		if origin == "*" {
			return nil, errors.New("API_ALLOWED_ORIGINS cannot contain wildcard origin")
		}

		allowedOrigins = append(allowedOrigins, origin)
	}

	if len(allowedOrigins) == 0 {
		return nil, errors.New("API_ALLOWED_ORIGINS must contain at least one origin")
	}

	return allowedOrigins, nil
}
