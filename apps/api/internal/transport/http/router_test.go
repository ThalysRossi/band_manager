package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	applicationcalendar "github.com/thalys/band-manager/apps/api/internal/application/calendar"
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

func TestCalendarEventsRouteReturnsEvents(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/calendar-events?from=2026-05-01&to=2026-05-31", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body struct {
		Range struct {
			Timezone string `json:"timezone"`
		} `json:"range"`
		Events []struct {
			Title string `json:"title"`
		} `json:"events"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode calendar events response: %v", err)
	}

	if body.Range.Timezone != "America/Recife" {
		t.Fatalf("expected calendar timezone, got %q", body.Range.Timezone)
	}
	if len(body.Events) != 1 {
		t.Fatalf("expected one calendar event, got %d", len(body.Events))
	}
}

func TestCalendarEventsRouteRejectsUnauthenticatedRequest(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/calendar-events?from=2026-05-01&to=2026-05-31", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestCalendarEventsRouteRejectsInvalidQueryParams(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/calendar-events?from=2026-05-31&to=2026-05-01", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestCalendarEventsRouteRejectsViewerWrite(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodPost, "/calendar-events", strings.NewReader(validCalendarEventRequestBody()))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Idempotency-Key", "idem_calendar_1")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestCalendarEventsRouteReturnsNotFound(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/calendar-events/40400000-0000-0000-0000-000000000000", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
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

type testCalendarRepository struct{}

func (repository testCalendarRepository) ListEvents(ctx context.Context, query applicationcalendar.ListEventsQuery) ([]applicationcalendar.Event, error) {
	location := time.FixedZone("America/Recife", -3*60*60)
	return []applicationcalendar.Event{
		{
			ID:            "11111111-1111-1111-1111-111111111111",
			BandID:        query.Account.BandID,
			Type:          applicationcalendar.EventTypeShow,
			Title:         "Show em Recife",
			StartsAtLocal: time.Date(2026, 5, 10, 20, 0, 0, 0, location),
			EndsAtLocal:   time.Date(2026, 5, 10, 22, 0, 0, 0, location),
			Timezone:      "America/Recife",
			Recurrence: applicationcalendar.Recurrence{
				Frequency: applicationcalendar.RecurrenceFrequencyNone,
			},
		},
	}, nil
}

func (repository testCalendarRepository) GetEvent(ctx context.Context, query applicationcalendar.GetEventQuery) (applicationcalendar.Event, error) {
	if query.EventID == "40400000-0000-0000-0000-000000000000" {
		return applicationcalendar.Event{}, applicationcalendar.ErrCalendarEventNotFound
	}

	location := time.FixedZone("America/Recife", -3*60*60)
	return applicationcalendar.Event{
		ID:            query.EventID,
		BandID:        query.Account.BandID,
		Type:          applicationcalendar.EventTypeShow,
		Title:         "Show em Recife",
		StartsAtLocal: time.Date(2026, 5, 10, 20, 0, 0, 0, location),
		EndsAtLocal:   time.Date(2026, 5, 10, 22, 0, 0, 0, location),
		Timezone:      "America/Recife",
		Recurrence: applicationcalendar.Recurrence{
			Frequency: applicationcalendar.RecurrenceFrequencyNone,
		},
	}, nil
}

func (repository testCalendarRepository) CreateEvent(ctx context.Context, command applicationcalendar.CreateEventCommand) (applicationcalendar.Event, error) {
	return applicationcalendar.Event{
		ID:            "11111111-1111-1111-1111-111111111111",
		BandID:        command.Account.BandID,
		Type:          command.Type,
		Title:         command.Title,
		StartsAtLocal: command.StartsAtLocal,
		EndsAtLocal:   command.EndsAtLocal,
		Timezone:      command.Account.BandTimezone,
		Recurrence:    command.Recurrence,
	}, nil
}

func (repository testCalendarRepository) UpdateEvent(ctx context.Context, command applicationcalendar.UpdateEventCommand) (applicationcalendar.Event, error) {
	return applicationcalendar.Event{
		ID:            command.EventID,
		BandID:        command.Account.BandID,
		Type:          command.Type,
		Title:         command.Title,
		StartsAtLocal: command.StartsAtLocal,
		EndsAtLocal:   command.EndsAtLocal,
		Timezone:      command.Account.BandTimezone,
		Recurrence:    command.Recurrence,
	}, nil
}

func (repository testCalendarRepository) SoftDeleteEvent(ctx context.Context, command applicationcalendar.SoftDeleteEventCommand) error {
	return nil
}

func testDependencies() Dependencies {
	return Dependencies{
		Authenticator:              testAuthenticator{},
		AccountRepository:          testAccountRepository{},
		InventoryRepository:        testInventoryRepository{},
		MerchBoothRepository:       testMerchBoothRepository{},
		FinancialReportsRepository: testFinancialReportsRepository{},
		CalendarRepository:         testCalendarRepository{},
		PaymentProvider:            testPaymentProvider{},
	}
}

func validCalendarEventRequestBody() string {
	return `{
		"type": "show",
		"title": "Show em Recife",
		"description": "Set de 45 minutos",
		"locationName": "Casa de Shows",
		"address": "Rua Principal, 123",
		"startsAtLocal": "2026-05-10T20:00:00",
		"endsAtLocal": "2026-05-10T22:00:00",
		"recurrence": {
			"frequency": "none"
		}
	}`
}
