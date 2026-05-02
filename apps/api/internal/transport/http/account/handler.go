package accounthandler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
	authhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/auth"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
)

type Handler struct {
	authenticator  session.Authenticator
	repository     accounts.BandAccountRepository
	logger         *slog.Logger
	now            func() time.Time
	tokenGenerator accounts.InviteTokenGenerator
}

type CreateInviteRequest struct {
	Email string `json:"email"`
}

type AcceptInviteRequest struct {
	Token string `json:"token"`
}

type MembersResponse struct {
	Members []MemberResponse `json:"members"`
}

type InvitesResponse struct {
	Invites []InviteResponse `json:"invites"`
}

type MemberResponse struct {
	UserID   string           `json:"userId"`
	Email    string           `json:"email"`
	BandID   string           `json:"bandId"`
	Role     permissions.Role `json:"role"`
	JoinedAt time.Time        `json:"joinedAt"`
}

type InviteResponse struct {
	ID        string                `json:"id"`
	Email     string                `json:"email"`
	Role      permissions.Role      `json:"role"`
	Status    accounts.InviteStatus `json:"status"`
	ExpiresAt time.Time             `json:"expiresAt"`
	CreatedAt time.Time             `json:"createdAt"`
	UpdatedAt time.Time             `json:"updatedAt"`
	Token     string                `json:"token,omitempty"`
}

func NewHandler(authenticator session.Authenticator, repository accounts.BandAccountRepository, logger *slog.Logger) Handler {
	return Handler{
		authenticator:  authenticator,
		repository:     repository,
		logger:         logger,
		now:            time.Now,
		tokenGenerator: accounts.GenerateInviteToken,
	}
}

