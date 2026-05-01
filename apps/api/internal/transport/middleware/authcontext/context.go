package authcontext

import (
	"context"
	"fmt"

	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

type Context struct {
	UserID string
	BandID string
	Role   permissions.Role
}

type contextKey struct{}

func WithContext(ctx context.Context, authContext Context) (context.Context, error) {
	if authContext.UserID == "" {
		return nil, fmt.Errorf("auth context user id is required")
	}

	if authContext.BandID == "" {
		return nil, fmt.Errorf("auth context band id is required")
	}

	if !authContext.Role.IsValid() {
		return nil, fmt.Errorf("auth context role %q is invalid", authContext.Role)
	}

	return context.WithValue(ctx, contextKey{}, authContext), nil
}

func FromContext(ctx context.Context) (Context, bool) {
	authContext, ok := ctx.Value(contextKey{}).(Context)
	return authContext, ok
}
