package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Environment    string
	Address        string
	AllowedOrigins []string
	DatabaseURL    string
	RedisURL       string
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

	allowedOrigins, err := parseAllowedOrigins(originsValue)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Environment:    environment,
		Address:        address,
		AllowedOrigins: allowedOrigins,
		DatabaseURL:    databaseURL,
		RedisURL:       redisURL,
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
