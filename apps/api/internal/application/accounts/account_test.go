package accounts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestCreateOwnerAccountValidatesInput(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{}
	input := CreateOwnerAccountInput{
		AuthProvider:       "supabase",
		AuthProviderUserID: "auth_user_1",
		Email:              " ",
		BandName:           "Os Testes",
		BandTimezone:       "America/Recife",
		IdempotencyKey:     "idem_1",
		RequestID:          "request_1",
		CreatedAt:          time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC),
	}

	_, err := CreateOwnerAccount(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCreateOwnerAccountStoresTrimmedOwnerCommand(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 4, 28, 12, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	repository := fakeBandAccountRepository{
		account: OwnerAccount{
			UserID:       "user_1",
			BandID:       "band_1",
			Email:        "band@example.com",
			BandName:     "Os Testes",
			BandTimezone: "America/Recife",
			Role:         permissions.RoleOwner,
		},
	}
	input := CreateOwnerAccountInput{
		AuthProvider:       " supabase ",
		AuthProviderUserID: " auth_user_1 ",
		Email:              " band@example.com ",
		BandName:           " Os Testes ",
		BandTimezone:       " America/Recife ",
		IdempotencyKey:     " idem_1 ",
		RequestID:          " request_1 ",
		CreatedAt:          createdAt,
	}

	account, err := CreateOwnerAccount(context.Background(), &repository, input)
	if err != nil {
		t.Fatalf("create owner account: %v", err)
	}

	if account.Role != permissions.RoleOwner {
		t.Fatalf("expected owner role, got %q", account.Role)
	}

	if repository.command.Email != "band@example.com" {
		t.Fatalf("expected trimmed email, got %q", repository.command.Email)
	}

	if repository.command.AuthProviderUserID != "auth_user_1" {
		t.Fatalf("expected trimmed auth provider user id, got %q", repository.command.AuthProviderUserID)
	}

	if repository.command.BandName != "Os Testes" {
		t.Fatalf("expected trimmed band name, got %q", repository.command.BandName)
	}

	if repository.command.CreatedAt.Location() != time.UTC {
		t.Fatalf("expected UTC created at, got %s", repository.command.CreatedAt.Location())
	}
}

func TestCreateOwnerAccountIncludesContextInRepositoryError(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{err: errors.New("database unavailable")}
	input := CreateOwnerAccountInput{
		AuthProvider:       "supabase",
		AuthProviderUserID: "auth_user_1",
		Email:              "band@example.com",
		BandName:           "Os Testes",
		BandTimezone:       "America/Recife",
		IdempotencyKey:     "idem_1",
		RequestID:          "request_1",
		CreatedAt:          time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC),
	}

	_, err := CreateOwnerAccount(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected repository error")
	}
}

func TestGetCurrentAccountValidatesQuery(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{}

	_, err := GetCurrentAccount(context.Background(), &repository, CurrentAccountQuery{
		AuthProvider:       "supabase",
		AuthProviderUserID: " ",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

type fakeBandAccountRepository struct {
	account OwnerAccount
	command CreateOwnerAccountCommand
	err     error
}

func (repository *fakeBandAccountRepository) CreateOwnerAccount(ctx context.Context, command CreateOwnerAccountCommand) (OwnerAccount, error) {
	if ctx == nil {
		return OwnerAccount{}, errors.New("context is required")
	}

	repository.command = command
	if repository.err != nil {
		return OwnerAccount{}, repository.err
	}

	return repository.account, nil
}

func (repository *fakeBandAccountRepository) GetCurrentAccount(ctx context.Context, query CurrentAccountQuery) (OwnerAccount, error) {
	if ctx == nil {
		return OwnerAccount{}, errors.New("context is required")
	}

	if repository.err != nil {
		return OwnerAccount{}, repository.err
	}

	return repository.account, nil
}
