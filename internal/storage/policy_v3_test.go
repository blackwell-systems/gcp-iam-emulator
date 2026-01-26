package storage

import (
	"strings"
	"testing"

	expr "google.golang.org/genproto/googleapis/type/expr"
	iampb "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // Using standard genproto package
)

func TestPolicyEtag(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
		Version: 1,
		Bindings: []*iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			{
				Role:    "roles/owner",
				Members: []string{"user:admin@example.com"},
			},
		},
	}

	result, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	if len(result.Etag) == 0 {
		t.Error("Expected etag to be generated")
	}

	etag1 := result.Etag

	policy2 := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
		Version: 1,
		Bindings: []*iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			{
				Role:    "roles/viewer",
				Members: []string{"user:dev@example.com"},
			},
		},
	}

	result2, err := s.SetIamPolicy("projects/test", policy2)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	etag2 := result2.Etag

	if string(etag1) == string(etag2) {
		t.Error("Expected different etags for different policies")
	}
}

func TestPolicyVersion(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
		Bindings: []*iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			{
				Role:    "roles/owner",
				Members: []string{"user:admin@example.com"},
			},
		},
	}

	result, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	if result.Version != 1 {
		t.Errorf("Expected version 1 for policy without conditions, got %d", result.Version)
	}
}

func TestPolicyVersion3_WithConditions(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
		Version: 3,
		Bindings: []*iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			{
				Role:    "roles/secretmanager.secretAccessor",
				Members: []string{"serviceAccount:ci@test.iam.gserviceaccount.com"},
				Condition: &expr.Expr{
					Expression: `resource.name.startsWith("projects/test/secrets/prod-")`,
					Title:      "Production secrets only",
				},
			},
		},
	}

	result, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	if result.Version != 3 {
		t.Errorf("Expected version 3 for policy with conditions, got %d", result.Version)
	}

	if result.Bindings[0].Condition == nil {
		t.Error("Expected condition to be preserved")
	}

	if result.Bindings[0].Condition.Expression != `resource.name.startsWith("projects/test/secrets/prod-")` {
		t.Errorf("Expected condition expression to be preserved, got %s", result.Bindings[0].Condition.Expression)
	}
}

func TestPolicyVersion3_EmptyCondition(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
		Version: 3,
		Bindings: []*iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			{
				Role:    "roles/owner",
				Members: []string{"user:admin@example.com"},
				Condition: &expr.Expr{
					Expression: "",
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err == nil {
		t.Error("Expected error for empty condition expression when version is 3")
	}

	if !strings.Contains(err.Error(), "condition expression cannot be empty") {
		t.Errorf("Expected error about empty condition, got: %v", err)
	}
}

func TestConditionalBinding_StartsWith(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
		Version: 3,
		Bindings: []*iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			{
				Role:    "roles/secretmanager.secretAccessor",
				Members: []string{"serviceAccount:ci@test.iam.gserviceaccount.com"},
				Condition: &expr.Expr{
					Expression: `resource.name.startsWith("projects/test/secrets/prod-")`,
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions(
		"projects/test/secrets/prod-api-key",
		"serviceAccount:ci@test.iam.gserviceaccount.com",
		[]string{"secretmanager.versions.access"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected permission allowed (matches condition), got %d", len(allowed))
	}

	denied, err := s.TestIamPermissions(
		"projects/test/secrets/staging-api-key",
		"serviceAccount:ci@test.iam.gserviceaccount.com",
		[]string{"secretmanager.versions.access"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(denied) != 0 {
		t.Errorf("Expected permission denied (condition fails), got %d allowed", len(denied))
	}
}

func TestConditionalBinding_ResourceType(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
		Version: 3,
		Bindings: []*iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			{
				Role:    "roles/viewer",
				Members: []string{"user:dev@example.com"},
				Condition: &expr.Expr{
					Expression: `resource.type == "SECRET"`,
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions(
		"projects/test/secrets/api-key",
		"user:dev@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected permission allowed for SECRET type, got %d", len(allowed))
	}

	denied, err := s.TestIamPermissions(
		"projects/test/locations/global/keyRings/ring/cryptoKeys/key",
		"user:dev@example.com",
		[]string{"cloudkms.cryptoKeys.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(denied) != 0 {
		t.Errorf("Expected permission denied for CRYPTO_KEY type (condition requires SECRET), got %d allowed", len(denied))
	}
}
