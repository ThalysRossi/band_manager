package authhandler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware/authcontext"
)

type Handler struct {
	repository accounts.BandAccountRepository
	logger     *slog.Logger
	now        func() time.Time
}

type OnboardOwnerRequest struct {
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

func NewHandler(repository accounts.BandAccountRepository, logger *slog.Logger) Handler {
	return Handler{
		repository: repository,
		logger:     logger,
		now:        time.Now,
	}
}

func (handler Handler) OnboardOwner(response http.ResponseWriter, request *http.Request) {
	verifiedUser, ok := session.VerifiedUserFromContext(request.Context())
	if !ok {
		handler.writeError(response, http.StatusInternalServerError, "missing_verified_user", "Verified user context is missing")
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

	var body OnboardOwnerRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		handler.writeError(response, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	account, err := accounts.CreateOwnerAccount(request.Context(), handler.repository, accounts.CreateOwnerAccountInput{
		AuthProvider:       verifiedUser.Provider,
		AuthProviderUserID: verifiedUser.ProviderUserID,
		Email:              verifiedUser.Email,
		BandName:           body.BandName,
		BandTimezone:       body.BandTimezone,
		IdempotencyKey:     idempotencyKey,
		RequestID:          requestID,
		CreatedAt:          handler.now().UTC(),
	})
	if err != nil {
		handler.logger.Warn("owner onboarding failed", "error", err, "email", verifiedUser.Email, "provider", verifiedUser.Provider, "provider_user_id", verifiedUser.ProviderUserID)
		handler.writeError(response, http.StatusBadRequest, "onboarding_failed", err.Error())
		return
	}

	handler.writeCurrentAccount(response, http.StatusCreated, toCurrentAccountResponse(account))
}

func (handler Handler) GetCurrentAccount(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := authcontext.FromContext(request.Context())
	if !ok {
		handler.writeError(response, http.StatusInternalServerError, "missing_account_context", "Account context is missing")
		return
	}

	handler.writeCurrentAccount(response, http.StatusOK, toCurrentAccountResponse(accounts.OwnerAccount{
		UserID:       accountContext.UserID,
		BandID:       accountContext.BandID,
		Email:        accountContext.Email,
		BandName:     accountContext.BandName,
		BandTimezone: accountContext.BandTimezone,
		Role:         accountContext.Role,
	}))
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
