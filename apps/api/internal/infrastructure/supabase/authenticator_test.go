package supabase

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func TestAuthenticatorReturnsAuthenticatedUserForJWKSAccessToken(t *testing.T) {
	t.Parallel()

	privateKey, jwksBody := signingFixture(t)
	httpClient := &http.Client{Transport: responseTransport{statusCode: http.StatusOK, body: jwksBody}}
	authenticator, err := NewAuthenticator(context.Background(), "https://example.supabase.co", httpClient, slog.Default())
	if err != nil {
		t.Fatalf("new authenticator: %v", err)
	}

	accessToken := signedAccessToken(t, privateKey, "https://example.supabase.co/auth/v1", "authenticated")
	user, err := authenticator.Authenticate(context.Background(), accessToken)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if user.Provider != "supabase" || user.ProviderUserID != "user_1" || user.Email != "band@example.com" {
		t.Fatalf("unexpected authenticated user: %#v", user)
	}
}

func TestAuthenticatorRejectsInvalidAudience(t *testing.T) {
	t.Parallel()

	privateKey, jwksBody := signingFixture(t)
	httpClient := &http.Client{Transport: responseTransport{statusCode: http.StatusOK, body: jwksBody}}
	authenticator, err := NewAuthenticator(context.Background(), "https://example.supabase.co", httpClient, slog.Default())
	if err != nil {
		t.Fatalf("new authenticator: %v", err)
	}

	_, err = authenticator.Authenticate(context.Background(), signedAccessToken(t, privateKey, "https://example.supabase.co/auth/v1", "anon"))
	if err == nil {
		t.Fatal("expected invalid audience error")
	}
}

func TestAuthenticatorRejectsSymmetricJWKS(t *testing.T) {
	t.Parallel()

	key, err := jwk.Import([]byte("legacy-shared-secret"))
	if err != nil {
		t.Fatalf("import symmetric key: %v", err)
	}
	if err := key.Set(jwk.KeyIDKey, "legacy-key"); err != nil {
		t.Fatalf("set key id: %v", err)
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.HS256()); err != nil {
		t.Fatalf("set algorithm: %v", err)
	}
	keySet := jwk.NewSet()
	if err := keySet.AddKey(key); err != nil {
		t.Fatalf("add key: %v", err)
	}
	body, err := json.Marshal(keySet)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	_, err = NewAuthenticator(context.Background(), "https://example.supabase.co", &http.Client{Transport: responseTransport{statusCode: http.StatusOK, body: string(body)}}, slog.Default())
	if err == nil {
		t.Fatal("expected symmetric jwks rejection")
	}
}

func TestVerifiedUserInspectorRequiresConfirmedEmail(t *testing.T) {
	t.Parallel()

	inspector, err := NewVerifiedUserInspector(
		"https://example.supabase.co",
		"anon-key",
		&http.Client{Transport: responseTransport{statusCode: http.StatusOK, body: `{"id":"user_1","email":"band@example.com","email_confirmed_at":null}`}},
		slog.Default(),
	)
	if err != nil {
		t.Fatalf("new inspector: %v", err)
	}

	_, err = inspector.InspectVerifiedUser(context.Background(), "access-token")
	if err == nil {
		t.Fatal("expected unverified email rejection")
	}
}

func signingFixture(t *testing.T) (jwk.Key, string) {
	t.Helper()
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	privateKey, err := jwk.Import(rsaKey)
	if err != nil {
		t.Fatalf("import private key: %v", err)
	}
	if err := privateKey.Set(jwk.KeyIDKey, "key-1"); err != nil {
		t.Fatalf("set private key id: %v", err)
	}
	if err := privateKey.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		t.Fatalf("set private algorithm: %v", err)
	}
	publicKey, err := privateKey.PublicKey()
	if err != nil {
		t.Fatalf("create public key: %v", err)
	}
	keySet := jwk.NewSet()
	if err := keySet.AddKey(publicKey); err != nil {
		t.Fatalf("add public key: %v", err)
	}
	body, err := json.Marshal(keySet)
	if err != nil {
		t.Fatalf("marshal key set: %v", err)
	}

	return privateKey, string(body)
}

func signedAccessToken(t *testing.T, key jwk.Key, issuer string, audience string) string {
	t.Helper()
	token, err := jwt.NewBuilder().
		Issuer(issuer).
		Audience([]string{audience}).
		Subject("user_1").
		Expiration(time.Now().Add(time.Hour)).
		Claim("email", "band@example.com").
		Build()
	if err != nil {
		t.Fatalf("build token: %v", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), key))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return string(signed)
}

type responseTransport struct {
	statusCode int
	body       string
}

func (transport responseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: transport.statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(transport.body)),
		Request:    request,
	}, nil
}
