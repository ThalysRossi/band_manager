package supabase

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
)

const (
	providerName        = "supabase"
	authenticatedRole   = "authenticated"
	jwksFetchAttempts   = 3
	jwksFetchRetryDelay = 250 * time.Millisecond
)

type Authenticator struct {
	jwksURL    string
	issuer     string
	httpClient *http.Client
	logger     *slog.Logger
	mu         *sync.RWMutex
	keySet     *jwk.Set
}

func NewAuthenticator(ctx context.Context, supabaseURL string, httpClient *http.Client, logger *slog.Logger) (Authenticator, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	if baseURL == "" {
		return Authenticator{}, fmt.Errorf("supabase url is required")
	}
	if httpClient == nil {
		return Authenticator{}, fmt.Errorf("supabase jwks http client is required")
	}
	if logger == nil {
		return Authenticator{}, fmt.Errorf("supabase authenticator logger is required")
	}

	emptyKeySet := jwk.NewSet()
	authenticator := Authenticator{
		jwksURL:    baseURL + "/auth/v1/.well-known/jwks.json",
		issuer:     baseURL + "/auth/v1",
		httpClient: httpClient,
		logger:     logger,
		mu:         &sync.RWMutex{},
		keySet:     &emptyKeySet,
	}
	if err := authenticator.refreshKeySet(ctx); err != nil {
		return Authenticator{}, fmt.Errorf("load supabase jwks: %w", err)
	}

	return authenticator, nil
}

func (authenticator Authenticator) Authenticate(ctx context.Context, bearerToken string) (session.AuthenticatedUser, error) {
	if ctx == nil {
		return session.AuthenticatedUser{}, fmt.Errorf("context is required")
	}
	token := strings.TrimSpace(bearerToken)
	if token == "" {
		return session.AuthenticatedUser{}, fmt.Errorf("supabase bearer token is required")
	}

	keyID, err := tokenKeyID([]byte(token))
	if err != nil {
		return session.AuthenticatedUser{}, err
	}
	if !authenticator.hasKey(keyID) {
		if err := authenticator.refreshKeySet(ctx); err != nil {
			return session.AuthenticatedUser{}, fmt.Errorf("refresh supabase jwks for key id %q: %w", keyID, err)
		}
	}

	parsedToken, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(authenticator.currentKeySet()),
		jwt.WithIssuer(authenticator.issuer),
		jwt.WithAudience(authenticatedRole),
		jwt.WithRequiredClaim(jwt.SubjectKey),
		jwt.WithRequiredClaim(jwt.ExpirationKey),
		jwt.WithRequiredClaim("email"),
	)
	if err != nil {
		return session.AuthenticatedUser{}, fmt.Errorf("verify supabase access token: %w", err)
	}

	subject, ok := parsedToken.Subject()
	if !ok || strings.TrimSpace(subject) == "" {
		return session.AuthenticatedUser{}, fmt.Errorf("supabase access token subject is required")
	}

	var email string
	if err := parsedToken.Get("email", &email); err != nil || strings.TrimSpace(email) == "" {
		return session.AuthenticatedUser{}, fmt.Errorf("supabase access token email is required")
	}

	return session.AuthenticatedUser{
		Provider:       providerName,
		ProviderUserID: strings.TrimSpace(subject),
		Email:          strings.TrimSpace(email),
	}, nil
}

func (authenticator Authenticator) refreshKeySet(ctx context.Context) error {
	var lastErr error
	for attempt := 1; attempt <= jwksFetchAttempts; attempt++ {
		keySet, err := jwk.Fetch(ctx, authenticator.jwksURL, jwk.WithHTTPClient(authenticator.httpClient))
		if err == nil {
			if validateKeySetErr := validateAsymmetricKeySet(keySet); validateKeySetErr != nil {
				return validateKeySetErr
			}
			authenticator.mu.Lock()
			*authenticator.keySet = keySet
			authenticator.mu.Unlock()
			return nil
		}

		lastErr = err
		authenticator.logger.Warn("supabase jwks fetch failed", "error", err, "jwks_url", authenticator.jwksURL, "attempt", attempt)
		if attempt < jwksFetchAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(jwksFetchRetryDelay):
			}
		}
	}

	return fmt.Errorf("fetch supabase jwks url=%q attempts=%d: %w", authenticator.jwksURL, jwksFetchAttempts, lastErr)
}

func (authenticator Authenticator) hasKey(keyID string) bool {
	_, ok := authenticator.currentKeySet().LookupKeyID(keyID)
	return ok
}

func (authenticator Authenticator) currentKeySet() jwk.Set {
	authenticator.mu.RLock()
	defer authenticator.mu.RUnlock()
	return *authenticator.keySet
}

func validateAsymmetricKeySet(keySet jwk.Set) error {
	if keySet == nil || keySet.Len() == 0 {
		return fmt.Errorf("supabase jwks must contain at least one key")
	}

	for index := 0; index < keySet.Len(); index++ {
		key, ok := keySet.Key(index)
		if !ok {
			return fmt.Errorf("supabase jwks key at index %d is unavailable", index)
		}
		keyID, ok := key.KeyID()
		if !ok || strings.TrimSpace(keyID) == "" {
			return fmt.Errorf("supabase jwks key at index %d must have a key id", index)
		}
		algorithm, ok := key.Algorithm()
		if !ok {
			return fmt.Errorf("supabase jwks key %q must declare an algorithm", keyID)
		}
		if algorithm.String() == jwa.HS256().String() || algorithm.String() == jwa.HS384().String() || algorithm.String() == jwa.HS512().String() {
			return fmt.Errorf("supabase jwks key %q uses forbidden symmetric algorithm %q", keyID, algorithm)
		}
	}

	return nil
}

func tokenKeyID(token []byte) (string, error) {
	message, err := jws.Parse(token)
	if err != nil {
		return "", fmt.Errorf("parse supabase access token headers: %w", err)
	}
	signatures := message.Signatures()
	if len(signatures) != 1 {
		return "", fmt.Errorf("supabase access token must contain exactly one signature")
	}
	keyID, ok := signatures[0].ProtectedHeaders().KeyID()
	if !ok || strings.TrimSpace(keyID) == "" {
		return "", fmt.Errorf("supabase access token key id is required")
	}

	return strings.TrimSpace(keyID), nil
}
