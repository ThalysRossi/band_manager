package accounts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

type BandAccountRepository interface {
	CreateOwnerAccount(ctx context.Context, command CreateOwnerAccountCommand) (OwnerAccount, error)
	GetCurrentAccount(ctx context.Context, query CurrentAccountQuery) (OwnerAccount, error)
}

type CreateOwnerAccountCommand struct {
	AuthProvider       string
	AuthProviderUserID string
	Email              string
	BandName           string
	BandTimezone       string
	IdempotencyKey     string
	RequestID          string
	CreatedAt          time.Time
}

type OwnerAccount struct {
	UserID       string
	BandID       string
	Email        string
	BandName     string
	BandTimezone string
	Role         permissions.Role
}

type CreateOwnerAccountInput struct {
	AuthProvider       string
	AuthProviderUserID string
	Email              string
	BandName           string
	BandTimezone       string
	IdempotencyKey     string
	RequestID          string
	CreatedAt          time.Time
}

type CurrentAccountQuery struct {
	AuthProvider       string
	AuthProviderUserID string
}

func CreateOwnerAccount(ctx context.Context, repository BandAccountRepository, input CreateOwnerAccountInput) (OwnerAccount, error) {
	command, err := validateCreateOwnerAccountInput(input)
	if err != nil {
		return OwnerAccount{}, err
	}

	account, err := repository.CreateOwnerAccount(ctx, command)
	if err != nil {
		return OwnerAccount{}, fmt.Errorf("create owner account for email %q and band %q: %w", command.Email, command.BandName, err)
	}

	if account.Role != permissions.RoleOwner {
		return OwnerAccount{}, fmt.Errorf("created owner account returned non-owner role %q for email %q and band %q", account.Role, command.Email, command.BandName)
	}

	return account, nil
}

func GetCurrentAccount(ctx context.Context, repository BandAccountRepository, query CurrentAccountQuery) (OwnerAccount, error) {
	validQuery, err := validateCurrentAccountQuery(query)
	if err != nil {
		return OwnerAccount{}, err
	}

	account, err := repository.GetCurrentAccount(ctx, validQuery)
	if err != nil {
		return OwnerAccount{}, fmt.Errorf("get current account for provider %q subject %q: %w", validQuery.AuthProvider, validQuery.AuthProviderUserID, err)
	}

	return account, nil
}

func validateCreateOwnerAccountInput(input CreateOwnerAccountInput) (CreateOwnerAccountCommand, error) {
	authProvider := strings.TrimSpace(input.AuthProvider)
	if authProvider == "" {
		return CreateOwnerAccountCommand{}, fmt.Errorf("auth provider is required")
	}

	authProviderUserID := strings.TrimSpace(input.AuthProviderUserID)
	if authProviderUserID == "" {
		return CreateOwnerAccountCommand{}, fmt.Errorf("auth provider user id is required")
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return CreateOwnerAccountCommand{}, fmt.Errorf("email is required")
	}

	if !strings.Contains(email, "@") {
		return CreateOwnerAccountCommand{}, fmt.Errorf("email %q must contain @", email)
	}

	bandName := strings.TrimSpace(input.BandName)
	if bandName == "" {
		return CreateOwnerAccountCommand{}, fmt.Errorf("band name is required")
	}

	bandTimezone := strings.TrimSpace(input.BandTimezone)
	if bandTimezone == "" {
		return CreateOwnerAccountCommand{}, fmt.Errorf("band timezone is required")
	}

	if input.CreatedAt.IsZero() {
		return CreateOwnerAccountCommand{}, fmt.Errorf("created at timestamp is required")
	}

	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey == "" {
		return CreateOwnerAccountCommand{}, fmt.Errorf("idempotency key is required")
	}

	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		return CreateOwnerAccountCommand{}, fmt.Errorf("request id is required")
	}

	return CreateOwnerAccountCommand{
		AuthProvider:       authProvider,
		AuthProviderUserID: authProviderUserID,
		Email:              email,
		BandName:           bandName,
		BandTimezone:       bandTimezone,
		IdempotencyKey:     idempotencyKey,
		RequestID:          requestID,
		CreatedAt:          input.CreatedAt.UTC(),
	}, nil
}

func validateCurrentAccountQuery(query CurrentAccountQuery) (CurrentAccountQuery, error) {
	authProvider := strings.TrimSpace(query.AuthProvider)
	if authProvider == "" {
		return CurrentAccountQuery{}, fmt.Errorf("auth provider is required")
	}

	authProviderUserID := strings.TrimSpace(query.AuthProviderUserID)
	if authProviderUserID == "" {
		return CurrentAccountQuery{}, fmt.Errorf("auth provider user id is required")
	}

	return CurrentAccountQuery{
		AuthProvider:       authProvider,
		AuthProviderUserID: authProviderUserID,
	}, nil
}
