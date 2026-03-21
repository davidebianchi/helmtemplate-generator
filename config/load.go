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

var validChangeActions = map[string]bool{
	"":       true,
	"set":    true,
	"delete": true,
	"inject": true,
}

func (cfg *Config) validate() error {
	if cfg.Filter != nil {
		if len(cfg.Filter.Include) == 0 && len(cfg.Filter.Exclude) == 0 {
			return fmt.Errorf("filter: must specify at least one of include or exclude")
		}
	}

	for i, rule := range cfg.Rules {
		if len(rule.Changes) == 0 && rule.Wrap == nil {
			return fmt.Errorf("rule %d: must specify at least one of changes or wrap", i)
		}
		for j, change := range rule.Changes {
			if change.Path == "" {
				return fmt.Errorf("rule %d, change %d: path is required", i, j)
			}
			if !validChangeActions[change.Action] {
				return fmt.Errorf("rule %d, change %d: unknown action %q (valid: set, delete, inject)", i, j, change.Action)
			}
			if err := validateChangeSetAction(i, j, change); err != nil {
				return err
			}
			if change.InjectRaw != nil {
				if pos := change.InjectRaw.Position; pos != "" && pos != "replace" {
					return fmt.Errorf("rule %d, change %d: unsupported injectRaw position %q (only \"replace\" is supported)", i, j, pos)
				}
			}
		}
	}
	return nil
}

func validateChangeSetAction(i, j int, change Change) error {
	action := change.Action
	if action != "" && action != "set" {
		return nil
	}
	if change.Value == "" && change.ReplaceWith == "" && change.AppendWith == "" && change.WrapValue == nil && change.InjectRaw == nil {
		return fmt.Errorf(
			"rule %d, change %d: set action requires at least one of value, replaceWith, appendWith, wrapValue, or injectRaw",
			i, j,
		)
	}
	return nil
}
