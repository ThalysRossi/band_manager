package account

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestRepositoryCreateListAndRevokeInvite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	invite, err := repository.CreateBandInvite(ctx, validCreateInviteCommand(account, "viewer@example.com", "token_create"))
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if invite.Token != "token_create" {
		t.Fatalf("expected response token, got %q", invite.Token)
	}

	invites, err := repository.ListBandInvites(ctx, accounts.ListBandInvitesQuery{Account: account})
	if err != nil {
		t.Fatalf("list invites: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("expected one invite, got %d", len(invites))
	}
	if invites[0].Token != "" {
		t.Fatalf("expected listed invite token to be omitted, got %q", invites[0].Token)
	}

	_, err = repository.CreateBandInvite(ctx, validCreateInviteCommand(account, "viewer@example.com", "token_duplicate"))
	if !errors.Is(err, accounts.ErrDuplicatePendingInvite) {
		t.Fatalf("expected duplicate pending invite error, got %v", err)
	}

	revokedInvite, err := repository.RevokeBandInvite(ctx, accounts.RevokeBandInviteCommand{
		Account:        account,
		InviteID:       invite.ID,
		IdempotencyKey: "idem_invite_revoke",
		RequestID:      "request_invite_revoke",
		RevokedAt:      testTimestamp().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("revoke invite: %v", err)
	}
	if revokedInvite.Status != accounts.InviteStatusRevoked {
		t.Fatalf("expected revoked status, got %q", revokedInvite.Status)
	}

	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND entity_id = $2", []interface{}{account.BandID, invite.ID}, 2)
}

func TestRepositoryAcceptInviteCreatesMembershipOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)
	command := validCreateInviteCommand(account, "viewer@example.com", "token_accept")

	invite, err := repository.CreateBandInvite(ctx, command)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	acceptCommand := accounts.AcceptBandInviteCommand{
		AuthProvider:       "supabase",
		AuthProviderUserID: "auth_viewer_1",
		Email:              "viewer@example.com",
		Token:              "token_accept",
		IdempotencyKey:     "idem_invite_accept",
		RequestID:          "request_invite_accept",
		AcceptedAt:         testTimestamp().Add(time.Minute),
	}
	member, err := repository.AcceptBandInvite(ctx, acceptCommand)
	if err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	if member.Role != permissions.RoleViewer {
		t.Fatalf("expected viewer membership, got %q", member.Role)
	}

	acceptedAgain, err := repository.AcceptBandInvite(ctx, acceptCommand)
	if err != nil {
		t.Fatalf("accept invite twice: %v", err)
	}
	if acceptedAgain.UserID != member.UserID {
		t.Fatalf("expected existing member %q, got %q", member.UserID, acceptedAgain.UserID)
	}

	assertTableCount(t, pool, "band_memberships", "band_id = $1 AND user_id = $2", []interface{}{account.BandID, member.UserID}, 1)
	assertTableCount(t, pool, "band_invites", "id = $1 AND status = $2 AND accepted_by_user_id = $3", []interface{}{invite.ID, "accepted", member.UserID}, 1)
}

func newIntegrationDatabase(t *testing.T) (*pgxpool.Pool, accounts.OwnerAccount) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		t.Skip("DATABASE_URL is not set; skipping Postgres integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	setupPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skipf("Postgres is unavailable: %v", err)
	}
	if err := setupPool.Ping(ctx); err != nil {
		setupPool.Close()
		t.Skipf("Postgres is unavailable: %v", err)
	}

	schemaName := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "_")
	quotedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	_, err = setupPool.Exec(ctx, "CREATE SCHEMA "+quotedSchemaName)
	if err != nil {
		setupPool.Close()
		t.Fatalf("create test schema %q: %v", schemaName, err)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		setupPool.Close()
		t.Fatalf("parse database url: %v", err)
	}
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = map[string]string{}
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schemaName
	poolConfig.MaxConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		setupPool.Close()
		t.Fatalf("create schema-scoped pool: %v", err)
	}

	applyMigrations(ctx, t, pool)
	account := seedAccount(ctx, t, pool)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		pool.Close()
		_, dropErr := setupPool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+quotedSchemaName+" CASCADE")
		if dropErr != nil {
			t.Logf("drop test schema %q: %v", schemaName, dropErr)
		}
		setupPool.Close()
	})

	return pool, account
}

