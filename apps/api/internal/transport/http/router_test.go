package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	"github.com/thalys/band-manager/apps/api/internal/platform/config"
)

func TestHealthRouteReturnsOK(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body healthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %s", body.Status)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}

	allowOrigin := response.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:5173" {
		t.Fatalf("expected allowed origin header, got %s", allowOrigin)
	}
}

func testConfig() config.Config {
	return config.Config{
		Environment:       "test",
		Address:           ":8080",
		AllowedOrigins:    []string{"http://localhost:5173"},
		DatabaseURL:       "postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable",
		RedisURL:          "redis://localhost:6379/0",
		SupabaseJWTSecret: "secret",
	}
}

type testAuthenticator struct{}

func (authenticator testAuthenticator) Authenticate(ctx context.Context, bearerToken string) (session.AuthenticatedUser, error) {
	return session.AuthenticatedUser{}, nil
}

type testAccountRepository struct{}

func (repository testAccountRepository) CreateOwnerAccount(ctx context.Context, command accounts.CreateOwnerAccountCommand) (accounts.OwnerAccount, error) {
	return accounts.OwnerAccount{}, nil
}

func (repository testAccountRepository) GetCurrentAccount(ctx context.Context, query accounts.CurrentAccountQuery) (accounts.OwnerAccount, error) {
	return accounts.OwnerAccount{}, nil
}

type testInventoryRepository struct{}

func (repository testInventoryRepository) CreateProduct(ctx context.Context, command applicationinventory.CreateProductCommand) (applicationinventory.Product, error) {
	return applicationinventory.Product{}, nil
}

func (repository testInventoryRepository) ListInventory(ctx context.Context, query applicationinventory.ListInventoryQuery) ([]applicationinventory.Product, error) {
	return nil, nil
}

func (repository testInventoryRepository) UpdateProduct(ctx context.Context, command applicationinventory.UpdateProductCommand) (applicationinventory.Product, error) {
	return applicationinventory.Product{}, nil
}

func (repository testInventoryRepository) UpdateVariant(ctx context.Context, command applicationinventory.UpdateVariantCommand) (applicationinventory.Variant, error) {
	return applicationinventory.Variant{}, nil
}

func (repository testInventoryRepository) SoftDeleteProduct(ctx context.Context, command applicationinventory.SoftDeleteProductCommand) error {
	return nil
}

func (repository testInventoryRepository) SoftDeleteVariant(ctx context.Context, command applicationinventory.SoftDeleteVariantCommand) error {
	return nil
}

func testDependencies() Dependencies {
	return Dependencies{
		Authenticator:       testAuthenticator{},
		AccountRepository:   testAccountRepository{},
		InventoryRepository: testInventoryRepository{},
	}
}
