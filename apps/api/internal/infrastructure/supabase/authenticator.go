package supabase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/application/session"
)

const providerName = "supabase"

type Authenticator struct {
	jwtSecret []byte
	now       func() time.Time
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtClaims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Expiry  int64  `json:"exp"`
}

func NewAuthenticator(jwtSecret string) (Authenticator, error) {
	secret := strings.TrimSpace(jwtSecret)
	if secret == "" {
		return Authenticator{}, fmt.Errorf("supabase jwt secret is required")
	}

	return Authenticator{
		jwtSecret: []byte(secret),
		now:       time.Now,
	}, nil
}

func (authenticator Authenticator) Authenticate(ctx context.Context, bearerToken string) (session.AuthenticatedUser, error) {
	if ctx == nil {
		return session.AuthenticatedUser{}, fmt.Errorf("context is required")
	}

	claims, err := authenticator.verifyToken(bearerToken)
	if err != nil {
		return session.AuthenticatedUser{}, err
	}

	return session.AuthenticatedUser{
		Provider:       providerName,
		ProviderUserID: claims.Subject,
		Email:          claims.Email,
	}, nil
}

func (authenticator Authenticator) verifyToken(token string) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, fmt.Errorf("supabase jwt must have three parts")
	}

	headerBytes, err := decodeJWTPart(parts[0])
	if err != nil {
		return jwtClaims{}, fmt.Errorf("decode supabase jwt header: %w", err)
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return jwtClaims{}, fmt.Errorf("parse supabase jwt header: %w", err)
	}

	if header.Algorithm != "HS256" {
		return jwtClaims{}, fmt.Errorf("unsupported supabase jwt algorithm %q", header.Algorithm)
	}

	expectedSignature := signJWTInput(parts[0]+"."+parts[1], authenticator.jwtSecret)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return jwtClaims{}, fmt.Errorf("supabase jwt signature is invalid")
	}

	claimsBytes, err := decodeJWTPart(parts[1])
	if err != nil {
		return jwtClaims{}, fmt.Errorf("decode supabase jwt claims: %w", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return jwtClaims{}, fmt.Errorf("parse supabase jwt claims: %w", err)
	}

	if strings.TrimSpace(claims.Subject) == "" {
		return jwtClaims{}, fmt.Errorf("supabase jwt subject is required")
	}

	if strings.TrimSpace(claims.Email) == "" {
		return jwtClaims{}, fmt.Errorf("supabase jwt email is required")
	}

	if claims.Expiry <= authenticator.now().Unix() {
		return jwtClaims{}, fmt.Errorf("supabase jwt is expired")
	}

	return claims, nil
}

func decodeJWTPart(value string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func signJWTInput(value string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(value))
	signature := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(signature)
}
