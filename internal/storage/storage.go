package storage

import (
	"fmt"
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

func (s *Storage) TestIamPermissions(resource string, permissions []string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, exists := s.policies[resource]
	if !exists {
		return []string{}, nil
	}

	allowed := []string{}
	for _, perm := range permissions {
		if s.hasPermission(policy, perm) {
			allowed = append(allowed, perm)
		}
	}

	return allowed, nil
}

func (s *Storage) hasPermission(policy *iampb.Policy, permission string) bool {
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

	for _, binding := range policy.Bindings {
		perms, ok := rolePerms[binding.Role]
		if !ok {
			continue
		}

		for _, p := range perms {
			if p == permission {
				return true
			}
		}
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
