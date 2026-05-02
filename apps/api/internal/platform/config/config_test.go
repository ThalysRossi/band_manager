package config

import "testing"

func TestLoadFromEnvironmentRejectsWildcardOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("API_ADDR", ":8080")
	t.Setenv("API_ALLOWED_ORIGINS", "*")
	t.Setenv("DATABASE_URL", "postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("SUPABASE_JWT_SECRET", "secret")
	t.Setenv("MERCADOPAGO_ACCESS_TOKEN", "token")
	t.Setenv("MERCADOPAGO_WEBHOOK_SECRET", "secret")
	t.Setenv("MERCADOPAGO_POINT_TERMINAL_ID", "terminal")

	_, err := LoadFromEnvironment()
	if err == nil {
		t.Fatal("expected wildcard origin error")
	}
}
