package storage

import (
	"testing"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func TestGroups_BasicMembership(t *testing.T) {
	s := NewStorage()

	groups := map[string][]string{
		"developers": {
			"user:alice@example.com",
			"user:bob@example.com",
		},
	}
	s.LoadGroups(groups)

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/viewer",
				Members: []string{
					"group:developers",
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
		"user:alice@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected permission allowed (alice is in developers group), got %d", len(allowed))
	}

	denied, err := s.TestIamPermissions(
		"projects/test",
		"user:charlie@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(denied) != 0 {
		t.Errorf("Expected permission denied (charlie not in developers group), got %d allowed", len(denied))
	}
}

func TestGroups_NestedGroups(t *testing.T) {
	s := NewStorage()

	groups := map[string][]string{
		"engineers": {
			"user:alice@example.com",
			"group:contractors",
		},
		"contractors": {
			"user:bob@example.com",
			"user:charlie@example.com",
		},
	}
	s.LoadGroups(groups)

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/viewer",
				Members: []string{
					"group:engineers",
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
		"user:alice@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowed) != 1 {
		t.Errorf("Expected permission allowed (alice directly in engineers), got %d", len(allowed))
	}

	allowedNested, err := s.TestIamPermissions(
		"projects/test",
		"user:bob@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowedNested) != 1 {
		t.Errorf("Expected permission allowed (bob in contractors, which is in engineers), got %d", len(allowedNested))
	}
}

func TestGroups_MultipleGroups(t *testing.T) {
	s := NewStorage()

	groups := map[string][]string{
		"developers": {
			"user:alice@example.com",
		},
		"operators": {
			"user:bob@example.com",
		},
	}
	s.LoadGroups(groups)

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role: "roles/viewer",
				Members: []string{
					"group:developers",
					"group:operators",
				},
			},
		},
	}

	_, err := s.SetIamPolicy("projects/test", policy)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	allowedAlice, err := s.TestIamPermissions(
		"projects/test",
		"user:alice@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowedAlice) != 1 {
		t.Errorf("Expected permission allowed for alice, got %d", len(allowedAlice))
	}

	allowedBob, err := s.TestIamPermissions(
		"projects/test",
		"user:bob@example.com",
		[]string{"secretmanager.secrets.get"},
		false,
	)

	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(allowedBob) != 1 {
		t.Errorf("Expected permission allowed for bob, got %d", len(allowedBob))
	}
}
