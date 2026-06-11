package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware/authcontext"
)

type authenticationErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Authenticate(authenticator session.Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			token, err := session.NormalizeBearerToken(request.Header.Get("Authorization"))
			if err != nil {
				writeAuthenticationError(response, http.StatusUnauthorized, "invalid_authorization", err.Error())
				return
			}

			user, err := authenticator.Authenticate(request.Context(), token)
			if err != nil {
				logger.Warn("request authentication failed", "error", err)
				writeAuthenticationError(response, http.StatusUnauthorized, "invalid_session", "Session is missing or invalid")
				return
			}

			ctx, err := session.WithAuthenticatedUser(request.Context(), user, token)
			if err != nil {
				logger.Error("authenticated request context creation failed", "error", err)
				writeAuthenticationError(response, http.StatusInternalServerError, "authentication_context_failed", "Authentication context could not be created")
				return
			}

			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

func RequireVerifiedIdentity(inspector session.VerifiedUserInspector, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			authenticatedUser, bearerToken, ok := session.AuthenticatedUserFromContext(request.Context())
			if !ok {
				writeAuthenticationError(response, http.StatusInternalServerError, "missing_authenticated_user", "Authenticated user context is missing")
				return
			}

			verifiedUser, err := inspector.InspectVerifiedUser(request.Context(), bearerToken)
			if err != nil {
				logger.Warn("verified user inspection failed", "error", err, "provider", authenticatedUser.Provider, "provider_user_id", authenticatedUser.ProviderUserID)
				if errors.Is(err, session.ErrEmailNotVerified) {
					writeAuthenticationError(response, http.StatusForbidden, "email_verification_required", "Email verification is required before creating application records")
					return
				}
				writeAuthenticationError(response, http.StatusBadGateway, "identity_provider_unavailable", "Verified identity could not be inspected")
				return
			}
			if verifiedUser.Provider != authenticatedUser.Provider ||
				verifiedUser.ProviderUserID != authenticatedUser.ProviderUserID ||
				!strings.EqualFold(verifiedUser.Email, authenticatedUser.Email) {
				writeAuthenticationError(response, http.StatusForbidden, "verified_identity_mismatch", "Verified provider identity does not match the authenticated session")
				return
			}

			ctx, err := session.WithVerifiedUser(request.Context(), verifiedUser)
			if err != nil {
				logger.Error("verified user context creation failed", "error", err)
				writeAuthenticationError(response, http.StatusInternalServerError, "verification_context_failed", "Verified identity context could not be created")
				return
			}

			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

func ResolveAccount(repository accounts.BandAccountRepository, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			user, _, ok := session.AuthenticatedUserFromContext(request.Context())
			if !ok {
				writeAuthenticationError(response, http.StatusInternalServerError, "missing_authenticated_user", "Authenticated user context is missing")
				return
			}

			account, err := accounts.GetCurrentAccount(request.Context(), repository, accounts.CurrentAccountQuery{
				AuthProvider:       user.Provider,
				AuthProviderUserID: user.ProviderUserID,
			})
			if err != nil {
				if errors.Is(err, accounts.ErrAccountNotFound) {
					writeAuthenticationError(response, http.StatusNotFound, "account_not_found", "Authenticated user has not completed onboarding")
					return
				}
				logger.Warn("account context resolution failed", "error", err, "provider", user.Provider, "provider_user_id", user.ProviderUserID)
				writeAuthenticationError(response, http.StatusInternalServerError, "account_context_failed", "Account context could not be resolved")
				return
			}

			ctx, err := authcontext.WithContext(request.Context(), authcontext.Context{
				UserID:       account.UserID,
				BandID:       account.BandID,
				Email:        account.Email,
				BandName:     account.BandName,
				BandTimezone: account.BandTimezone,
				Role:         account.Role,
			})
			if err != nil {
				logger.Error("account request context creation failed", "error", err)
				writeAuthenticationError(response, http.StatusInternalServerError, "account_context_failed", "Account context could not be created")
				return
			}

			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

func writeAuthenticationError(response http.ResponseWriter, statusCode int, code string, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)
	_ = json.NewEncoder(response).Encode(authenticationErrorResponse{Code: code, Message: message})
}
