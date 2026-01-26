package config

import (
	"os"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	yamlContent := `
projects:
  test-project:
    bindings:
      - role: roles/owner
        members:
          - user:admin@example.com
      - role: roles/viewer
        members:
          - user:viewer@example.com
    resources:
      secrets/db-password:
        bindings:
          - role: roles/secretmanager.secretAccessor
            members:
              - serviceAccount:app@test-project.iam.gserviceaccount.com
`

	tmpfile, err := os.CreateTemp("", "policy-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if len(cfg.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(cfg.Projects))
	}

	project, exists := cfg.Projects["test-project"]
	if !exists {
		t.Fatal("test-project not found")
	}

	if len(project.Bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(project.Bindings))
	}

	if len(project.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(project.Resources))
	}
}

func TestToPolicies(t *testing.T) {
	cfg := &Config{
		Projects: map[string]ProjectConfig{
			"test-project": {
				Bindings: []BindingConfig{
					{
						Role:    "roles/owner",
						Members: []string{"user:admin@example.com"},
					},
				},
				Resources: map[string]ResourceConfig{
					"secrets/db-password": {
						Bindings: []BindingConfig{
							{
								Role:    "roles/secretmanager.secretAccessor",
								Members: []string{"serviceAccount:app@test.iam.gserviceaccount.com"},
							},
						},
					},
				},
			},
		},
	}

	policies := cfg.ToPolicies()

	if len(policies) != 2 {
		t.Errorf("Expected 2 policies, got %d", len(policies))
	}

	projectPolicy, exists := policies["projects/test-project"]
	if !exists {
		t.Fatal("Project policy not found")
	}

	if len(projectPolicy.Bindings) != 1 {
		t.Errorf("Expected 1 binding in project policy, got %d", len(projectPolicy.Bindings))
	}

	if projectPolicy.Bindings[0].Role != "roles/owner" {
		t.Errorf("Expected roles/owner, got %s", projectPolicy.Bindings[0].Role)
	}

	secretPolicy, exists := policies["projects/test-project/secrets/db-password"]
	if !exists {
		t.Fatal("Secret policy not found")
	}

	if len(secretPolicy.Bindings) != 1 {
		t.Errorf("Expected 1 binding in secret policy, got %d", len(secretPolicy.Bindings))
	}

	if secretPolicy.Bindings[0].Role != "roles/secretmanager.secretAccessor" {
		t.Errorf("Expected roles/secretmanager.secretAccessor, got %s", secretPolicy.Bindings[0].Role)
	}
}
