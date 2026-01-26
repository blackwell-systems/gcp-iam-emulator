package storage

import (
	"testing"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func TestCustomRoles_BasicUsage(t *testing.T) {
	s := NewStorage()

	customRoles := map[string][]string{
		"roles/custom.dataReader": {
			"bigquery.datasets.get",
			"bigquery.tables.list",
			"bigquery.tables.getData",
		},
	}
	s.LoadCustomRoles(customRoles)

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/custom.dataReader",
				Members: []string{
					"user:analyst@example.com",
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
		"user:analyst@example.com",
		[]string{"bigquery.datasets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected permission allowed, got %d", len(allowed))
	}
}

func TestCustomRoles_OverrideBuiltIn(t *testing.T) {
	s := NewStorage()

	customRoles := map[string][]string{
		"roles/viewer": {
			"custom.permission.read",
		},
	}
	s.LoadCustomRoles(customRoles)

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
		[]string{"custom.permission.read"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected custom permission allowed, got %d", len(allowed))
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
		t.Errorf("Expected built-in permission denied (overridden), got %d allowed", len(denied))
	}
}

func TestWildcardRole_SecretManager(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/secretmanager.customRole",
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
		t.Errorf("Expected wildcard permission allowed, got %d", len(allowed))
	}
}

func TestWildcardRole_KMS(t *testing.T) {
	s := NewStorage()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/cloudkms.encryptOnly",
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
		[]string{"cloudkms.cryptoKeys.encrypt"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected wildcard permission allowed, got %d", len(allowed))
	}
}

func TestWildcardRole_NoMatch(t *testing.T) {
	s := NewStorage()

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
		t.Errorf("Expected permission denied (no wildcard match), got %d allowed", len(denied))
	}
}

func TestCustomRoles_MultiplePermissions(t *testing.T) {
	s := NewStorage()

	customRoles := map[string][]string{
		"roles/custom.fullAccess": {
			"service1.resource.read",
			"service1.resource.write",
			"service2.resource.execute",
		},
	}
	s.LoadCustomRoles(customRoles)

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/custom.fullAccess",
				Members: []string{
					"user:admin@example.com",
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
		"user:admin@example.com",
		[]string{
			"service1.resource.read",
			"service1.resource.write",
			"service2.resource.execute",
			"service3.resource.read",
		},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 3 {
		t.Errorf("Expected 3 permissions allowed, got %d", len(allowed))
	}
}
