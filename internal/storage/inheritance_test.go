package storage

import (
	"testing"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func TestResourceHierarchyInheritance(t *testing.T) {
	s := NewStorage()

	projectPolicy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/viewer",
				Members: []string{"user:dev@example.com"},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test-project", projectPolicy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions(
		"projects/test-project/secrets/db-password",
		"user:dev@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected permission to be inherited from project, got %d permissions", len(allowed))
	}

	if len(allowed) > 0 && allowed[0] != "secretmanager.secrets.get" {
		t.Errorf("Expected secretmanager.secrets.get, got %s", allowed[0])
	}
}

func TestResourceOverridesParent(t *testing.T) {
	s := NewStorage()

	projectPolicy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/viewer",
				Members: []string{"user:dev@example.com"},
			},
		},
	}
	_, err := s.SetIamPolicy("projects/test-project", projectPolicy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	secretPolicy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/secretmanager.secretAccessor",
				Members: []string{"serviceAccount:app@test.iam.gserviceaccount.com"},
			},
		},
	}
	_, err = s.SetIamPolicy("projects/test-project/secrets/db-password", secretPolicy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions(
		"projects/test-project/secrets/db-password",
		"user:dev@example.com",
		[]string{"secretmanager.versions.access"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 0 {
		t.Errorf("Expected resource policy to override parent (deny viewer), got %d allowed", len(allowed))
	}

	allowed, err = s.TestIamPermissions(
		"projects/test-project/secrets/db-password",
		"serviceAccount:app@test.iam.gserviceaccount.com",
		[]string{"secretmanager.versions.access"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected app service account to have access via resource policy, got %d allowed", len(allowed))
	}
}

func TestPrincipalMatching(t *testing.T) {
	s := NewStorage()

	tests := []struct {
		name      string
		member    string
		principal string
		expected  bool
	}{
		{"exact match user", "user:alice@example.com", "user:alice@example.com", true},
		{"exact match sa", "serviceAccount:ci@test.iam.gserviceaccount.com", "serviceAccount:ci@test.iam.gserviceaccount.com", true},
		{"allUsers", "allUsers", "user:anyone@example.com", true},
		{"allAuthenticatedUsers", "allAuthenticatedUsers", "serviceAccount:anyone@test.iam.gserviceaccount.com", true},
		{"no match", "user:alice@example.com", "user:bob@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.principalMatches(tt.principal, tt.member)
			if result != tt.expected {
				t.Errorf("principalMatches(%q, %q) = %v, expected %v", tt.principal, tt.member, result, tt.expected)
			}
		})
	}
}

func TestNoPrincipalBackwardCompatibility(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/secretmanager.secretAccessor",
				Members: []string{"serviceAccount:ci@test.iam.gserviceaccount.com"},
			},
		},
	}
	_, err := s.SetIamPolicy("projects/test/secrets/secret1", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions("projects/test/secrets/secret1", "", []string{
		"secretmanager.versions.access",
	}, false)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected permission allowed without principal check (backward compat), got %d", len(allowed))
	}
}
