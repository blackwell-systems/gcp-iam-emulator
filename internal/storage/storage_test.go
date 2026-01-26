package storage

import (
	"testing"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func TestSetIamPolicy(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/owner",
				Members: []string{"user:alice@example.com"},
			},
		},
	}

	result, err := s.SetIamPolicy("projects/test/secrets/secret1", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	if len(result.Bindings) != 1 {
		t.Errorf("Expected 1 binding, got %d", len(result.Bindings))
	}
}

func TestGetIamPolicy(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/viewer",
				Members: []string{"user:bob@example.com"},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test/secrets/secret1", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	result, err := s.GetIamPolicy("projects/test/secrets/secret1")
	if err != nil {
		t.Fatalf("GetIamPolicy failed: %v", err)
	}

	if len(result.Bindings) != 1 {
		t.Errorf("Expected 1 binding, got %d", len(result.Bindings))
	}

	if result.Bindings[0].Role != "roles/viewer" {
		t.Errorf("Expected role 'roles/viewer', got '%s'", result.Bindings[0].Role)
	}
}

func TestGetIamPolicyEmpty(t *testing.T) {
	s := NewStorage()

	policy, err := s.GetIamPolicy("projects/test/secrets/nonexistent")
	if err != nil {
		t.Fatalf("GetIamPolicy failed: %v", err)
	}

	if len(policy.Bindings) != 0 {
		t.Errorf("Expected empty bindings for nonexistent resource, got %d", len(policy.Bindings))
	}

	if policy.Version != 1 {
		t.Errorf("Expected version 1, got %d", policy.Version)
	}
}

func TestTestIamPermissions_SecretAccessor(t *testing.T) {
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

	allowed, err := s.TestIamPermissions("projects/test/secrets/secret1", []string{
		"secretmanager.versions.access",
		"secretmanager.secrets.delete",
	})
	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected 1 allowed permission, got %d", len(allowed))
	}

	if allowed[0] != "secretmanager.versions.access" {
		t.Errorf("Expected 'secretmanager.versions.access', got '%s'", allowed[0])
	}
}

func TestTestIamPermissions_Owner(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/owner",
				Members: []string{"user:admin@example.com"},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test/secrets/secret1", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions("projects/test/secrets/secret1", []string{
		"secretmanager.versions.access",
		"secretmanager.secrets.delete",
		"cloudkms.cryptoKeys.encrypt",
	})
	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 3 {
		t.Errorf("Expected 3 allowed permissions (owner has all), got %d: %v", len(allowed), allowed)
	}
}

func TestTestIamPermissions_NoPolicy(t *testing.T) {
	s := NewStorage()

	allowed, err := s.TestIamPermissions("projects/test/secrets/secret1", []string{
		"secretmanager.versions.access",
	})
	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 0 {
		t.Errorf("Expected 0 allowed permissions (no policy set), got %d", len(allowed))
	}
}

func TestTestIamPermissions_KMS(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/cloudkms.cryptoKeyEncrypterDecrypter",
				Members: []string{"serviceAccount:app@test.iam.gserviceaccount.com"},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test/locations/global/keyRings/ring1/cryptoKeys/key1", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowed, err := s.TestIamPermissions("projects/test/locations/global/keyRings/ring1/cryptoKeys/key1", []string{
		"cloudkms.cryptoKeys.encrypt",
		"cloudkms.cryptoKeys.decrypt",
		"cloudkms.cryptoKeyVersions.create",
	})
	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 2 {
		t.Errorf("Expected 2 allowed permissions (encrypt, decrypt), got %d: %v", len(allowed), allowed)
	}
}
