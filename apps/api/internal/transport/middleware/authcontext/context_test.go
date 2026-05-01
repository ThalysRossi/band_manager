package authcontext

import (
	"context"
	"testing"

	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestWithContextRequiresValidRole(t *testing.T) {
	t.Parallel()

	_, err := WithContext(context.Background(), Context{
		UserID: "user_1",
		BandID: "band_1",
		Role:   permissions.Role("manager"),
	})
	if err == nil {
		t.Fatal("expected invalid role error")
	}
}

func TestFromContextReturnsAuthContext(t *testing.T) {
	t.Parallel()

	expected := Context{
		UserID: "user_1",
		BandID: "band_1",
		Role:   permissions.RoleOwner,
	}

	ctx, err := WithContext(context.Background(), expected)
	if err != nil {
		t.Fatalf("with auth context: %v", err)
	}

	actual, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected auth context")
	}

	if actual != expected {
		t.Fatalf("expected %+v, got %+v", expected, actual)
	}
}
