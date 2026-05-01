package permissions

import "fmt"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

type RolePolicy struct {
	CanReadInAlpha  bool
	CanWriteInAlpha bool
}

var rolePolicies = map[Role]RolePolicy{
	RoleOwner: {
		CanReadInAlpha:  true,
		CanWriteInAlpha: true,
	},
	RoleAdmin: {
		CanReadInAlpha:  true,
		CanWriteInAlpha: false,
	},
	RoleMember: {
		CanReadInAlpha:  true,
		CanWriteInAlpha: false,
	},
	RoleViewer: {
		CanReadInAlpha:  true,
		CanWriteInAlpha: false,
	},
}

func ParseRole(value string) (Role, error) {
	role := Role(value)
	if !role.IsValid() {
		return "", fmt.Errorf("invalid role %q", value)
	}

	return role, nil
}

func (role Role) IsValid() bool {
	_, ok := rolePolicies[role]
	return ok
}

func PolicyForRole(role Role) (RolePolicy, error) {
	policy, ok := rolePolicies[role]
	if !ok {
		return RolePolicy{}, fmt.Errorf("invalid role %q", role)
	}

	return policy, nil
}

func CanReadInAlpha(role Role) bool {
	policy, err := PolicyForRole(role)
	if err != nil {
		return false
	}

	return policy.CanReadInAlpha
}

func CanWriteInAlpha(role Role) bool {
	policy, err := PolicyForRole(role)
	if err != nil {
		return false
	}

	return policy.CanWriteInAlpha
}

func RequireAlphaWrite(role Role) error {
	if !role.IsValid() {
		return fmt.Errorf("cannot authorize invalid role %q", role)
	}

	if !CanWriteInAlpha(role) {
		return fmt.Errorf("alpha write access requires owner role, got %q", role)
	}

	return nil
}
