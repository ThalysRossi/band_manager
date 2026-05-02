package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

const inviteExpirationDuration = 7 * 24 * time.Hour
const inviteTokenByteLength = 32

var (
	ErrDuplicatePendingInvite = errors.New("duplicate pending invite")
	ErrInviteNotFound         = errors.New("invite not found")
	ErrInviteExpired          = errors.New("invite expired")
	ErrInviteEmailMismatch    = errors.New("invite email mismatch")
	ErrInviteRevoked          = errors.New("invite revoked")
	ErrInviteAccepted         = errors.New("invite already accepted")
	ErrMembershipConflict     = errors.New("membership conflict")
)

type BandAccountRepository interface {
	CreateOwnerAccount(ctx context.Context, command CreateOwnerAccountCommand) (OwnerAccount, error)
	GetCurrentAccount(ctx context.Context, query CurrentAccountQuery) (OwnerAccount, error)
	ListBandMembers(ctx context.Context, query ListBandMembersQuery) ([]BandMember, error)
	ListBandInvites(ctx context.Context, query ListBandInvitesQuery) ([]BandInvite, error)
	CreateBandInvite(ctx context.Context, command CreateBandInviteCommand) (BandInvite, error)
	RevokeBandInvite(ctx context.Context, command RevokeBandInviteCommand) (BandInvite, error)
	AcceptBandInvite(ctx context.Context, command AcceptBandInviteCommand) (BandMember, error)
}

type InviteTokenGenerator func() (string, error)

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

type BandMember struct {
	UserID   string
	Email    string
	BandID   string
	Role     permissions.Role
	JoinedAt time.Time
}

type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusRevoked  InviteStatus = "revoked"
	InviteStatusExpired  InviteStatus = "expired"
)