func applyMigrations(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, migrationPath := range migrationFilePaths(t) {
		body, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration %s: %v", migrationPath, err)
		}

		statements, err := upMigrationStatements(string(body))
		if err != nil {
			t.Fatalf("parse migration %s: %v", migrationPath, err)
		}

		for _, statement := range statements {
			_, err := pool.Exec(ctx, statement)
			if err != nil {
				t.Fatalf("apply migration %s statement %q: %v", migrationPath, statement, err)
			}
		}
	}
}

func migrationFilePaths(t *testing.T) []string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../.."))
	migrationPaths, err := filepath.Glob(filepath.Join(apiRoot, "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migration files: %v", err)
	}
	sort.Strings(migrationPaths)

	return migrationPaths
}

func upMigrationStatements(body string) ([]string, error) {
	upMarkerIndex := strings.Index(body, "-- +goose Up")
	if upMarkerIndex == -1 {
		return nil, errors.New("goose up marker is required")
	}

	downMarkerIndex := strings.Index(body, "-- +goose Down")
	if downMarkerIndex == -1 {
		return nil, errors.New("goose down marker is required")
	}

	if downMarkerIndex <= upMarkerIndex {
		return nil, errors.New("goose down marker must follow up marker")
	}

	upBody := body[upMarkerIndex+len("-- +goose Up") : downMarkerIndex]
	parts := strings.Split(upBody, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement != "" {
			statements = append(statements, statement)
		}
	}

	return statements, nil
}

func seedAccount(ctx context.Context, t *testing.T, pool *pgxpool.Pool) accounts.OwnerAccount {
	t.Helper()

	now := testTimestamp()
	userID := uuid.NewString()
	bandID := uuid.NewString()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, auth_provider, auth_provider_user_id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, userID, "supabase", "auth_"+userID, "owner_"+userID+"@example.com", now)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO bands (id, name, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
	`, bandID, "Os Testes", "America/Recife", now)
	if err != nil {
		t.Fatalf("seed band: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO band_memberships (id, band_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, uuid.NewString(), bandID, userID, permissions.RoleOwner, now)
	if err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	return accounts.OwnerAccount{
		UserID:       userID,
		BandID:       bandID,
		Email:        "owner_" + userID + "@example.com",
		BandName:     "Os Testes",
		BandTimezone: "America/Recife",
		Role:         permissions.RoleOwner,
	}
}

func validCreateInviteCommand(account accounts.OwnerAccount, email string, token string) accounts.CreateBandInviteCommand {
	return accounts.CreateBandInviteCommand{
		Account:        account,
		Email:          email,
		Role:           permissions.RoleViewer,
		Status:         accounts.InviteStatusPending,
		Token:          token,
		ExpiresAt:      testTimestamp().Add(7 * 24 * time.Hour),
		IdempotencyKey: "idem_invite_create_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		RequestID:      "request_invite_create",
		CreatedAt:      testTimestamp(),
	}
}

func assertTableCount(t *testing.T, pool *pgxpool.Pool, tableName string, whereClause string, args []interface{}, expectedCount int) {
	t.Helper()

	query := "SELECT COUNT(*) FROM " + tableName + " WHERE " + whereClause
	var count int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", tableName, err)
	}

	if count != expectedCount {
		t.Fatalf("expected %d rows in %s, got %d", expectedCount, tableName, count)
	}
}

func testTimestamp() time.Time {
	return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
}
