package authhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware/authcontext"
)

func TestOnboardOwnerRequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	handler := testHandler()
	request := authenticatedRequest(t, http.MethodPost, "/account/onboarding", `{"bandName":"Os Testes","bandTimezone":"America/Recife"}`)
	response := httptest.NewRecorder()

	middleware.RequestID(http.HandlerFunc(handler.OnboardOwner)).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestOnboardOwnerCreatesAccount(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		account: accounts.OwnerAccount{
			UserID:       "user_1",
			BandID:       "band_1",
			Email:        "band@example.com",
			BandName:     "Os Testes",
			BandTimezone: "America/Recife",
			Role:         permissions.RoleOwner,
		},
	}
	handler := testHandlerWithRepository(repository)
	handler.now = func() time.Time {
		return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	}
	request := authenticatedRequest(t, http.MethodPost, "/account/onboarding", `{"bandName":"Os Testes","bandTimezone":"America/Recife"}`)
	request.Header.Set("Idempotency-Key", "idem_1")
	response := httptest.NewRecorder()

	middleware.RequestID(http.HandlerFunc(handler.OnboardOwner)).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}

	if repository.command.AuthProviderUserID != "auth_user_1" {
		t.Fatalf("expected provider user id from token, got %q", repository.command.AuthProviderUserID)
	}

	var body CurrentAccountResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !body.ActiveBand.CanWrite {
		t.Fatal("expected owner account to write")
	}
}

func TestGetCurrentAccountReturnsResolvedContext(t *testing.T) {
	t.Parallel()

	handler := testHandler()
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	ctx, err := authcontext.WithContext(request.Context(), authcontext.Context{
		UserID:       "user_1",
		BandID:       "band_1",
		Email:        "band@example.com",
		BandName:     "Os Testes",
		BandTimezone: "America/Recife",
		Role:         permissions.RoleOwner,
	})
	if err != nil {
		t.Fatalf("create account context: %v", err)
	}
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()

	handler.GetCurrentAccount(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

type fakeRepository struct {
	account accounts.OwnerAccount
	command accounts.CreateOwnerAccountCommand
	err     error
}

func (repository *fakeRepository) CreateOwnerAccount(ctx context.Context, command accounts.CreateOwnerAccountCommand) (accounts.OwnerAccount, error) {
	if ctx == nil {
		return accounts.OwnerAccount{}, errors.New("context is required")
	}
	repository.command = command
	if repository.err != nil {
		return accounts.OwnerAccount{}, repository.err
	}

	return repository.account, nil
}

func (repository *fakeRepository) GetCurrentAccount(ctx context.Context, query accounts.CurrentAccountQuery) (accounts.OwnerAccount, error) {
	if ctx == nil {
		return accounts.OwnerAccount{}, errors.New("context is required")
	}
	if repository.err != nil {
		return accounts.OwnerAccount{}, repository.err
	}

	return repository.account, nil
}

func (repository *fakeRepository) ListBandMembers(ctx context.Context, query accounts.ListBandMembersQuery) ([]accounts.BandMember, error) {
	return nil, nil
}

func (repository *fakeRepository) ListBandInvites(ctx context.Context, query accounts.ListBandInvitesQuery) ([]accounts.BandInvite, error) {
	return nil, nil
}

func (repository *fakeRepository) CreateBandInvite(ctx context.Context, command accounts.CreateBandInviteCommand) (accounts.BandInvite, error) {
	return accounts.BandInvite{}, nil
}

func (repository *fakeRepository) RevokeBandInvite(ctx context.Context, command accounts.RevokeBandInviteCommand) (accounts.BandInvite, error) {
	return accounts.BandInvite{}, nil
}

func (repository *fakeRepository) AcceptBandInvite(ctx context.Context, command accounts.AcceptBandInviteCommand) (accounts.BandMember, error) {
	return accounts.BandMember{}, nil
}

func testHandler() Handler {
	return testHandlerWithRepository(&fakeRepository{})
}

func testHandlerWithRepository(repository *fakeRepository) Handler {
	return NewHandler(repository, slog.Default())
}

func authenticatedRequest(t *testing.T, method string, target string, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	ctx, err := session.WithAuthenticatedUser(request.Context(), session.AuthenticatedUser{
		Provider:       "supabase",
		ProviderUserID: "auth_user_1",
		Email:          "band@example.com",
	}, "token")
	if err != nil {
		t.Fatalf("create authenticated context: %v", err)
	}
	ctx, err = session.WithVerifiedUser(ctx, session.VerifiedUser{
		Provider:       "supabase",
		ProviderUserID: "auth_user_1",
		Email:          "band@example.com",
	})
	if err != nil {
		t.Fatalf("create verified context: %v", err)
	}

	return request.WithContext(ctx)
}