type BandInvite struct {
	ID        string
	BandID    string
	Email     string
	Role      permissions.Role
	Status    InviteStatus
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	Token     string
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

type ListBandMembersInput struct {
	Account OwnerAccount
}

type ListBandInvitesInput struct {
	Account OwnerAccount
}

type CreateBandInviteInput struct {
	Account        OwnerAccount
	Email          string
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type RevokeBandInviteInput struct {
	Account        OwnerAccount
	InviteID       string
	IdempotencyKey string
	RequestID      string
	RevokedAt      time.Time
}

type AcceptBandInviteInput struct {
	AuthProvider       string
	AuthProviderUserID string
	Email              string
	Token              string
	IdempotencyKey     string
	RequestID          string
	AcceptedAt         time.Time
}

type ListBandMembersQuery struct {
	Account OwnerAccount
}

type ListBandInvitesQuery struct {
	Account OwnerAccount
}

type CreateBandInviteCommand struct {
	Account        OwnerAccount
	Email          string
	Role           permissions.Role
	Status         InviteStatus
	Token          string
	ExpiresAt      time.Time
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type RevokeBandInviteCommand struct {
	Account        OwnerAccount
	InviteID       string
	IdempotencyKey string
	RequestID      string
	RevokedAt      time.Time
}

type AcceptBandInviteCommand struct {
	AuthProvider       string
	AuthProviderUserID string
	Email              string
	Token              string
	IdempotencyKey     string
	RequestID          string
	AcceptedAt         time.Time
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

func ListBandMembers(ctx context.Context, repository BandAccountRepository, input ListBandMembersInput) ([]BandMember, error) {
	query, err := validateListBandMembersInput(input)
	if err != nil {
		return nil, err
	}

	members, err := repository.ListBandMembers(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list band members band_id=%q: %w", query.Account.BandID, err)
	}

	return members, nil
}

func ListBandInvites(ctx context.Context, repository BandAccountRepository, input ListBandInvitesInput) ([]BandInvite, error) {
	query, err := validateListBandInvitesInput(input)
	if err != nil {
		return nil, err
	}

	invites, err := repository.ListBandInvites(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list band invites band_id=%q: %w", query.Account.BandID, err)
	}

	return invites, nil
}

func CreateBandInvite(ctx context.Context, repository BandAccountRepository, tokenGenerator InviteTokenGenerator, input CreateBandInviteInput) (BandInvite, error) {
	command, err := validateCreateBandInviteInput(input)
	if err != nil {
		return BandInvite{}, err
	}

	token, err := tokenGenerator()
	if err != nil {
		return BandInvite{}, fmt.Errorf("generate invite token band_id=%q email=%q: %w", command.Account.BandID, command.Email, err)
	}
	if strings.TrimSpace(token) == "" {
		return BandInvite{}, fmt.Errorf("invite token generator returned an empty token")
	}
	command.Token = strings.TrimSpace(token)

	invite, err := repository.CreateBandInvite(ctx, command)
	if err != nil {
		return BandInvite{}, fmt.Errorf("create band invite band_id=%q email=%q: %w", command.Account.BandID, command.Email, err)
	}

	if invite.Token == "" {
		invite.Token = command.Token
	}

	return invite, nil
}

func RevokeBandInvite(ctx context.Context, repository BandAccountRepository, input RevokeBandInviteInput) (BandInvite, error) {
	command, err := validateRevokeBandInviteInput(input)
	if err != nil {
		return BandInvite{}, err
	}

	invite, err := repository.RevokeBandInvite(ctx, command)
	if err != nil {
		return BandInvite{}, fmt.Errorf("revoke band invite band_id=%q invite_id=%q: %w", command.Account.BandID, command.InviteID, err)
	}

	return invite, nil
}

func AcceptBandInvite(ctx context.Context, repository BandAccountRepository, input AcceptBandInviteInput) (BandMember, error) {
	command, err := validateAcceptBandInviteInput(input)
	if err != nil {
		return BandMember{}, err
	}

	member, err := repository.AcceptBandInvite(ctx, command)
	if err != nil {
		return BandMember{}, fmt.Errorf("accept band invite token=%q email=%q provider=%q subject=%q: %w", redactedInviteToken(command.Token), command.Email, command.AuthProvider, command.AuthProviderUserID, err)
	}

	return member, nil
}

func GenerateInviteToken() (string, error) {
	buffer := make([]byte, inviteTokenByteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("read invite token randomness: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
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

func validateListBandMembersInput(input ListBandMembersInput) (ListBandMembersQuery, error) {
	if err := validateReadAccount(input.Account); err != nil {
		return ListBandMembersQuery{}, err
	}

	return ListBandMembersQuery{Account: input.Account}, nil
}

func validateListBandInvitesInput(input ListBandInvitesInput) (ListBandInvitesQuery, error) {
	if err := validateReadAccount(input.Account); err != nil {
		return ListBandInvitesQuery{}, err
	}

	return ListBandInvitesQuery{Account: input.Account}, nil
}

func validateCreateBandInviteInput(input CreateBandInviteInput) (CreateBandInviteCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return CreateBandInviteCommand{}, err
	}

	email, err := normalizeEmail(input.Email)
	if err != nil {
		return CreateBandInviteCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.CreatedAt)
	if err != nil {
		return CreateBandInviteCommand{}, err
	}

	return CreateBandInviteCommand{
		Account:        input.Account,
		Email:          email,
		Role:           permissions.RoleViewer,
		Status:         InviteStatusPending,
		ExpiresAt:      input.CreatedAt.UTC().Add(inviteExpirationDuration),
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      input.CreatedAt.UTC(),
	}, nil
}

func validateRevokeBandInviteInput(input RevokeBandInviteInput) (RevokeBandInviteCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return RevokeBandInviteCommand{}, err
	}

	inviteID := strings.TrimSpace(input.InviteID)
	if inviteID == "" {
		return RevokeBandInviteCommand{}, fmt.Errorf("invite id is required")
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.RevokedAt)
	if err != nil {
		return RevokeBandInviteCommand{}, err
	}

	return RevokeBandInviteCommand{
		Account:        input.Account,
		InviteID:       inviteID,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		RevokedAt:      input.RevokedAt.UTC(),
	}, nil
}

func validateAcceptBandInviteInput(input AcceptBandInviteInput) (AcceptBandInviteCommand, error) {
	authProvider := strings.TrimSpace(input.AuthProvider)
	if authProvider == "" {
		return AcceptBandInviteCommand{}, fmt.Errorf("auth provider is required")
	}

	authProviderUserID := strings.TrimSpace(input.AuthProviderUserID)
	if authProviderUserID == "" {
		return AcceptBandInviteCommand{}, fmt.Errorf("auth provider user id is required")
	}

	email, err := normalizeEmail(input.Email)
	if err != nil {
		return AcceptBandInviteCommand{}, err
	}

	token := strings.TrimSpace(input.Token)
	if token == "" {
		return AcceptBandInviteCommand{}, fmt.Errorf("invite token is required")
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.AcceptedAt)
	if err != nil {
		return AcceptBandInviteCommand{}, err
	}

	return AcceptBandInviteCommand{
		AuthProvider:       authProvider,
		AuthProviderUserID: authProviderUserID,
		Email:              email,
		Token:              token,
		IdempotencyKey:     idempotencyKey,
		RequestID:          requestID,
		AcceptedAt:         input.AcceptedAt.UTC(),
	}, nil
}

func validateReadAccount(account OwnerAccount) error {
	if strings.TrimSpace(account.UserID) == "" {
		return fmt.Errorf("user id is required")
	}

	if strings.TrimSpace(account.BandID) == "" {
		return fmt.Errorf("band id is required")
	}

	if !permissions.CanReadInAlpha(account.Role) {
		return fmt.Errorf("alpha read access denied for role %q", account.Role)
	}

	return nil
}

func validateWriteAccount(account OwnerAccount) error {
	if err := validateReadAccount(account); err != nil {
		return err
	}

	if err := permissions.RequireAlphaWrite(account.Role); err != nil {
		return err
	}

	return nil
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" {
		return "", fmt.Errorf("email is required")
	}

	if !strings.Contains(email, "@") {
		return "", fmt.Errorf("email %q must contain @", email)
	}

	return email, nil
}

func validateMutationMetadata(idempotencyKeyInput string, requestIDInput string, timestamp time.Time) (string, string, error) {
	idempotencyKey := strings.TrimSpace(idempotencyKeyInput)
	if idempotencyKey == "" {
		return "", "", fmt.Errorf("idempotency key is required")
	}

	requestID := strings.TrimSpace(requestIDInput)
	if requestID == "" {
		return "", "", fmt.Errorf("request id is required")
	}

	if timestamp.IsZero() {
		return "", "", fmt.Errorf("mutation timestamp is required")
	}

	return idempotencyKey, requestID, nil
}

func redactedInviteToken(token string) string {
	trimmedToken := strings.TrimSpace(token)
	if len(trimmedToken) <= 8 {
		return "[redacted]"
	}

	return trimmedToken[:4] + "[redacted]" + trimmedToken[len(trimmedToken)-4:]
}
