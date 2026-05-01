package session

import (
	"context"
	"fmt"
	"strings"
)

type AuthenticatedUser struct {
	Provider       string
	ProviderUserID string
	Email          string
}

type Authenticator interface {
	Authenticate(ctx context.Context, bearerToken string) (AuthenticatedUser, error)
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
