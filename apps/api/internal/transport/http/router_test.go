package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	applicationfinancialreports "github.com/thalys/band-manager/apps/api/internal/application/financialreports"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	applicationmerchbooth "github.com/thalys/band-manager/apps/api/internal/application/merchbooth"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
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

func TestFinancialReportsRouteReturnsReport(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/financial-reports?from=2026-05-01&to=2026-05-02", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body struct {
		Range struct {
			From     string `json:"from"`
			To       string `json:"to"`
			Timezone string `json:"timezone"`
		} `json:"range"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode financial report response: %v", err)
	}

	if body.Range.Timezone != "America/Recife" {
		t.Fatalf("expected report timezone, got %q", body.Range.Timezone)
	}
}

func TestFinancialReportsRouteRejectsUnauthenticatedRequest(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/financial-reports", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestFinancialReportsRouteRejectsInvalidQueryParams(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/financial-reports?from=2026-05-03&to=2026-05-01", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func testConfig() config.Config {
	return config.Config{
		Environment:                "test",
		Address:                    ":8080",
		AllowedOrigins:             []string{"http://localhost:5173"},
		DatabaseURL:                "postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable",
		RedisURL:                   "redis://localhost:6379/0",
		SupabaseJWTSecret:          "secret",
		MercadoPagoAccessToken:     "token",
		MercadoPagoWebhookSecret:   "webhook_secret",
		MercadoPagoPointTerminalID: "terminal",
	}
}

type testAuthenticator struct{}

func (authenticator testAuthenticator) Authenticate(ctx context.Context, bearerToken string) (session.AuthenticatedUser, error) {
	if bearerToken != "valid-token" {
		return session.AuthenticatedUser{}, errors.New("invalid bearer token")
	}

	return session.AuthenticatedUser{
		Provider:       "supabase",
		ProviderUserID: "auth_user_1",
		Email:          "band@example.com",
	}, nil
}

type testAccountRepository struct{}

func (repository testAccountRepository) CreateOwnerAccount(ctx context.Context, command accounts.CreateOwnerAccountCommand) (accounts.OwnerAccount, error) {
	return accounts.OwnerAccount{}, nil
}

func (repository testAccountRepository) GetCurrentAccount(ctx context.Context, query accounts.CurrentAccountQuery) (accounts.OwnerAccount, error) {
	return accounts.OwnerAccount{
		UserID:       "00000000-0000-0000-0000-000000000001",
		BandID:       "00000000-0000-0000-0000-000000000002",
		Email:        "band@example.com",
		BandName:     "Os Testes",
		BandTimezone: "America/Recife",
		Role:         permissions.RoleViewer,
	}, nil
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

type testMerchBoothRepository struct{}

func (repository testMerchBoothRepository) ListBoothItems(ctx context.Context, query applicationmerchbooth.ListBoothItemsQuery) ([]applicationmerchbooth.BoothItem, error) {
	return nil, nil
}

func (repository testMerchBoothRepository) CreateCashCheckout(ctx context.Context, command applicationmerchbooth.CreateCashCheckoutCommand) (applicationmerchbooth.Sale, error) {
	return applicationmerchbooth.Sale{}, nil
}

func (repository testMerchBoothRepository) ReservePixCheckout(ctx context.Context, command applicationmerchbooth.CreatePixCheckoutCommand) (applicationmerchbooth.Sale, bool, error) {
	return applicationmerchbooth.Sale{}, false, nil
}

func (repository testMerchBoothRepository) ReserveCardCheckout(ctx context.Context, command applicationmerchbooth.CreateCardCheckoutCommand) (applicationmerchbooth.Sale, bool, error) {
	return applicationmerchbooth.Sale{}, false, nil
}

func (repository testMerchBoothRepository) CompletePixCheckoutPayment(ctx context.Context, command applicationmerchbooth.CompletePixCheckoutPaymentCommand) (applicationmerchbooth.Sale, error) {
	return applicationmerchbooth.Sale{}, nil
}

func (repository testMerchBoothRepository) CompleteCardCheckoutPayment(ctx context.Context, command applicationmerchbooth.CompleteCardCheckoutPaymentCommand) (applicationmerchbooth.Sale, error) {
	return applicationmerchbooth.Sale{}, nil
}

func (repository testMerchBoothRepository) FailPixCheckoutPaymentCreation(ctx context.Context, command applicationmerchbooth.FailPixCheckoutPaymentCreationCommand) error {
	return nil
}

func (repository testMerchBoothRepository) FailCardCheckoutPaymentCreation(ctx context.Context, command applicationmerchbooth.FailCardCheckoutPaymentCreationCommand) error {
	return nil
}

func (repository testMerchBoothRepository) GetPixPaymentProviderOrderID(ctx context.Context, query applicationmerchbooth.GetPixPaymentProviderOrderIDQuery) (string, error) {
	return "order_1", nil
}

func (repository testMerchBoothRepository) ApplyPixPaymentStatus(ctx context.Context, command applicationmerchbooth.ApplyPixPaymentStatusCommand) (applicationmerchbooth.Sale, error) {
	return applicationmerchbooth.Sale{}, nil
}

func (repository testMerchBoothRepository) RecordPaymentEvent(ctx context.Context, command applicationmerchbooth.PaymentEventCommand) error {
	return nil
}

type testPaymentProvider struct{}

func (provider testPaymentProvider) CreatePixPayment(ctx context.Context, command applicationmerchbooth.CreatePixPaymentCommand) (applicationmerchbooth.PixPayment, error) {
	return applicationmerchbooth.PixPayment{}, nil
}

func (provider testPaymentProvider) CreateCardPayment(ctx context.Context, command applicationmerchbooth.CreateCardPaymentCommand) (applicationmerchbooth.PixPayment, error) {
	return applicationmerchbooth.PixPayment{}, nil
}

func (provider testPaymentProvider) GetPaymentStatus(ctx context.Context, command applicationmerchbooth.GetPaymentStatusCommand) (applicationmerchbooth.PixPayment, error) {
	return applicationmerchbooth.PixPayment{}, nil
}

type testFinancialReportsRepository struct{}

func (repository testFinancialReportsRepository) GetReport(ctx context.Context, query applicationfinancialreports.ReportQuery) (applicationfinancialreports.Report, error) {
	return applicationfinancialreports.Report{
		Range: query.Range,
		Summary: applicationfinancialreports.ReportSummary{
			SaleCount:           1,
			ItemCount:           2,
			GrossRevenue:        inventorydomain.Money{Amount: 10000, Currency: "BRL"},
			TotalHistoricalCost: inventorydomain.Money{Amount: 4000, Currency: "BRL"},
			ExpectedProfit:      inventorydomain.Money{Amount: 6000, Currency: "BRL"},
		},
	}, nil
}

func testDependencies() Dependencies {
	return Dependencies{
		Authenticator:              testAuthenticator{},
		AccountRepository:          testAccountRepository{},
		InventoryRepository:        testInventoryRepository{},
		MerchBoothRepository:       testMerchBoothRepository{},
		FinancialReportsRepository: testFinancialReportsRepository{},
		PaymentProvider:            testPaymentProvider{},
	}
}
