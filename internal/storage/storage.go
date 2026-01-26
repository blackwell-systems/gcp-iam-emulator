package storage

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

type Storage struct {
	mu               sync.RWMutex
	projects         map[string]*Project
	serviceAccounts  map[string]*ServiceAccount
	policies         map[string]*iampb.Policy
}

type Project struct {
	Name       string
	CreateTime time.Time
}

type ServiceAccount struct {
	Name        string
	Email       string
	ProjectID   string
	DisplayName string
	Description string
	CreateTime  time.Time
	Keys        map[string]*ServiceAccountKey
	NextKeyID   int64
}

type ServiceAccountKey struct {
	Name       string
	PrivateKey []byte
	PublicKey  []byte
	CreateTime time.Time
	KeyType    string
}

func NewStorage() *Storage {
	return &Storage{
		projects:        make(map[string]*Project),
		serviceAccounts: make(map[string]*ServiceAccount),
		policies:        make(map[string]*iampb.Policy),
	}
}

func (s *Storage) CreateProject(projectID string) (*Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := fmt.Sprintf("projects/%s", projectID)
	if _, exists := s.projects[name]; exists {
		return nil, fmt.Errorf("project already exists: %s", name)
	}

	project := &Project{
		Name:       name,
		CreateTime: time.Now(),
	}

	s.projects[name] = project
	return project, nil
}

func (s *Storage) GetProject(name string) (*Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, exists := s.projects[name]
	if !exists {
		return nil, fmt.Errorf("project not found: %s", name)
	}

	return project, nil
}

func (s *Storage) SetIamPolicy(resource string, policy *iampb.Policy) (*iampb.Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.policies[resource] = policy
	return policy, nil
}

func (s *Storage) LoadPolicies(policies map[string]*iampb.Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for resource, policy := range policies {
		s.policies[resource] = policy
	}
}

func (s *Storage) GetIamPolicy(resource string) (*iampb.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, exists := s.policies[resource]
	if !exists {
		return &iampb.Policy{
			Bindings: []*iampb.Binding{},
			Version:  1,
		}, nil
	}

	return policy, nil
}

func (s *Storage) TestIamPermissions(resource string, principal string, permissions []string, trace bool) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy := s.resolvePolicy(resource)
	if policy == nil {
		if trace {
			slog.Info("authz decision", "decision", "DENY", "resource", resource, "principal", principal, "reason", "no policy found")
		}
		return []string{}, nil
	}

	allowed := []string{}
	for _, perm := range permissions {
		decision, reason := s.hasPermission(policy, principal, perm)
		if decision {
			allowed = append(allowed, perm)
			if trace {
				slog.Info("authz decision", "decision", "ALLOW", "resource", resource, "principal", principal, "permission", perm, "reason", reason)
			}
		} else {
			if trace {
				slog.Info("authz decision", "decision", "DENY", "resource", resource, "principal", principal, "permission", perm, "reason", reason)
			}
		}
	}

	return allowed, nil
}

func (s *Storage) resolvePolicy(resource string) *iampb.Policy {
	if policy, exists := s.policies[resource]; exists {
		return policy
	}

	parts := strings.Split(resource, "/")
	for len(parts) > 2 {
		parts = parts[:len(parts)-2]
		parentResource := strings.Join(parts, "/")
		if policy, exists := s.policies[parentResource]; exists {
			return policy
		}
	}

	return nil
}

func (s *Storage) hasPermission(policy *iampb.Policy, principal string, permission string) (bool, string) {
	rolePerms := map[string][]string{
		"roles/owner": {
			"secretmanager.secrets.get",
			"secretmanager.secrets.create",
			"secretmanager.secrets.delete",
			"secretmanager.versions.access",
			"cloudkms.cryptoKeys.encrypt",
			"cloudkms.cryptoKeys.decrypt",
			"cloudkms.cryptoKeyVersions.create",
		},
		"roles/editor": {
			"secretmanager.secrets.get",
			"secretmanager.secrets.create",
			"secretmanager.versions.access",
			"cloudkms.cryptoKeys.encrypt",
			"cloudkms.cryptoKeys.decrypt",
		},
		"roles/viewer": {
			"secretmanager.secrets.get",
			"cloudkms.cryptoKeys.get",
		},
		"roles/secretmanager.admin": {
			"secretmanager.secrets.get",
			"secretmanager.secrets.create",
			"secretmanager.secrets.delete",
			"secretmanager.versions.access",
		},
		"roles/secretmanager.secretAccessor": {
			"secretmanager.versions.access",
		},
		"roles/cloudkms.admin": {
			"cloudkms.cryptoKeys.encrypt",
			"cloudkms.cryptoKeys.decrypt",
			"cloudkms.cryptoKeyVersions.create",
		},
		"roles/cloudkms.cryptoKeyEncrypterDecrypter": {
			"cloudkms.cryptoKeys.encrypt",
			"cloudkms.cryptoKeys.decrypt",
		},
	}

	if principal == "" {
		for _, binding := range policy.Bindings {
			perms, ok := rolePerms[binding.Role]
			if !ok {
				continue
			}

			for _, p := range perms {
				if p == permission {
					return true, fmt.Sprintf("matched role=%s (no principal check)", binding.Role)
				}
			}
		}
		return false, "no role grants permission (no principal provided)"
	}

	for _, binding := range policy.Bindings {
		perms, ok := rolePerms[binding.Role]
		if !ok {
			continue
		}

		hasPermission := false
		for _, p := range perms {
			if p == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			continue
		}

		for _, member := range binding.Members {
			if s.principalMatches(principal, member) {
				return true, fmt.Sprintf("matched binding: role=%s member=%s", binding.Role, member)
			}
		}
	}

	return false, "no matching binding found for principal"
}

func (s *Storage) principalMatches(principal, member string) bool {
	if principal == member {
		return true
	}

	if member == "allUsers" || member == "allAuthenticatedUsers" {
		return true
	}

	return false
}

func (s *Storage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects = make(map[string]*Project)
	s.serviceAccounts = make(map[string]*ServiceAccount)
	s.policies = make(map[string]*iampb.Policy)
}
