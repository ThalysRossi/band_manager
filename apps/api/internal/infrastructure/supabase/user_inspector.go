package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/application/session"
)

const (
	userInspectionAttempts   = 3
	userInspectionRetryDelay = 250 * time.Millisecond
)

type VerifiedUserInspector struct {
	userURL    string
	anonKey    string
	httpClient *http.Client
	logger     *slog.Logger
}

type userResponse struct {
	ID               string  `json:"id"`
	Email            string  `json:"email"`
	EmailConfirmedAt *string `json:"email_confirmed_at"`
}

func NewVerifiedUserInspector(supabaseURL string, anonKey string, httpClient *http.Client, logger *slog.Logger) (VerifiedUserInspector, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	if baseURL == "" {
		return VerifiedUserInspector{}, fmt.Errorf("supabase url is required")
	}
	trimmedAnonKey := strings.TrimSpace(anonKey)
	if trimmedAnonKey == "" {
		return VerifiedUserInspector{}, fmt.Errorf("supabase anon key is required")
	}
	if httpClient == nil {
		return VerifiedUserInspector{}, fmt.Errorf("supabase user inspector http client is required")
	}
	if logger == nil {
		return VerifiedUserInspector{}, fmt.Errorf("supabase user inspector logger is required")
	}

	return VerifiedUserInspector{
		userURL:    baseURL + "/auth/v1/user",
		anonKey:    trimmedAnonKey,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func (inspector VerifiedUserInspector) InspectVerifiedUser(ctx context.Context, bearerToken string) (session.VerifiedUser, error) {
	token := strings.TrimSpace(bearerToken)
	if token == "" {
		return session.VerifiedUser{}, fmt.Errorf("supabase bearer token is required")
	}

	var lastErr error
	for attempt := 1; attempt <= userInspectionAttempts; attempt++ {
		user, retry, err := inspector.inspect(ctx, token)
		if err == nil {
			return user, nil
		}
		lastErr = err
		if !retry {
			return session.VerifiedUser{}, err
		}

		inspector.logger.Warn("supabase verified user inspection failed", "error", err, "user_url", inspector.userURL, "attempt", attempt)
		if attempt < userInspectionAttempts {
			select {
			case <-ctx.Done():
				return session.VerifiedUser{}, ctx.Err()
			case <-time.After(userInspectionRetryDelay):
			}
		}
	}

	return session.VerifiedUser{}, fmt.Errorf("inspect supabase verified user url=%q attempts=%d: %w", inspector.userURL, userInspectionAttempts, lastErr)
}

func (inspector VerifiedUserInspector) inspect(ctx context.Context, bearerToken string) (session.VerifiedUser, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, inspector.userURL, nil)
	if err != nil {
		return session.VerifiedUser{}, false, fmt.Errorf("create supabase user request url=%q: %w", inspector.userURL, err)
	}
	request.Header.Set("Authorization", "Bearer "+bearerToken)
	request.Header.Set("apikey", inspector.anonKey)

	response, err := inspector.httpClient.Do(request)
	if err != nil {
		return session.VerifiedUser{}, true, fmt.Errorf("execute supabase user request url=%q: %w", inspector.userURL, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return session.VerifiedUser{}, true, fmt.Errorf("read supabase user response url=%q status_code=%d: %w", inspector.userURL, response.StatusCode, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		retry := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
		return session.VerifiedUser{}, retry, fmt.Errorf("supabase user request failed url=%q status_code=%d response_body=%q", inspector.userURL, response.StatusCode, string(body))
	}

	var user userResponse
	if err := json.Unmarshal(body, &user); err != nil {
		return session.VerifiedUser{}, false, fmt.Errorf("parse supabase user response url=%q status_code=%d response_body=%q: %w", inspector.userURL, response.StatusCode, string(body), err)
	}
	if strings.TrimSpace(user.ID) == "" {
		return session.VerifiedUser{}, false, fmt.Errorf("supabase verified user id is required")
	}
	if strings.TrimSpace(user.Email) == "" {
		return session.VerifiedUser{}, false, fmt.Errorf("supabase verified user email is required")
	}
	if user.EmailConfirmedAt == nil || strings.TrimSpace(*user.EmailConfirmedAt) == "" {
		return session.VerifiedUser{}, false, fmt.Errorf("supabase user email is not verified: %w", session.ErrEmailNotVerified)
	}

	return session.VerifiedUser{
		Provider:       providerName,
		ProviderUserID: strings.TrimSpace(user.ID),
		Email:          strings.TrimSpace(user.Email),
	}, false, nil
}
