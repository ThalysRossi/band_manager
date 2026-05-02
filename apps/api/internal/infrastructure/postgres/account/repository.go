package account

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

const signupOperation = "auth_signup"
const uniqueViolationCode = "23505"

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

type memberRow struct {
	UserID   string
	Email    string
	BandID   string
	Role     string
	JoinedAt time.Time
}

type inviteRow struct {
	ID        string
	BandID    string
	Email     string
	Role      string
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
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

func (repository Repository) ListBandMembers(ctx context.Context, query accounts.ListBandMembersQuery) ([]accounts.BandMember, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT users.id, users.email, band_memberships.band_id, band_memberships.role, band_memberships.created_at
		FROM band_memberships
		INNER JOIN users ON users.id = band_memberships.user_id
		WHERE band_memberships.band_id = $1
		ORDER BY band_memberships.created_at ASC
	`, query.Account.BandID)
	if err != nil {
		return nil, fmt.Errorf("query band members band_id=%q: %w", query.Account.BandID, err)
	}
	defer rows.Close()

	members := make([]accounts.BandMember, 0)
	for rows.Next() {
		member, scanErr := scanBandMember(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan band member band_id=%q: %w", query.Account.BandID, scanErr)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate band members band_id=%q: %w", query.Account.BandID, err)
	}

	return members, nil
}

func (repository Repository) ListBandInvites(ctx context.Context, query accounts.ListBandInvitesQuery) ([]accounts.BandInvite, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id, band_id, email, role,
			CASE
				WHEN status = 'pending' AND expires_at <= NOW() THEN 'expired'
				ELSE status
			END AS status,
			expires_at, created_at, updated_at
		FROM band_invites
		WHERE band_id = $1
		ORDER BY created_at DESC
	`, query.Account.BandID)
	if err != nil {
		return nil, fmt.Errorf("query band invites band_id=%q: %w", query.Account.BandID, err)
	}
	defer rows.Close()

	invites := make([]accounts.BandInvite, 0)
	for rows.Next() {
		invite, scanErr := scanBandInvite(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan band invite band_id=%q: %w", query.Account.BandID, scanErr)
		}
		invites = append(invites, invite)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate band invites band_id=%q: %w", query.Account.BandID, err)
	}

	return invites, nil
}