func (handler Handler) ListMembers(response http.ResponseWriter, request *http.Request) {
	account, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	members, err := accounts.ListBandMembers(request.Context(), handler.repository, accounts.ListBandMembersInput{
		Account: account,
	})
	if err != nil {
		handler.writeAccountError(response, "account members list failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, MembersResponse{Members: toMemberResponses(members)})
}

func (handler Handler) ListInvites(response http.ResponseWriter, request *http.Request) {
	account, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	invites, err := accounts.ListBandInvites(request.Context(), handler.repository, accounts.ListBandInvitesInput{
		Account: account,
	})
	if err != nil {
		handler.writeAccountError(response, "account invites list failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, InvitesResponse{Invites: toInviteResponses(invites)})
}

func (handler Handler) CreateInvite(response http.ResponseWriter, request *http.Request) {
	account, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body CreateInviteRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	invite, err := accounts.CreateBandInvite(request.Context(), handler.repository, handler.tokenGenerator, accounts.CreateBandInviteInput{
		Account:        account,
		Email:          body.Email,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeAccountError(response, "account invite create failed", err)
		return
	}

	handler.writeJSON(response, http.StatusCreated, toInviteResponse(invite))
}

func (handler Handler) RevokeInvite(response http.ResponseWriter, request *http.Request) {
	account, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	invite, err := accounts.RevokeBandInvite(request.Context(), handler.repository, accounts.RevokeBandInviteInput{
		Account:        account,
		InviteID:       chi.URLParam(request, "inviteID"),
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		RevokedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeAccountError(response, "account invite revoke failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, toInviteResponse(invite))
}

func (handler Handler) AcceptInvite(response http.ResponseWriter, request *http.Request) {
	authenticatedUser, ok := handler.authenticate(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body AcceptInviteRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	member, err := accounts.AcceptBandInvite(request.Context(), handler.repository, accounts.AcceptBandInviteInput{
		AuthProvider:       authenticatedUser.Provider,
		AuthProviderUserID: authenticatedUser.ProviderUserID,
		Email:              authenticatedUser.Email,
		Token:              body.Token,
		IdempotencyKey:     idempotencyKey,
		RequestID:          requestID,
		AcceptedAt:         handler.now().UTC(),
	})
	if err != nil {
		handler.writeAccountError(response, "account invite accept failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, toMemberResponse(member))
}

func (handler Handler) accountContext(response http.ResponseWriter, request *http.Request) (accounts.OwnerAccount, bool) {
	authenticatedUser, ok := handler.authenticate(response, request)
	if !ok {
		return accounts.OwnerAccount{}, false
	}

	account, err := accounts.GetCurrentAccount(request.Context(), handler.repository, accounts.CurrentAccountQuery{
		AuthProvider:       authenticatedUser.Provider,
		AuthProviderUserID: authenticatedUser.ProviderUserID,
	})
	if err != nil {
		handler.logger.Warn("account lookup failed", "error", err, "provider", authenticatedUser.Provider, "provider_user_id", authenticatedUser.ProviderUserID)
		handler.writeError(response, http.StatusUnauthorized, "account_not_found", "Authenticated user is not linked to a band account")
		return accounts.OwnerAccount{}, false
	}

	return account, true
}

func (handler Handler) authenticate(response http.ResponseWriter, request *http.Request) (session.AuthenticatedUser, bool) {
	token, err := session.NormalizeBearerToken(request.Header.Get("Authorization"))
	if err != nil {
		handler.writeError(response, http.StatusUnauthorized, "invalid_authorization", err.Error())
		return session.AuthenticatedUser{}, false
	}

	authenticatedUser, err := handler.authenticator.Authenticate(request.Context(), token)
	if err != nil {
		handler.writeError(response, http.StatusUnauthorized, "invalid_session", err.Error())
		return session.AuthenticatedUser{}, false
	}

	return authenticatedUser, true
}

func (handler Handler) mutationHeaders(response http.ResponseWriter, request *http.Request) (string, string, bool) {
	idempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		handler.writeError(response, http.StatusBadRequest, "missing_idempotency_key", "Idempotency-Key header is required")
		return "", "", false
	}

	requestID, ok := middleware.RequestIDFromContext(request.Context())
	if !ok {
		handler.writeError(response, http.StatusInternalServerError, "missing_request_id", "request id is missing")
		return "", "", false
	}

	return idempotencyKey, requestID, true
}

func (handler Handler) decodeJSON(response http.ResponseWriter, request *http.Request, target interface{}) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		handler.writeError(response, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return false
	}

	return true
}

func (handler Handler) writeAccountError(response http.ResponseWriter, logMessage string, err error) {
	handler.logger.Warn(logMessage, "error", err)

	switch {
	case errors.Is(err, accounts.ErrInviteNotFound):
		handler.writeError(response, http.StatusNotFound, "invite_not_found", err.Error())
	case errors.Is(err, accounts.ErrDuplicatePendingInvite):
		handler.writeError(response, http.StatusConflict, "duplicate_pending_invite", err.Error())
	case strings.Contains(err.Error(), "alpha write access requires owner role"):
		handler.writeError(response, http.StatusForbidden, "write_forbidden", err.Error())
	case strings.Contains(err.Error(), "alpha read access denied"):
		handler.writeError(response, http.StatusForbidden, "read_forbidden", err.Error())
	case errors.Is(err, accounts.ErrInviteExpired):
		handler.writeError(response, http.StatusBadRequest, "invite_expired", err.Error())
	case errors.Is(err, accounts.ErrInviteRevoked):
		handler.writeError(response, http.StatusBadRequest, "invite_revoked", err.Error())
	case errors.Is(err, accounts.ErrInviteAccepted):
		handler.writeError(response, http.StatusBadRequest, "invite_accepted", err.Error())
	case errors.Is(err, accounts.ErrInviteEmailMismatch):
		handler.writeError(response, http.StatusBadRequest, "invite_email_mismatch", err.Error())
	case errors.Is(err, accounts.ErrMembershipConflict):
		handler.writeError(response, http.StatusConflict, "membership_conflict", err.Error())
	case strings.Contains(err.Error(), "list band members"),
		strings.Contains(err.Error(), "list band invites"),
		strings.Contains(err.Error(), "create band invite"),
		strings.Contains(err.Error(), "revoke band invite"),
		strings.Contains(err.Error(), "accept band invite"):
		handler.writeError(response, http.StatusInternalServerError, "account_management_failed", err.Error())
	default:
		handler.writeError(response, http.StatusBadRequest, "invalid_account_request", err.Error())
	}
}

func (handler Handler) writeJSON(response http.ResponseWriter, statusCode int, body interface{}) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	if err := json.NewEncoder(response).Encode(body); err != nil {
		handler.logger.Error("account response encoding failed", "error", err)
	}
}

func (handler Handler) writeError(response http.ResponseWriter, statusCode int, code string, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	err := json.NewEncoder(response).Encode(authhandler.ErrorResponse{
		Code:    code,
		Message: message,
	})
	if err != nil {
		handler.logger.Error("account error response encoding failed", "error", err, "code", code, "status_code", statusCode)
	}
}

func toMemberResponses(members []accounts.BandMember) []MemberResponse {
	responses := make([]MemberResponse, 0, len(members))
	for _, member := range members {
		responses = append(responses, toMemberResponse(member))
	}

	return responses
}

func toMemberResponse(member accounts.BandMember) MemberResponse {
	return MemberResponse{
		UserID:   member.UserID,
		Email:    member.Email,
		BandID:   member.BandID,
		Role:     member.Role,
		JoinedAt: member.JoinedAt,
	}
}

func toInviteResponses(invites []accounts.BandInvite) []InviteResponse {
	responses := make([]InviteResponse, 0, len(invites))
	for _, invite := range invites {
		responses = append(responses, toInviteResponse(invite))
	}

	return responses
}

func toInviteResponse(invite accounts.BandInvite) InviteResponse {
	token := ""
	if invite.Status == accounts.InviteStatusPending {
		token = invite.Token
	}

	return InviteResponse{
		ID:        invite.ID,
		Email:     invite.Email,
		Role:      invite.Role,
		Status:    invite.Status,
		ExpiresAt: invite.ExpiresAt,
		CreatedAt: invite.CreatedAt,
		UpdatedAt: invite.UpdatedAt,
		Token:     token,
	}
}
