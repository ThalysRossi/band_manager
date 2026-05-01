package account

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

const signupOperation = "auth_signup"

type Repository struct {
	pool *pgxpool.Pool
}

type currentAccountRow struct {
	UserID       string
	BandID       string
	Email        string
	BandName     string
	BandTimezone string
	Role         string
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

func (repository Repository) CreateOwnerAccount(ctx context.Context, command accounts.CreateOwnerAccountCommand) (accounts.OwnerAccount, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("begin owner account transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	requestHash, err := hashSignupRequest(command)
	if err != nil {
		return accounts.OwnerAccount{}, err
	}

	existingAccount, found, err := findIdempotentSignup(ctx, tx, command.AuthProviderUserID, command.IdempotencyKey, requestHash)
	if err != nil {
		return accounts.OwnerAccount{}, err
	}
	if found {
		return existingAccount, nil
	}

	userID := uuid.NewString()
	bandID := uuid.NewString()
	membershipID := uuid.NewString()
	auditLogID := uuid.NewString()
	idempotencyRecordID := uuid.NewString()

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, auth_provider, auth_provider_user_id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		`, userID, command.AuthProvider, command.AuthProviderUserID, command.Email, command.CreatedAt)
	if err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("insert owner user provider=%q provider_user_id=%q email=%q: %w", command.AuthProvider, command.AuthProviderUserID, command.Email, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO bands (id, name, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
	`, bandID, command.BandName, command.BandTimezone, command.CreatedAt)
	if err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("insert owner band band_id=%q band_name=%q: %w", bandID, command.BandName, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO band_memberships (id, band_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, membershipID, bandID, userID, permissions.RoleOwner, command.CreatedAt)
	if err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("insert owner membership band_id=%q user_id=%q: %w", bandID, userID, err)
	}

	account := accounts.OwnerAccount{
		UserID:       userID,
		BandID:       bandID,
		Email:        command.Email,
		BandName:     command.BandName,
		BandTimezone: command.BandTimezone,
		Role:         permissions.RoleOwner,
	}

	responseBody, err := json.Marshal(account)
	if err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("marshal idempotent signup response user_id=%q band_id=%q: %w", userID, bandID, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_records (id, scope_id, band_id, operation, idempotency_key, request_hash, response_body, status_code, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, idempotencyRecordID, command.AuthProviderUserID, bandID, signupOperation, command.IdempotencyKey, requestHash, responseBody, 201, command.CreatedAt.Add(15*time.Minute), command.CreatedAt)
	if err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("insert signup idempotency record scope_id=%q key=%q: %w", command.AuthProviderUserID, command.IdempotencyKey, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, band_id, action, entity_type, entity_id, request_id, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, auditLogID, userID, bandID, "auth.signup_owner", "band", bandID, command.RequestID, command.IdempotencyKey, command.CreatedAt)
	if err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("insert owner signup audit log user_id=%q band_id=%q: %w", userID, bandID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("commit owner account transaction user_id=%q band_id=%q: %w", userID, bandID, err)
	}

	return account, nil
}

func (repository Repository) GetCurrentAccount(ctx context.Context, query accounts.CurrentAccountQuery) (accounts.OwnerAccount, error) {
	row := repository.pool.QueryRow(ctx, `
		SELECT users.id, bands.id, users.email, bands.name, bands.timezone, band_memberships.role
		FROM users
		INNER JOIN band_memberships ON band_memberships.user_id = users.id
		INNER JOIN bands ON bands.id = band_memberships.band_id
		WHERE users.auth_provider = $1 AND users.auth_provider_user_id = $2
		ORDER BY band_memberships.created_at ASC
		LIMIT 1
	`, query.AuthProvider, query.AuthProviderUserID)

	account, err := scanCurrentAccount(row)
	if err != nil {
		return accounts.OwnerAccount{}, fmt.Errorf("query current account provider=%q provider_user_id=%q: %w", query.AuthProvider, query.AuthProviderUserID, err)
	}

	return account, nil
}

func findIdempotentSignup(ctx context.Context, tx pgx.Tx, scopeID string, idempotencyKey string, requestHash string) (accounts.OwnerAccount, bool, error) {
	var storedRequestHash string
	var responseBody []byte
	err := tx.QueryRow(ctx, `
		SELECT request_hash, response_body
		FROM idempotency_records
		WHERE scope_id = $1 AND operation = $2 AND idempotency_key = $3 AND expires_at > NOW()
	`, scopeID, signupOperation, idempotencyKey).Scan(&storedRequestHash, &responseBody)
	if errors.Is(err, pgx.ErrNoRows) {
		return accounts.OwnerAccount{}, false, nil
	}
	if err != nil {
		return accounts.OwnerAccount{}, false, fmt.Errorf("query signup idempotency record scope_id=%q key=%q: %w", scopeID, idempotencyKey, err)
	}

	if storedRequestHash != requestHash {
		return accounts.OwnerAccount{}, false, fmt.Errorf("idempotency key %q was already used with a different signup request", idempotencyKey)
	}

	var account accounts.OwnerAccount
	if err := json.Unmarshal(responseBody, &account); err != nil {
		return accounts.OwnerAccount{}, false, fmt.Errorf("parse idempotent signup response scope_id=%q key=%q: %w", scopeID, idempotencyKey, err)
	}

	return account, true, nil
}

func hashSignupRequest(command accounts.CreateOwnerAccountCommand) (string, error) {
	body, err := json.Marshal(struct {
		AuthProvider       string `json:"authProvider"`
		AuthProviderUserID string `json:"authProviderUserId"`
		Email              string `json:"email"`
		BandName           string `json:"bandName"`
		BandTimezone       string `json:"bandTimezone"`
	}{
		AuthProvider:       command.AuthProvider,
		AuthProviderUserID: command.AuthProviderUserID,
		Email:              command.Email,
		BandName:           command.BandName,
		BandTimezone:       command.BandTimezone,
	})
	if err != nil {
		return "", fmt.Errorf("marshal signup request hash body: %w", err)
	}

	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func scanCurrentAccount(row pgx.Row) (accounts.OwnerAccount, error) {
	var accountRow currentAccountRow
	if err := row.Scan(
		&accountRow.UserID,
		&accountRow.BandID,
		&accountRow.Email,
		&accountRow.BandName,
		&accountRow.BandTimezone,
		&accountRow.Role,
	); err != nil {
		return accounts.OwnerAccount{}, err
	}

	role, err := permissions.ParseRole(accountRow.Role)
	if err != nil {
		return accounts.OwnerAccount{}, err
	}

	return accounts.OwnerAccount{
		UserID:       accountRow.UserID,
		BandID:       accountRow.BandID,
		Email:        accountRow.Email,
		BandName:     accountRow.BandName,
		BandTimezone: accountRow.BandTimezone,
		Role:         role,
	}, nil
}
