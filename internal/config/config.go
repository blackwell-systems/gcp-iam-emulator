package config

import (
	"fmt"
	"os"

	iampb "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // Using standard genproto package
	"gopkg.in/yaml.v3"
)

type Config struct {
	Projects map[string]ProjectConfig `yaml:"projects"`
}

type ProjectConfig struct {
	Bindings  []BindingConfig            `yaml:"bindings"`
	Resources map[string]ResourceConfig `yaml:"resources,omitempty"`
}

type ResourceConfig struct {
	Bindings []BindingConfig `yaml:"bindings"`
}

type BindingConfig struct {
	Role    string   `yaml:"role"`
	Members []string `yaml:"members"`
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

		if len(projectCfg.Bindings) > 0 {
			policies[projectResource] = &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
				Version:  1,
				Bindings: bindingsToProto(projectCfg.Bindings),
			}
		}

		for resourcePath, resourceCfg := range projectCfg.Resources {
			fullResource := fmt.Sprintf("%s/%s", projectResource, resourcePath)
			policies[fullResource] = &iampb.Policy{ //nolint:staticcheck // Using standard genproto package
				Version:  1,
				Bindings: bindingsToProto(resourceCfg.Bindings),
			}
		}
	}

	return policies
}

func bindingsToProto(bindings []BindingConfig) []*iampb.Binding { //nolint:staticcheck // Using standard genproto package
	result := make([]*iampb.Binding, len(bindings)) //nolint:staticcheck // Using standard genproto package
	for i, b := range bindings {
		result[i] = &iampb.Binding{ //nolint:staticcheck // Using standard genproto package
			Role:    b.Role,
			Members: b.Members,
		}
	}
	return result
}
