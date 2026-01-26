package storage

import (
	"testing"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func TestStrictMode_UnknownRoleDenied(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/custom.unknownRole",
				Members: []string{
					"user:user@example.com",
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	denied, err := s.TestIamPermissions(
		"projects/test",
		"user:user@example.com",
		[]string{"custom.permission.read"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(denied) != 0 {
		t.Errorf("Expected permission denied in strict mode, got %d allowed", len(denied))
	}
}

func TestCompatMode_UnknownRoleAllowed(t *testing.T) {
	s := NewStorage()
	s.SetAllowUnknownRoles(true)

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/custom.unknownRole",
				Members: []string{
					"user:user@example.com",
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions(
		"projects/test",
		"user:user@example.com",
		[]string{"custom.permission.read"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected permission allowed in compat mode (wildcard match), got %d", len(allowed))
	}
}

func TestStrictMode_CustomRoleStillWorks(t *testing.T) {
	s := NewStorage()

	customRoles := map[string][]string{
		"roles/custom.definedRole": {
			"custom.permission.read",
			"custom.permission.write",
		},
	}
	s.LoadCustomRoles(customRoles)

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/custom.definedRole",
				Members: []string{
					"user:user@example.com",
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions(
		"projects/test",
		"user:user@example.com",
		[]string{"custom.permission.read"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected custom role to work in strict mode, got %d", len(allowed))
	}
}

func TestStrictMode_BuiltInRolesStillWork(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/viewer",
				Members: []string{
					"user:user@example.com",
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions(
		"projects/test",
		"user:user@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected built-in role to work in strict mode, got %d", len(allowed))
	}
}

func TestCompatMode_WildcardDoesNotMatchWrongService(t *testing.T) {
	s := NewStorage()
	s.SetAllowUnknownRoles(true)

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/storage.objectViewer",
				Members: []string{
					"user:user@example.com",
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	denied, err := s.TestIamPermissions(
		"projects/test",
		"user:user@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(denied) != 0 {
		t.Errorf("Expected wildcard to NOT match wrong service, got %d allowed", len(denied))
	}
}
