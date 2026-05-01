package session

import "testing"

func TestNormalizeBearerTokenRequiresBearerScheme(t *testing.T) {
	t.Parallel()

	_, err := NormalizeBearerToken("Basic abc")
	if err == nil {
		t.Fatal("expected bearer scheme error")
	}
}

func TestNormalizeBearerTokenReturnsToken(t *testing.T) {
	t.Parallel()

	token, err := NormalizeBearerToken("Bearer abc.def.ghi")
	if err != nil {
		t.Fatalf("normalize bearer token: %v", err)
	}

	if token != "abc.def.ghi" {
		t.Fatalf("expected token, got %q", token)
	}
}
