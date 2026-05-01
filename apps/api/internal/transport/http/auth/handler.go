package authhandler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
)

type Handler struct {
	authenticator session.Authenticator
	repository    accounts.BandAccountRepository
	logger        *slog.Logger
	now           func() time.Time
}

type SignupOwnerRequest struct {
	Email        string `json:"email"`
	BandName     string `json:"bandName"`
	BandTimezone string `json:"bandTimezone"`
}

type CurrentAccountResponse struct {
	User       UserResponse           `json:"user"`
	ActiveBand BandMembershipResponse `json:"activeBand"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type BandMembershipResponse struct {
	BandID   string           `json:"bandId"`
	BandName string           `json:"bandName"`
	Role     permissions.Role `json:"role"`
	CanWrite bool             `json:"canWrite"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(authenticator session.Authenticator, repository accounts.BandAccountRepository, logger *slog.Logger) Handler {
	return Handler{
		authenticator: authenticator,
		repository:    repository,
		logger:        logger,
		now:           time.Now,
	}
}

func (handler Handler) SignupOwner(response http.ResponseWriter, request *http.Request) {
	authenticatedUser, ok := handler.authenticate(response, request)
	if !ok {
		return
	}

	idempotencyKey := request.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		handler.writeError(response, http.StatusBadRequest, "missing_idempotency_key", "Idempotency-Key header is required")
		return
	}

	requestID, ok := middleware.RequestIDFromContext(request.Context())
	if !ok {
		handler.writeError(response, http.StatusInternalServerError, "missing_request_id", "request id is missing")
		return
	}

	var body SignupOwnerRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		handler.writeError(response, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	if !strings.EqualFold(strings.TrimSpace(body.Email), authenticatedUser.Email) {
		handler.writeError(response, http.StatusBadRequest, "email_mismatch", "Signup email must match the authenticated user email")
		return
	}

	account, err := accounts.CreateOwnerAccount(request.Context(), handler.repository, accounts.CreateOwnerAccountInput{
		AuthProvider:       authenticatedUser.Provider,
		AuthProviderUserID: authenticatedUser.ProviderUserID,
		Email:              body.Email,
		BandName:           body.BandName,
		BandTimezone:       body.BandTimezone,
		IdempotencyKey:     idempotencyKey,
		RequestID:          requestID,
		CreatedAt:          handler.now().UTC(),
	})
	if err != nil {
		handler.logger.Warn("owner signup failed", "error", err, "email", body.Email, "provider", authenticatedUser.Provider, "provider_user_id", authenticatedUser.ProviderUserID)
		handler.writeError(response, http.StatusBadRequest, "signup_failed", err.Error())
		return
	}

	handler.writeCurrentAccount(response, http.StatusCreated, toCurrentAccountResponse(account))
}

func (handler Handler) GetCurrentAccount(response http.ResponseWriter, request *http.Request) {
	authenticatedUser, ok := handler.authenticate(response, request)
	if !ok {
		return
	}

	account, err := accounts.GetCurrentAccount(request.Context(), handler.repository, accounts.CurrentAccountQuery{
		AuthProvider:       authenticatedUser.Provider,
		AuthProviderUserID: authenticatedUser.ProviderUserID,
	})
	if err != nil {
		handler.logger.Warn("current account lookup failed", "error", err, "provider", authenticatedUser.Provider, "provider_user_id", authenticatedUser.ProviderUserID)
		handler.writeError(response, http.StatusUnauthorized, "account_not_found", "Authenticated user is not linked to a band account")
		return
	}

	handler.writeCurrentAccount(response, http.StatusOK, toCurrentAccountResponse(account))
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

func toCurrentAccountResponse(account accounts.OwnerAccount) CurrentAccountResponse {
	return CurrentAccountResponse{
		User: UserResponse{
			ID:    account.UserID,
			Email: account.Email,
		},
		ActiveBand: BandMembershipResponse{
			BandID:   account.BandID,
			BandName: account.BandName,
			Role:     account.Role,
			CanWrite: permissions.CanWriteInAlpha(account.Role),
		},
	}
}

func (handler Handler) writeCurrentAccount(response http.ResponseWriter, statusCode int, body CurrentAccountResponse) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	if err := json.NewEncoder(response).Encode(body); err != nil {
		handler.logger.Error("current account response encoding failed", "error", err)
	}
}

func (handler Handler) writeError(response http.ResponseWriter, statusCode int, code string, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	err := json.NewEncoder(response).Encode(ErrorResponse{
		Code:    code,
		Message: message,
	})
	if err != nil {
		handler.logger.Error("error response encoding failed", "error", err, "code", code, "status_code", statusCode)
	}
}
