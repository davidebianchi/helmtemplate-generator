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
	if cfg.Filter != nil {
		if len(cfg.Filter.Include) == 0 && len(cfg.Filter.Exclude) == 0 {
			return fmt.Errorf("filter: must specify at least one of include or exclude")
		}
	}

	for i, rule := range cfg.Rules {
		if rule.Path == "" && len(rule.Changes) == 0 && rule.Wrap == nil {
			return fmt.Errorf("rule %d: must specify at least one of path, changes, or wrap", i)
		}
		if !validRuleActions[rule.Action] {
			return fmt.Errorf("rule %d: unknown action %q (valid: set, delete, inject)", i, rule.Action)
		}
		if err := validateRuleSetAction(i, rule); err != nil {
			return err
		}
		if rule.InjectRaw != nil {
			if pos := rule.InjectRaw.Position; pos != "" && pos != "replace" {
				return fmt.Errorf("rule %d: unsupported injectRaw position %q (only \"replace\" is supported)", i, pos)
			}
		}
		for j, change := range rule.Changes {
			if change.Path == "" {
				return fmt.Errorf("rule %d, change %d: path is required", i, j)
			}
			if !validChangeActions[change.Action] {
				return fmt.Errorf("rule %d, change %d: unknown action %q (valid: set, delete)", i, j, change.Action)
			}
			if err := validateChangeSetAction(i, j, change); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRuleSetAction(i int, rule Rule) error {
	action := rule.Action
	if action != "" && action != "set" {
		return nil
	}
	if rule.Path == "" {
		return nil
	}
	if rule.Value == "" && rule.ReplaceWith == "" && rule.AppendWith == "" && rule.InjectRaw == nil {
		return fmt.Errorf("rule %d: set action requires at least one of value, replaceWith, appendWith, or injectRaw", i)
	}
	return nil
}

func validateChangeSetAction(i, j int, change Change) error {
	action := change.Action
	if action != "" && action != "set" {
		return nil
	}
	if change.Value == "" && change.ReplaceWith == "" && change.AppendWith == "" && change.WrapValue == nil {
		return fmt.Errorf(
			"rule %d, change %d: set action requires at least one of value, replaceWith, appendWith, or wrapValue",
			i, j,
		)
	}
	return nil
}
