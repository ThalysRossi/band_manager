package permissions

import "testing"

func TestCanWriteInAlphaAllowsOnlyOwner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{name: "owner", role: RoleOwner, expected: true},
		{name: "admin", role: RoleAdmin, expected: false},
		{name: "member", role: RoleMember, expected: false},
		{name: "viewer", role: RoleViewer, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := CanWriteInAlpha(test.role)
			if actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestCanReadInAlphaAllowsAllKnownRoles(t *testing.T) {
	t.Parallel()

	roles := []Role{
		RoleOwner,
		RoleAdmin,
		RoleMember,
		RoleViewer,
	}

	for _, role := range roles {
		if !CanReadInAlpha(role) {
			t.Fatalf("expected role %q to read in alpha", role)
		}
	}
}

func TestPolicyForRoleRejectsInvalidRole(t *testing.T) {
	t.Parallel()

	_, err := PolicyForRole(Role("manager"))
	if err == nil {
		t.Fatal("expected invalid role error")
	}
}

func TestParseRoleRejectsInvalidRole(t *testing.T) {
	t.Parallel()

	_, err := ParseRole("manager")
	if err == nil {
		t.Fatal("expected invalid role error")
	}
}

func TestRequireAlphaWriteRejectsViewer(t *testing.T) {
	t.Parallel()

	err := RequireAlphaWrite(RoleViewer)
	if err == nil {
		t.Fatal("expected viewer write rejection")
	}
}
