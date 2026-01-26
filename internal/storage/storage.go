package storage

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	groups           map[string][]string
	customRoles      map[string][]string
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
		groups:          make(map[string][]string),
		customRoles:     make(map[string][]string),
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

	if policy.Version == 0 {
		policy.Version = 1
	}

	if policy.Version == 3 {
		for _, binding := range policy.Bindings {
			if binding.Condition != nil {
				if binding.Condition.Expression == "" {
					return nil, fmt.Errorf("condition expression cannot be empty when version is 3")
				}
			}
		}
	}

	policy.Etag = s.generateEtag(policy)

	s.policies[resource] = policy
	return policy, nil
}

func (s *Storage) generateEtag(policy *iampb.Policy) []byte {
	data, _ := json.Marshal(policy)
	hash := sha256.Sum256(data)
	return []byte(base64.StdEncoding.EncodeToString(hash[:]))
}

func (s *Storage) LoadPolicies(policies map[string]*iampb.Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for resource, policy := range policies {
		if policy.Version == 0 {
			policy.Version = 1
		}
		policy.Etag = s.generateEtag(policy)
		s.policies[resource] = policy
	}
}

func (s *Storage) LoadGroups(groups map[string][]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.groups = groups
}

func (s *Storage) LoadCustomRoles(roles map[string][]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.customRoles = roles
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

	evalCtx := EvalContext{
		ResourceName: resource,
		ResourceType: extractResourceType(resource),
		RequestTime:  time.Now(),
	}

	allowed := []string{}
	for _, perm := range permissions {
		decision, reason := s.hasPermission(policy, principal, perm, evalCtx, trace)
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

func (s *Storage) getRolePermissions(role string, permission string) ([]string, bool) {
	if perms, ok := s.customRoles[role]; ok {
		return perms, true
	}

	builtInRoles := map[string][]string{
		"roles/owner": {
			"secretmanager.secrets.get",
			"secretmanager.secrets.create",
			"secretmanager.secrets.update",
			"secretmanager.secrets.delete",
			"secretmanager.secrets.list",
			"secretmanager.versions.add",
			"secretmanager.versions.get",
			"secretmanager.versions.access",
			"secretmanager.versions.list",
			"secretmanager.versions.enable",
			"secretmanager.versions.disable",
			"secretmanager.versions.destroy",
			"cloudkms.keyRings.create",
			"cloudkms.keyRings.get",
			"cloudkms.keyRings.list",
			"cloudkms.cryptoKeys.create",
			"cloudkms.cryptoKeys.get",
			"cloudkms.cryptoKeys.list",
			"cloudkms.cryptoKeys.update",
			"cloudkms.cryptoKeys.encrypt",
			"cloudkms.cryptoKeys.decrypt",
			"cloudkms.cryptoKeyVersions.create",
			"cloudkms.cryptoKeyVersions.get",
			"cloudkms.cryptoKeyVersions.list",
			"cloudkms.cryptoKeyVersions.update",
			"cloudkms.cryptoKeyVersions.destroy",
		},
		"roles/editor": {
			"secretmanager.secrets.get",
			"secretmanager.secrets.create",
			"secretmanager.secrets.update",
			"secretmanager.secrets.list",
			"secretmanager.versions.add",
			"secretmanager.versions.get",
			"secretmanager.versions.access",
			"secretmanager.versions.list",
			"secretmanager.versions.enable",
			"secretmanager.versions.disable",
			"cloudkms.keyRings.get",
			"cloudkms.keyRings.list",
			"cloudkms.cryptoKeys.create",
			"cloudkms.cryptoKeys.get",
			"cloudkms.cryptoKeys.list",
			"cloudkms.cryptoKeys.update",
			"cloudkms.cryptoKeys.encrypt",
			"cloudkms.cryptoKeys.decrypt",
			"cloudkms.cryptoKeyVersions.create",
			"cloudkms.cryptoKeyVersions.get",
			"cloudkms.cryptoKeyVersions.list",
			"cloudkms.cryptoKeyVersions.update",
		},
		"roles/viewer": {
			"secretmanager.secrets.get",
			"secretmanager.secrets.list",
			"secretmanager.versions.get",
			"secretmanager.versions.list",
			"cloudkms.keyRings.get",
			"cloudkms.keyRings.list",
			"cloudkms.cryptoKeys.get",
			"cloudkms.cryptoKeys.list",
			"cloudkms.cryptoKeyVersions.get",
			"cloudkms.cryptoKeyVersions.list",
		},
		"roles/secretmanager.admin": {
			"secretmanager.secrets.get",
			"secretmanager.secrets.create",
			"secretmanager.secrets.update",
			"secretmanager.secrets.delete",
			"secretmanager.secrets.list",
			"secretmanager.versions.add",
			"secretmanager.versions.get",
			"secretmanager.versions.access",
			"secretmanager.versions.list",
			"secretmanager.versions.enable",
			"secretmanager.versions.disable",
			"secretmanager.versions.destroy",
		},
		"roles/secretmanager.secretAccessor": {
			"secretmanager.versions.access",
		},
		"roles/secretmanager.secretVersionManager": {
			"secretmanager.versions.add",
			"secretmanager.versions.get",
			"secretmanager.versions.list",
			"secretmanager.versions.enable",
			"secretmanager.versions.disable",
			"secretmanager.versions.destroy",
		},
		"roles/cloudkms.admin": {
			"cloudkms.keyRings.create",
			"cloudkms.keyRings.get",
			"cloudkms.keyRings.list",
			"cloudkms.cryptoKeys.create",
			"cloudkms.cryptoKeys.get",
			"cloudkms.cryptoKeys.list",
			"cloudkms.cryptoKeys.update",
			"cloudkms.cryptoKeys.encrypt",
			"cloudkms.cryptoKeys.decrypt",
			"cloudkms.cryptoKeyVersions.create",
			"cloudkms.cryptoKeyVersions.get",
			"cloudkms.cryptoKeyVersions.list",
			"cloudkms.cryptoKeyVersions.update",
			"cloudkms.cryptoKeyVersions.destroy",
		},
		"roles/cloudkms.cryptoKeyEncrypterDecrypter": {
			"cloudkms.cryptoKeys.encrypt",
			"cloudkms.cryptoKeys.decrypt",
		},
		"roles/cloudkms.viewer": {
			"cloudkms.keyRings.get",
			"cloudkms.keyRings.list",
			"cloudkms.cryptoKeys.get",
			"cloudkms.cryptoKeys.list",
			"cloudkms.cryptoKeyVersions.get",
			"cloudkms.cryptoKeyVersions.list",
		},
	}

	if perms, ok := builtInRoles[role]; ok {
		return perms, true
	}

	return s.wildcardRolePermissions(role, permission)
}

func (s *Storage) wildcardRolePermissions(role, permission string) ([]string, bool) {
	if !strings.HasPrefix(role, "roles/") {
		return nil, false
	}

	roleName := strings.TrimPrefix(role, "roles/")
	permPrefix := strings.Split(permission, ".")[0]

	if strings.Contains(roleName, permPrefix) {
		return []string{permission}, true
	}

	return nil, false
}

func (s *Storage) hasPermission(policy *iampb.Policy, principal string, permission string, evalCtx EvalContext, trace bool) (bool, string) { //nolint:staticcheck // Using standard genproto package

	if principal == "" {
		for _, binding := range policy.Bindings {
			perms, ok := s.getRolePermissions(binding.Role, permission)
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
		perms, ok := s.getRolePermissions(binding.Role, permission)
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
				if binding.Condition != nil {
					condResult, condReason := evaluateCondition(binding.Condition, evalCtx)
					if trace {
						slog.Info("condition evaluation", "resource", evalCtx.ResourceName, "principal", principal, "condition", binding.Condition.Expression, "result", condResult, "reason", condReason)
					}
					if !condResult {
						return false, fmt.Sprintf("condition failed: %s", condReason)
					}
					return true, fmt.Sprintf("matched binding: role=%s member=%s condition=%s", binding.Role, member, condReason)
				}
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

	if strings.HasPrefix(member, "group:") {
		groupName := strings.TrimPrefix(member, "group:")
		if groupMembers, exists := s.groups[groupName]; exists {
			for _, groupMember := range groupMembers {
				if groupMember == principal {
					return true
				}
				if strings.HasPrefix(groupMember, "group:") {
					nestedGroupName := strings.TrimPrefix(groupMember, "group:")
					if nestedMembers, nestedExists := s.groups[nestedGroupName]; nestedExists {
						for _, nestedMember := range nestedMembers {
							if nestedMember == principal {
								return true
							}
						}
					}
				}
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
	s.groups = make(map[string][]string)
	s.customRoles = make(map[string][]string)
}
