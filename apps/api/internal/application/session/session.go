package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrEmailNotVerified = errors.New("email is not verified")

type AuthenticatedUser struct {
	Provider       string
	ProviderUserID string
	Email          string
}

type VerifiedUser struct {
	Provider       string
	ProviderUserID string
	Email          string
}

type Authenticator interface {
	Authenticate(ctx context.Context, bearerToken string) (AuthenticatedUser, error)
}

type VerifiedUserInspector interface {
	InspectVerifiedUser(ctx context.Context, bearerToken string) (VerifiedUser, error)
}

type authenticatedContext struct {
	User        AuthenticatedUser
	BearerToken string
}

type authenticatedContextKey struct{}
type verifiedContextKey struct{}

func WithAuthenticatedUser(ctx context.Context, user AuthenticatedUser, bearerToken string) (context.Context, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if strings.TrimSpace(user.Provider) == "" {
		return nil, fmt.Errorf("authenticated user provider is required")
	}
	if strings.TrimSpace(user.ProviderUserID) == "" {
		return nil, fmt.Errorf("authenticated user provider id is required")
	}
	if strings.TrimSpace(user.Email) == "" {
		return nil, fmt.Errorf("authenticated user email is required")
	}
	token := strings.TrimSpace(bearerToken)
	if token == "" {
		return nil, fmt.Errorf("authenticated bearer token is required")
	}

	return context.WithValue(ctx, authenticatedContextKey{}, authenticatedContext{
		User:        user,
		BearerToken: token,
	}), nil
}

func AuthenticatedUserFromContext(ctx context.Context) (AuthenticatedUser, string, bool) {
	authenticated, ok := ctx.Value(authenticatedContextKey{}).(authenticatedContext)
	return authenticated.User, authenticated.BearerToken, ok
}

func WithVerifiedUser(ctx context.Context, user VerifiedUser) (context.Context, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if strings.TrimSpace(user.Provider) == "" {
		return nil, fmt.Errorf("verified user provider is required")
	}
	if strings.TrimSpace(user.ProviderUserID) == "" {
		return nil, fmt.Errorf("verified user provider id is required")
	}
	if strings.TrimSpace(user.Email) == "" {
		return nil, fmt.Errorf("verified user email is required")
	}

	return context.WithValue(ctx, verifiedContextKey{}, user), nil
}

func VerifiedUserFromContext(ctx context.Context) (VerifiedUser, bool) {
	user, ok := ctx.Value(verifiedContextKey{}).(VerifiedUser)
	return user, ok
}

func NormalizeBearerToken(authorizationHeader string) (string, error) {
	header := strings.TrimSpace(authorizationHeader)
	if header == "" {
		return "", fmt.Errorf("authorization header is required")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", fmt.Errorf("authorization header must use Bearer scheme")
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", fmt.Errorf("bearer token is required")
	}

	return token, nil
}
