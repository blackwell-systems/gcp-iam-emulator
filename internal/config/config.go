package config

import (
	"fmt"
	"os"

	expr "google.golang.org/genproto/googleapis/type/expr"
	iampb "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // Using standard genproto package
	"gopkg.in/yaml.v3"
)

type Config struct {
	Projects map[string]ProjectConfig `yaml:"projects"`
	Groups   map[string]GroupConfig   `yaml:"groups,omitempty"`
	Roles    map[string]RoleConfig    `yaml:"roles,omitempty"`
}

type GroupConfig struct {
	Members []string `yaml:"members"`
}

type RoleConfig struct {
	Permissions []string `yaml:"permissions"`
}

type ProjectConfig struct {
	Bindings     []BindingConfig            `yaml:"bindings"`
	AuditConfigs []AuditConfigYAML          `yaml:"auditConfigs,omitempty"`
	Resources    map[string]ResourceConfig  `yaml:"resources,omitempty"`
}

type ResourceConfig struct {
	Bindings     []BindingConfig   `yaml:"bindings"`
	AuditConfigs []AuditConfigYAML `yaml:"auditConfigs,omitempty"`
}

type BindingConfig struct {
	Role      string          `yaml:"role"`
	Members   []string        `yaml:"members"`
	Condition *ConditionYAML  `yaml:"condition,omitempty"`
}

type ConditionYAML struct {
	Expression  string `yaml:"expression"`
	Title       string `yaml:"title,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type AuditConfigYAML struct {
	Service         string              `yaml:"service"`
	AuditLogConfigs []AuditLogConfigYAML `yaml:"auditLogConfigs"`
}

type AuditLogConfigYAML struct {
	LogType         string   `yaml:"logType"`
	ExemptedMembers []string `yaml:"exemptedMembers,omitempty"`
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) ToPolicies() map[string]*iampb.Policy { //nolint:staticcheck // Using standard genproto package
	policies := make(map[string]*iampb.Policy) //nolint:staticcheck // Using standard genproto package

	for projectID, projectCfg := range c.Projects {
		projectResource := fmt.Sprintf("projects/%s", projectID)

		if len(projectCfg.Bindings) > 0 || len(projectCfg.AuditConfigs) > 0 {
			policy := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
				Bindings:     bindingsToProto(projectCfg.Bindings),
				AuditConfigs: auditConfigsToProto(projectCfg.AuditConfigs),
			}
			
			policy.Version = determineVersion(policy)
			policies[projectResource] = policy
		}

		for resourcePath, resourceCfg := range projectCfg.Resources {
			fullResource := fmt.Sprintf("%s/%s", projectResource, resourcePath)
			policy := &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
				Bindings:     bindingsToProto(resourceCfg.Bindings),
				AuditConfigs: auditConfigsToProto(resourceCfg.AuditConfigs),
			}
			
			policy.Version = determineVersion(policy)
			policies[fullResource] = policy
		}
	}

	return policies
}

func determineVersion(policy *iampb.Policy) int32 { //nolint:staticcheck // Using standard genproto package
	for _, binding := range policy.Bindings {
		if binding.Condition != nil {
			return 3
		}
	}
	return 1
}

func bindingsToProto(bindings []BindingConfig) []*iampb.Binding { //nolint:staticcheck // Using standard genproto package
	result := make([]*iampb.Binding, len(bindings)) //nolint:staticcheck // Using standard genproto package
	for i, b := range bindings {
		binding := &iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			Role:    b.Role,
			Members: b.Members,
		}
		
		if b.Condition != nil {
			binding.Condition = &expr.Expr{
				Expression:  b.Condition.Expression,
				Title:       b.Condition.Title,
				Description: b.Condition.Description,
			}
		}
		
		result[i] = binding
	}
	return result
}

func auditConfigsToProto(configs []AuditConfigYAML) []*iampb.AuditConfig { //nolint:staticcheck // Using standard genproto package
	if len(configs) == 0 {
		return nil
	}
	
	result := make([]*iampb.AuditConfig, len(configs)) //nolint:staticcheck // Using standard genproto package
	for i, cfg := range configs {
		auditConfig := &iampb.AuditConfig{ //nolint:staticcheck // Using standard genproto package
			Service: cfg.Service,
		}
		
		for _, logCfg := range cfg.AuditLogConfigs {
			auditConfig.AuditLogConfigs = append(auditConfig.AuditLogConfigs, &iampb.AuditLogConfig{ //nolint:staticcheck // Using standard genproto package
				LogType:         iampb.AuditLogConfig_LogType(iampb.AuditLogConfig_LogType_value[logCfg.LogType]),
				ExemptedMembers: logCfg.ExemptedMembers,
			})
		}
		
		result[i] = auditConfig
	}
	return result
}
