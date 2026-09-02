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

func TestCurrentAccountRouteReturnsNotFoundBeforeOnboarding(t *testing.T) {
	t.Parallel()

	dependencies := testDependencies()
	dependencies.AccountRepository = testAccountRepository{err: accounts.ErrAccountNotFound}
	router := NewRouter(testConfig(), slog.Default(), dependencies)
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode authentication error response: %v", err)
	}
	if body.Code != "account_not_found" {
		t.Fatalf("expected account_not_found code, got %q", body.Code)
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

func TestInventoryPhotoUploadRequestReturnsSignedTargets(t *testing.T) {
	t.Parallel()

	dependencies := testDependencies()
	dependencies.AccountRepository = testAccountRepository{role: permissions.RoleOwner}
	router := NewRouter(testConfig(), slog.Default(), dependencies)
	request := httptest.NewRequest(http.MethodPost, "/inventory/photos/upload-requests", strings.NewReader(`{
		"full": {
			"contentType": "image/webp",
			"sizeBytes": 1024,
			"width": 1200,
			"height": 900
		},
		"display": {
			"contentType": "image/webp",
			"sizeBytes": 512,
			"width": 1280,
			"height": 960
		}
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}

	var body struct {
		Photo struct {
			Full struct {
				ObjectKey string `json:"objectKey"`
			} `json:"full"`
		} `json:"photo"`
		Uploads struct {
			Full struct {
				Token string `json:"token"`
			} `json:"full"`
		} `json:"uploads"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	if body.Photo.Full.ObjectKey == "" {
		t.Fatal("expected generated full photo object key")
	}
	if body.Uploads.Full.Token != "upload-token" {
		t.Fatalf("expected upload token, got %q", body.Uploads.Full.Token)
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

func TestAccountMembersRouteRejectsUnauthenticatedRequest(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/account/members", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestAccountMembersRouteReturnsMembers(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodGet, "/account/members", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body struct {
		Members []struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"members"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode members response: %v", err)
	}

	if len(body.Members) != 1 {
		t.Fatalf("expected one member, got %d", len(body.Members))
	}
	if body.Members[0].Email != "band@example.com" {
		t.Fatalf("expected member email, got %q", body.Members[0].Email)
	}
}

func TestAccountInviteCreateReturnsTokenForOwner(t *testing.T) {
	t.Parallel()

	dependencies := testDependencies()
	dependencies.AccountRepository = testAccountRepository{role: permissions.RoleOwner}
	router := NewRouter(testConfig(), slog.Default(), dependencies)
	request := httptest.NewRequest(http.MethodPost, "/account/invites", strings.NewReader(`{"email":"viewer@example.com"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Idempotency-Key", "idem_account_invite_1")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}

	var body struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode invite response: %v", err)
	}

	if body.Token == "" {
		t.Fatal("expected invite token")
	}
}

func TestAccountInviteMutationRejectsViewer(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodPost, "/account/invites", strings.NewReader(`{"email":"viewer@example.com"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Idempotency-Key", "idem_account_invite_1")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestAccountInviteAcceptRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	router := NewRouter(testConfig(), slog.Default(), testDependencies())
	request := httptest.NewRequest(http.MethodPost, "/account/invites/accept", strings.NewReader(`{"token":"missing"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Idempotency-Key", "idem_account_invite_accept")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestAccountOnboardingRejectsUnverifiedEmail(t *testing.T) {
	t.Parallel()

	dependencies := testDependencies()
	dependencies.VerifiedUserInspector = testVerifiedUserInspector{err: session.ErrEmailNotVerified}
	router := NewRouter(testConfig(), slog.Default(), dependencies)
	request := httptest.NewRequest(http.MethodPost, "/account/onboarding", strings.NewReader(`{"bandName":"Os Testes","bandTimezone":"America/Recife"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Idempotency-Key", "idem_onboarding")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestAccountInviteAcceptReturnsBadGatewayWhenIdentityProviderFails(t *testing.T) {
	t.Parallel()

	dependencies := testDependencies()
	dependencies.VerifiedUserInspector = testVerifiedUserInspector{err: errors.New("supabase unavailable")}
	router := NewRouter(testConfig(), slog.Default(), dependencies)
	request := httptest.NewRequest(http.MethodPost, "/account/invites/accept", strings.NewReader(`{"token":"token_accept"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Idempotency-Key", "idem_account_invite_accept")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, response.Code)
	}
}

func testConfig() config.Config {
	return config.Config{
		Environment:                "test",
		Address:                    ":8080",
		AllowedOrigins:             []string{"http://localhost:5173"},
		DatabaseURL:                "postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable",
		RedisURL:                   "redis://localhost:6379/0",
		SupabaseURL:                "https://example.supabase.co",
		SupabasePublishableKey:     "publishable-key",
		SupabaseSecretKey:          "secret-key",
		SupabaseStorageBucket:      "inventory-photos",
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

type testAccountRepository struct {
	role permissions.Role
	err  error
}

func (repository testAccountRepository) CreateOwnerAccount(ctx context.Context, command accounts.CreateOwnerAccountCommand) (accounts.OwnerAccount, error) {
	return accounts.OwnerAccount{}, nil
}

func (repository testAccountRepository) GetCurrentAccount(ctx context.Context, query accounts.CurrentAccountQuery) (accounts.OwnerAccount, error) {
	if repository.err != nil {
		return accounts.OwnerAccount{}, repository.err
	}

	role := repository.role
	if role == "" {
		role = permissions.RoleViewer
	}

	return accounts.OwnerAccount{
		UserID:       "00000000-0000-0000-0000-000000000001",
		BandID:       "00000000-0000-0000-0000-000000000002",
		Email:        "band@example.com",
		BandName:     "Os Testes",
		BandTimezone: "America/Recife",
		Role:         role,
	}, nil
}

func (repository testAccountRepository) ListBandMembers(ctx context.Context, query accounts.ListBandMembersQuery) ([]accounts.BandMember, error) {
	return []accounts.BandMember{
		{
			UserID:   query.Account.UserID,
			Email:    query.Account.Email,
			BandID:   query.Account.BandID,
			Role:     query.Account.Role,
			JoinedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		},
	}, nil
}

func (repository testAccountRepository) ListBandInvites(ctx context.Context, query accounts.ListBandInvitesQuery) ([]accounts.BandInvite, error) {
	return []accounts.BandInvite{
		{
			ID:        "11111111-1111-1111-1111-111111111111",
			BandID:    query.Account.BandID,
			Email:     "viewer@example.com",
			Role:      permissions.RoleViewer,
			Status:    accounts.InviteStatusPending,
			ExpiresAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
			CreatedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		},
	}, nil
}

func (repository testAccountRepository) CreateBandInvite(ctx context.Context, command accounts.CreateBandInviteCommand) (accounts.BandInvite, error) {
	return accounts.BandInvite{
		ID:        "11111111-1111-1111-1111-111111111111",
		BandID:    command.Account.BandID,
		Email:     command.Email,
		Role:      command.Role,
		Status:    command.Status,
		ExpiresAt: command.ExpiresAt,
		CreatedAt: command.CreatedAt,
		UpdatedAt: command.CreatedAt,
		Token:     command.Token,
	}, nil
}

func (repository testAccountRepository) RevokeBandInvite(ctx context.Context, command accounts.RevokeBandInviteCommand) (accounts.BandInvite, error) {
	return accounts.BandInvite{
		ID:        command.InviteID,
		BandID:    command.Account.BandID,
		Email:     "viewer@example.com",
		Role:      permissions.RoleViewer,
		Status:    accounts.InviteStatusRevoked,
		ExpiresAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: command.RevokedAt,
	}, nil
}

func (repository testAccountRepository) AcceptBandInvite(ctx context.Context, command accounts.AcceptBandInviteCommand) (accounts.BandMember, error) {
	if command.Token == "missing" {
		return accounts.BandMember{}, accounts.ErrInviteNotFound
	}

	return accounts.BandMember{
		UserID:   "00000000-0000-0000-0000-000000000003",
		Email:    command.Email,
		BandID:   "00000000-0000-0000-0000-000000000002",
		Role:     permissions.RoleViewer,
		JoinedAt: command.AcceptedAt,
	}, nil
}

type testInventoryRepository struct{}

func (repository testInventoryRepository) CreateProduct(ctx context.Context, command applicationinventory.CreateProductCommand) (applicationinventory.Product, error) {
	return applicationinventory.Product{}, nil
}

func (repository testInventoryRepository) CreateVariant(ctx context.Context, command applicationinventory.CreateVariantCommand) (applicationinventory.Variant, error) {
	return applicationinventory.Variant{}, nil
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

type testPhotoStorage struct{}

func (storage testPhotoStorage) CreateSignedUpload(ctx context.Context, command applicationinventory.CreatePhotoUploadCommand) (applicationinventory.SignedPhotoUpload, error) {
	return applicationinventory.SignedPhotoUpload{
		ObjectKey: command.ObjectKey,
		SignedURL: "https://example.supabase.co/storage/v1/object/upload/sign/inventory-photos/" + command.ObjectKey + "?token=upload-token",
		Token:     "upload-token",
		ExpiresAt: time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC),
		PublicURL: storage.PublicURL(command.ObjectKey),
	}, nil
}

func (storage testPhotoStorage) GetObjectInfo(ctx context.Context, query applicationinventory.PhotoObjectInfoQuery) (applicationinventory.PhotoObjectInfo, error) {
	return applicationinventory.PhotoObjectInfo{
		ObjectKey:   query.ObjectKey,
		ContentType: inventorydomain.PhotoContentTypeWebP,
		SizeBytes:   1024,
	}, nil
}

func (storage testPhotoStorage) PublicURL(objectKey string) string {
	return "https://example.supabase.co/storage/v1/object/public/inventory-photos/" + objectKey
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
		VerifiedUserInspector:      testVerifiedUserInspector{},
		AccountRepository:          testAccountRepository{},
		InventoryRepository:        testInventoryRepository{},
		PhotoStorage:               testPhotoStorage{},
		MerchBoothRepository:       testMerchBoothRepository{},
		FinancialReportsRepository: testFinancialReportsRepository{},
		CalendarRepository:         testCalendarRepository{},
		PaymentProvider:            testPaymentProvider{},
	}
}

type testVerifiedUserInspector struct {
	err error
}

func (inspector testVerifiedUserInspector) InspectVerifiedUser(ctx context.Context, bearerToken string) (session.VerifiedUser, error) {
	if inspector.err != nil {
		return session.VerifiedUser{}, inspector.err
	}

	return session.VerifiedUser{
		Provider:       "supabase",
		ProviderUserID: "auth_user_1",
		Email:          "band@example.com",
	}, nil
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
