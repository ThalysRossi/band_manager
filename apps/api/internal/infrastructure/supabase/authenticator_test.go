package supabase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
	"time"
)

func TestAuthenticatorRejectsInvalidSignature(t *testing.T) {
	t.Parallel()

	authenticator, err := NewAuthenticator("secret")
	if err != nil {
		t.Fatalf("new authenticator: %v", err)
	}
	authenticator.now = func() time.Time {
		return time.Unix(10, 0)
	}

	_, err = authenticator.Authenticate(context.Background(), signedToken("other-secret", "user_1", "band@example.com", 100))
	if err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestAuthenticatorReturnsAuthenticatedUser(t *testing.T) {
	t.Parallel()

	authenticator, err := NewAuthenticator("secret")
	if err != nil {
		t.Fatalf("new authenticator: %v", err)
	}
	authenticator.now = func() time.Time {
		return time.Unix(10, 0)
	}

	user, err := authenticator.Authenticate(context.Background(), signedToken("secret", "user_1", "band@example.com", 100))
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}

	if user.Provider != "supabase" {
		t.Fatalf("expected supabase provider, got %q", user.Provider)
	}

	if user.ProviderUserID != "user_1" {
		t.Fatalf("expected provider user id, got %q", user.ProviderUserID)
	}
}

func signedToken(secret string, subject string, email string, expiry int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + subject + `","email":"` + email + `","exp":` + strconv.FormatInt(expiry, 10) + `}`))
	signingInput := header + "." + claims
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature
}
