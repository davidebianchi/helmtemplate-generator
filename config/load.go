package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

var validRuleActions = map[string]bool{
	"":       true,
	"set":    true,
	"delete": true,
	"inject": true,
}

var validChangeActions = map[string]bool{
	"":       true,
	"set":    true,
	"delete": true,
}

func (cfg *Config) validate() error {
	for i, rule := range cfg.Rules {
		if rule.Path == "" && len(rule.Changes) == 0 && rule.Wrap == nil {
			return fmt.Errorf("rule %d: must specify at least one of path, changes, or wrap", i)
		}
		if !validRuleActions[rule.Action] {
			return fmt.Errorf("rule %d: unknown action %q (valid: set, delete, inject)", i, rule.Action)
		}
		for j, change := range rule.Changes {
			if change.Path == "" {
				return fmt.Errorf("rule %d, change %d: path is required", i, j)
			}
			if !validChangeActions[change.Action] {
				return fmt.Errorf("rule %d, change %d: unknown action %q (valid: set, delete)", i, j, change.Action)
			}
		}
	}
	return nil
}