func (repository Repository) CreateBandInvite(ctx context.Context, command accounts.CreateBandInviteCommand) (accounts.BandInvite, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return accounts.BandInvite{}, fmt.Errorf("begin create invite transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	inviteID := uuid.NewString()
	tokenHash := hashInviteToken(command.Token)
	_, err = tx.Exec(ctx, `
		INSERT INTO band_invites (id, band_id, email, role, status, token_hash, invited_by_user_id, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, inviteID, command.Account.BandID, command.Email, command.Role, command.Status, tokenHash, command.Account.UserID, command.ExpiresAt, command.CreatedAt)
	if err != nil {
		if isUniqueViolation(err, "band_invites_pending_email_idx") {
			return accounts.BandInvite{}, accounts.ErrDuplicatePendingInvite
		}
		if isUniqueViolation(err, "band_invites_token_hash_idx") {
			return accounts.BandInvite{}, fmt.Errorf("invite token hash collision band_id=%q email=%q: %w", command.Account.BandID, command.Email, err)
		}
		return accounts.BandInvite{}, fmt.Errorf("insert band invite band_id=%q email=%q: %w", command.Account.BandID, command.Email, err)
	}

	if err := insertAccountAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "account.invite_created", "band_invite", inviteID, command.RequestID, command.IdempotencyKey, command.CreatedAt); err != nil {
		return accounts.BandInvite{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return accounts.BandInvite{}, fmt.Errorf("commit create invite transaction band_id=%q invite_id=%q: %w", command.Account.BandID, inviteID, err)
	}

	return accounts.BandInvite{
		ID:        inviteID,
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

func (repository Repository) RevokeBandInvite(ctx context.Context, command accounts.RevokeBandInviteCommand) (accounts.BandInvite, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return accounts.BandInvite{}, fmt.Errorf("begin revoke invite transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	invite, err := getInviteForBandUpdate(ctx, tx, command.Account.BandID, command.InviteID)
	if err != nil {
		return accounts.BandInvite{}, err
	}

	if invite.Status == accounts.InviteStatusAccepted {
		return accounts.BandInvite{}, accounts.ErrInviteAccepted
	}
	if invite.Status == accounts.InviteStatusRevoked {
		return accounts.BandInvite{}, accounts.ErrInviteRevoked
	}
	if invite.Status == accounts.InviteStatusExpired || !invite.ExpiresAt.After(command.RevokedAt) {
		return accounts.BandInvite{}, accounts.ErrInviteExpired
	}

	_, err = tx.Exec(ctx, `
		UPDATE band_invites
		SET status = 'revoked', revoked_at = $1, updated_at = $1
		WHERE id = $2 AND band_id = $3
	`, command.RevokedAt, command.InviteID, command.Account.BandID)
	if err != nil {
		return accounts.BandInvite{}, fmt.Errorf("update revoked invite band_id=%q invite_id=%q: %w", command.Account.BandID, command.InviteID, err)
	}

	if err := insertAccountAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "account.invite_revoked", "band_invite", command.InviteID, command.RequestID, command.IdempotencyKey, command.RevokedAt); err != nil {
		return accounts.BandInvite{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return accounts.BandInvite{}, fmt.Errorf("commit revoke invite transaction band_id=%q invite_id=%q: %w", command.Account.BandID, command.InviteID, err)
	}

	invite.Status = accounts.InviteStatusRevoked
	invite.UpdatedAt = command.RevokedAt
	invite.Token = ""
	return invite, nil
}

func (repository Repository) AcceptBandInvite(ctx context.Context, command accounts.AcceptBandInviteCommand) (accounts.BandMember, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return accounts.BandMember{}, fmt.Errorf("begin accept invite transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	invite, acceptedByUserID, err := getInviteByTokenHashForUpdate(ctx, tx, hashInviteToken(command.Token))
	if err != nil {
		return accounts.BandMember{}, err
	}

	if !strings.EqualFold(invite.Email, command.Email) {
		return accounts.BandMember{}, accounts.ErrInviteEmailMismatch
	}

	if invite.Status == accounts.InviteStatusAccepted {
		if !acceptedByUserID.Valid {
			return accounts.BandMember{}, accounts.ErrMembershipConflict
		}
		member, memberErr := getBandMemberByUserID(ctx, tx, invite.BandID, acceptedByUserID.String)
		if memberErr != nil {
			return accounts.BandMember{}, memberErr
		}
		if err := tx.Commit(ctx); err != nil {
			return accounts.BandMember{}, fmt.Errorf("commit duplicate accept invite transaction band_id=%q invite_id=%q: %w", invite.BandID, invite.ID, err)
		}
		return member, nil
	}
	if invite.Status == accounts.InviteStatusRevoked {
		return accounts.BandMember{}, accounts.ErrInviteRevoked
	}
	if invite.Status == accounts.InviteStatusExpired || !invite.ExpiresAt.After(command.AcceptedAt) {
		if err := markInviteExpired(ctx, tx, invite.ID, command.AcceptedAt); err != nil {
			return accounts.BandMember{}, err
		}
		return accounts.BandMember{}, accounts.ErrInviteExpired
	}

	userID, err := findOrCreateAcceptedUser(ctx, tx, command)
	if err != nil {
		return accounts.BandMember{}, err
	}

	_, err = getBandMemberByUserID(ctx, tx, invite.BandID, userID)
	if err == nil {
		return accounts.BandMember{}, accounts.ErrMembershipConflict
	}
	if !errors.Is(err, accounts.ErrMembershipConflict) {
		return accounts.BandMember{}, err
	}

	membershipID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO band_memberships (id, band_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, membershipID, invite.BandID, userID, permissions.RoleViewer, command.AcceptedAt)
	if err != nil {
		if isUniqueViolation(err, "band_memberships_band_id_user_id_key") {
			return accounts.BandMember{}, accounts.ErrMembershipConflict
		}
		return accounts.BandMember{}, fmt.Errorf("insert accepted invite membership band_id=%q user_id=%q: %w", invite.BandID, userID, err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE band_invites
		SET status = 'accepted', accepted_by_user_id = $1, accepted_at = $2, updated_at = $2
		WHERE id = $3
	`, userID, command.AcceptedAt, invite.ID)
	if err != nil {
		return accounts.BandMember{}, fmt.Errorf("update accepted invite band_id=%q invite_id=%q user_id=%q: %w", invite.BandID, invite.ID, userID, err)
	}

	if err := insertAccountAuditLog(ctx, tx, userID, invite.BandID, "account.invite_accepted", "band_invite", invite.ID, command.RequestID, command.IdempotencyKey, command.AcceptedAt); err != nil {
		return accounts.BandMember{}, err
	}

	member, err := getBandMemberByUserID(ctx, tx, invite.BandID, userID)
	if err != nil {
		return accounts.BandMember{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return accounts.BandMember{}, fmt.Errorf("commit accept invite transaction band_id=%q invite_id=%q: %w", invite.BandID, invite.ID, err)
	}

	return member, nil
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

func hashInviteToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
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

func scanBandMember(row pgx.Row) (accounts.BandMember, error) {
	var member memberRow
	if err := row.Scan(&member.UserID, &member.Email, &member.BandID, &member.Role, &member.JoinedAt); err != nil {
		return accounts.BandMember{}, err
	}

	role, err := permissions.ParseRole(member.Role)
	if err != nil {
		return accounts.BandMember{}, err
	}

	return accounts.BandMember{
		UserID:   member.UserID,
		Email:    member.Email,
		BandID:   member.BandID,
		Role:     role,
		JoinedAt: member.JoinedAt,
	}, nil
}

func scanBandInvite(row pgx.Row) (accounts.BandInvite, error) {
	var invite inviteRow
	if err := row.Scan(&invite.ID, &invite.BandID, &invite.Email, &invite.Role, &invite.Status, &invite.ExpiresAt, &invite.CreatedAt, &invite.UpdatedAt); err != nil {
		return accounts.BandInvite{}, err
	}

	role, err := permissions.ParseRole(invite.Role)
	if err != nil {
		return accounts.BandInvite{}, err
	}

	status, err := parseInviteStatus(invite.Status)
	if err != nil {
		return accounts.BandInvite{}, err
	}

	return accounts.BandInvite{
		ID:        invite.ID,
		BandID:    invite.BandID,
		Email:     invite.Email,
		Role:      role,
		Status:    status,
		ExpiresAt: invite.ExpiresAt,
		CreatedAt: invite.CreatedAt,
		UpdatedAt: invite.UpdatedAt,
	}, nil
}

func parseInviteStatus(value string) (accounts.InviteStatus, error) {
	status := accounts.InviteStatus(value)
	switch status {
	case accounts.InviteStatusPending, accounts.InviteStatusAccepted, accounts.InviteStatusRevoked, accounts.InviteStatusExpired:
		return status, nil
	default:
		return "", fmt.Errorf("invalid invite status %q", value)
	}
}

func getInviteForBandUpdate(ctx context.Context, tx pgx.Tx, bandID string, inviteID string) (accounts.BandInvite, error) {
	invite, err := scanBandInvite(tx.QueryRow(ctx, `
		SELECT id, band_id, email, role, status, expires_at, created_at, updated_at
		FROM band_invites
		WHERE id = $1 AND band_id = $2
		FOR UPDATE
	`, inviteID, bandID))
	if errors.Is(err, pgx.ErrNoRows) {
		return accounts.BandInvite{}, accounts.ErrInviteNotFound
	}
	if err != nil {
		return accounts.BandInvite{}, fmt.Errorf("query invite for update band_id=%q invite_id=%q: %w", bandID, inviteID, err)
	}

	return invite, nil
}

func getInviteByTokenHashForUpdate(ctx context.Context, tx pgx.Tx, tokenHash string) (accounts.BandInvite, sql.NullString, error) {
	var acceptedByUserID sql.NullString
	var invite inviteRow
	err := tx.QueryRow(ctx, `
		SELECT id, band_id, email, role, status, expires_at, created_at, updated_at, accepted_by_user_id::text
		FROM band_invites
		WHERE token_hash = $1
		FOR UPDATE
	`, tokenHash).Scan(&invite.ID, &invite.BandID, &invite.Email, &invite.Role, &invite.Status, &invite.ExpiresAt, &invite.CreatedAt, &invite.UpdatedAt, &acceptedByUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return accounts.BandInvite{}, sql.NullString{}, accounts.ErrInviteNotFound
	}
	if err != nil {
		return accounts.BandInvite{}, sql.NullString{}, fmt.Errorf("query invite by token hash: %w", err)
	}

	role, err := permissions.ParseRole(invite.Role)
	if err != nil {
		return accounts.BandInvite{}, sql.NullString{}, err
	}

	status, err := parseInviteStatus(invite.Status)
	if err != nil {
		return accounts.BandInvite{}, sql.NullString{}, err
	}

	return accounts.BandInvite{
		ID:        invite.ID,
		BandID:    invite.BandID,
		Email:     invite.Email,
		Role:      role,
		Status:    status,
		ExpiresAt: invite.ExpiresAt,
		CreatedAt: invite.CreatedAt,
		UpdatedAt: invite.UpdatedAt,
	}, acceptedByUserID, nil
}

func findOrCreateAcceptedUser(ctx context.Context, tx pgx.Tx, command accounts.AcceptBandInviteCommand) (string, error) {
	var userID string
	var email string
	err := tx.QueryRow(ctx, `
		SELECT id, email
		FROM users
		WHERE auth_provider = $1 AND auth_provider_user_id = $2
	`, command.AuthProvider, command.AuthProviderUserID).Scan(&userID, &email)
	if err == nil {
		if !strings.EqualFold(email, command.Email) {
			return "", accounts.ErrInviteEmailMismatch
		}
		return userID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("query accepted invite user provider=%q subject=%q: %w", command.AuthProvider, command.AuthProviderUserID, err)
	}

	err = tx.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE email = $1
	`, command.Email).Scan(&userID)
	if err == nil {
		return "", accounts.ErrMembershipConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("query accepted invite user by email=%q: %w", command.Email, err)
	}

	userID = uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, auth_provider, auth_provider_user_id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, userID, command.AuthProvider, command.AuthProviderUserID, command.Email, command.AcceptedAt)
	if err != nil {
		return "", fmt.Errorf("insert accepted invite user provider=%q subject=%q email=%q: %w", command.AuthProvider, command.AuthProviderUserID, command.Email, err)
	}

	return userID, nil
}

func getBandMemberByUserID(ctx context.Context, tx pgx.Tx, bandID string, userID string) (accounts.BandMember, error) {
	member, err := scanBandMember(tx.QueryRow(ctx, `
		SELECT users.id, users.email, band_memberships.band_id, band_memberships.role, band_memberships.created_at
		FROM band_memberships
		INNER JOIN users ON users.id = band_memberships.user_id
		WHERE band_memberships.band_id = $1 AND users.id = $2
	`, bandID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return accounts.BandMember{}, accounts.ErrMembershipConflict
	}
	if err != nil {
		return accounts.BandMember{}, fmt.Errorf("query band member band_id=%q user_id=%q: %w", bandID, userID, err)
	}

	return member, nil
}

func markInviteExpired(ctx context.Context, tx pgx.Tx, inviteID string, updatedAt time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE band_invites
		SET status = 'expired', updated_at = $1
		WHERE id = $2 AND status = 'pending'
	`, updatedAt, inviteID)
	if err != nil {
		return fmt.Errorf("mark invite expired invite_id=%q: %w", inviteID, err)
	}

	return nil
}

func insertAccountAuditLog(ctx context.Context, tx pgx.Tx, userID string, bandID string, action string, entityType string, entityID string, requestID string, idempotencyKey string, createdAt time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, band_id, action, entity_type, entity_id, request_id, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, uuid.NewString(), userID, bandID, action, entityType, entityID, requestID, idempotencyKey, createdAt)
	if err != nil {
		return fmt.Errorf("insert account audit log user_id=%q band_id=%q action=%q entity_type=%q entity_id=%q: %w", userID, bandID, action, entityType, entityID, err)
	}

	return nil
}

func isUniqueViolation(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == constraintName
}
