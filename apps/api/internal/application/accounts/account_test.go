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

func TestCreateBandInviteRequiresOwner(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{}
	input := validCreateBandInviteInput()
	input.Account.Role = permissions.RoleViewer

	_, err := CreateBandInvite(context.Background(), &repository, fixedInviteTokenGenerator("token_1"), input)
	if err == nil {
		t.Fatal("expected owner permission error")
	}
}

func TestCreateBandInviteValidatesEmail(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{}
	input := validCreateBandInviteInput()
	input.Email = "viewer.example.com"

	_, err := CreateBandInvite(context.Background(), &repository, fixedInviteTokenGenerator("token_1"), input)
	if err == nil {
		t.Fatal("expected email validation error")
	}
}

func TestCreateBandInviteStoresViewerCommand(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{
		invite: BandInvite{
			ID:        "invite_1",
			BandID:    "band_1",
			Email:     "viewer@example.com",
			Role:      permissions.RoleViewer,
			Status:    InviteStatusPending,
			ExpiresAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
			CreatedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	invite, err := CreateBandInvite(context.Background(), &repository, fixedInviteTokenGenerator("token_1"), validCreateBandInviteInput())
	if err != nil {
		t.Fatalf("create band invite: %v", err)
	}

	if repository.createInviteCommand.Role != permissions.RoleViewer {
		t.Fatalf("expected viewer role, got %q", repository.createInviteCommand.Role)
	}
	if repository.createInviteCommand.ExpiresAt.Format(time.RFC3339) != "2026-05-08T12:00:00Z" {
		t.Fatalf("expected seven-day expiration, got %s", repository.createInviteCommand.ExpiresAt.Format(time.RFC3339))
	}
	if invite.Token != "token_1" {
		t.Fatalf("expected response token, got %q", invite.Token)
	}
}

func TestRevokeBandInviteRequiresOwner(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{}
	input := RevokeBandInviteInput{
		Account:        viewerOwnerAccount(),
		InviteID:       "invite_1",
		IdempotencyKey: "idem_1",
		RequestID:      "request_1",
		RevokedAt:      time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}

	_, err := RevokeBandInvite(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected owner permission error")
	}
}

func TestAcceptBandInviteValidatesAuthenticatedEmail(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{}
	input := validAcceptBandInviteInput()
	input.Email = "viewer.example.com"

	_, err := AcceptBandInvite(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected email validation error")
	}
}

func TestAcceptBandInvitePropagatesInviteErrors(t *testing.T) {
	t.Parallel()

	repository := fakeBandAccountRepository{err: ErrInviteExpired}

	_, err := AcceptBandInvite(context.Background(), &repository, validAcceptBandInviteInput())
	if !errors.Is(err, ErrInviteExpired) {
		t.Fatalf("expected expired invite error, got %v", err)
	}
}

type fakeBandAccountRepository struct {
	account             OwnerAccount
	invite              BandInvite
	member              BandMember
	command             CreateOwnerAccountCommand
	createInviteCommand CreateBandInviteCommand
	err                 error
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

func (repository *fakeBandAccountRepository) ListBandMembers(ctx context.Context, query ListBandMembersQuery) ([]BandMember, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if repository.err != nil {
		return nil, repository.err
	}

	return []BandMember{repository.member}, nil
}

func (repository *fakeBandAccountRepository) ListBandInvites(ctx context.Context, query ListBandInvitesQuery) ([]BandInvite, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if repository.err != nil {
		return nil, repository.err
	}

	return []BandInvite{repository.invite}, nil
}

func (repository *fakeBandAccountRepository) CreateBandInvite(ctx context.Context, command CreateBandInviteCommand) (BandInvite, error) {
	if ctx == nil {
		return BandInvite{}, errors.New("context is required")
	}
	repository.createInviteCommand = command
	if repository.err != nil {
		return BandInvite{}, repository.err
	}

	return repository.invite, nil
}

func (repository *fakeBandAccountRepository) RevokeBandInvite(ctx context.Context, command RevokeBandInviteCommand) (BandInvite, error) {
	if ctx == nil {
		return BandInvite{}, errors.New("context is required")
	}
	if repository.err != nil {
		return BandInvite{}, repository.err
	}

	return repository.invite, nil
}

func (repository *fakeBandAccountRepository) AcceptBandInvite(ctx context.Context, command AcceptBandInviteCommand) (BandMember, error) {
	if ctx == nil {
		return BandMember{}, errors.New("context is required")
	}
	if repository.err != nil {
		return BandMember{}, repository.err
	}

	return repository.member, nil
}

func fixedInviteTokenGenerator(token string) InviteTokenGenerator {
	return func() (string, error) {
		return token, nil
	}
}

func validCreateBandInviteInput() CreateBandInviteInput {
	return CreateBandInviteInput{
		Account:        ownerAccount(),
		Email:          " VIEWER@example.com ",
		IdempotencyKey: " idem_1 ",
		RequestID:      " request_1 ",
		CreatedAt:      time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
}

func validAcceptBandInviteInput() AcceptBandInviteInput {
	return AcceptBandInviteInput{
		AuthProvider:       "supabase",
		AuthProviderUserID: "auth_viewer_1",
		Email:              "viewer@example.com",
		Token:              "token_1",
		IdempotencyKey:     "idem_1",
		RequestID:          "request_1",
		AcceptedAt:         time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
}

func ownerAccount() OwnerAccount {
	return OwnerAccount{
		UserID:       "user_1",
		BandID:       "band_1",
		Email:        "owner@example.com",
		BandName:     "Os Testes",
		BandTimezone: "America/Recife",
		Role:         permissions.RoleOwner,
	}
}

func viewerOwnerAccount() OwnerAccount {
	account := ownerAccount()
	account.Role = permissions.RoleViewer
	return account
}
