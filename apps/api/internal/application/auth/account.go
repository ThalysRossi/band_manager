package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

type BandAccountRepository interface {
	CreateOwnerAccount(ctx context.Context, command CreateOwnerAccountCommand) (OwnerAccount, error)
}

type CreateOwnerAccountCommand struct {
	Email        string
	BandName     string
	BandTimezone string
	CreatedAt    time.Time
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
	Email        string
	BandName     string
	BandTimezone string
	CreatedAt    time.Time
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

func validateCreateOwnerAccountInput(input CreateOwnerAccountInput) (CreateOwnerAccountCommand, error) {
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

	return CreateOwnerAccountCommand{
		Email:        email,
		BandName:     bandName,
		BandTimezone: bandTimezone,
		CreatedAt:    input.CreatedAt.UTC(),
	}, nil
}
