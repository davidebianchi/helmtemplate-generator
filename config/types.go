package config

// Filter defines top-level resource filtering rules.
// Resources are filtered before any transformation rules are applied.
type Filter struct {
	// Include specifies criteria for resources to include (scoped by kind).
	// Only restricts resources whose kind is mentioned; other kinds pass through.
	Include []Match `yaml:"include,omitempty"`
	// Exclude specifies criteria for resources to exclude.
	// Resources matching any entry are excluded from processing.
	Exclude []Match `yaml:"exclude,omitempty"`
}

// Config represents the configuration file
type Config struct {
	// Filter determines which resources to include/exclude from processing
	Filter *Filter `yaml:"filter,omitempty"`
	// Global rules applied to all resources
	Global *GlobalRules `yaml:"global,omitempty"`
	// Rules for transforming resources
	Rules []Rule `yaml:"rules"`
	// Output configuration for splitting documents into files
	Output *OutputConfig `yaml:"output,omitempty"`
	// ChartFiles defines static files to generate (values.yaml, _helpers.tpl, etc.)
	ChartFiles []ChartFile `yaml:"chartFiles,omitempty"`
}

// ChartFile defines a static chart file to generate from a template
type ChartFile struct {
	// Template is the path to the template file (relative to config file)
	Template string `yaml:"template"`
	// Output is the output path (relative to output directory)
	Output string `yaml:"output"`
}

// OutputConfig defines how to split and organize output files
type OutputConfig struct {
	// SplitByDocument splits multi-document YAML into separate files
	SplitByDocument bool `yaml:"splitByDocument,omitempty"`
	// FileNameTemplate is the template for output filenames (default: "{{.Kind | lower}}-{{.Name}}.yaml")
	FileNameTemplate string `yaml:"fileNameTemplate,omitempty"`
	// DirectoryRules maps resource kinds to subdirectories
	DirectoryRules []DirectoryRule `yaml:"directoryRules,omitempty"`
}

// DirectoryRule maps resource kinds to output subdirectories
type DirectoryRule struct {
	// Match criteria (kind, name pattern)
	Match *Match `yaml:"match,omitempty"`
	// Directory to place matching resources (relative to output path)
	Directory string `yaml:"directory"`
}

// GlobalRules are applied to all resources
type GlobalRules struct {
	// Delete these paths from all resources
	Delete []string `yaml:"delete,omitempty"`
}

// Rule defines a transformation rule
type Rule struct {
	// Match criteria for selecting resources (optional - if empty, applies to all)
	Match *Match `yaml:"match,omitempty"`
	// Path to the field to modify (JSONPath-like syntax)
	Path string `yaml:"path,omitempty"`
	// Value to set at the path (can include Helm template syntax)
	Value string `yaml:"value,omitempty"`
	// Action to perform: set (default), delete, inject
	Action string `yaml:"action,omitempty"`
	// Changes to apply (alternative to single path/value)
	Changes []Change `yaml:"changes,omitempty"`
	// Wrap the entire matched resource
	Wrap *Wrap `yaml:"wrap,omitempty"`
	// ReplaceWith replaces the field at path with raw content
	ReplaceWith string `yaml:"replaceWith,omitempty"`
	// InjectRaw injects raw content at the path
	InjectRaw *InjectRaw `yaml:"injectRaw,omitempty"`
}

// Match criteria for selecting resources
type Match struct {
	// Kinds of Kubernetes resources to match (e.g., [Deployment, Service])
	Kinds []string `yaml:"kinds,omitempty"`
	// Names of the resources to match (supports * wildcard, matches if ANY pattern matches)
	Names []string `yaml:"names,omitempty"`
	// Labels to match (all must match)
	Labels map[string]string `yaml:"labels,omitempty"`
}

// Change represents a single field change
type Change struct {
	// Path to the field (JSONPath-like syntax)
	Path string `yaml:"path"`
	// Value to set (can include Helm template syntax)
	Value string `yaml:"value,omitempty"`
	// Action: set (default), delete
	Action string `yaml:"action,omitempty"`
	// ReplaceWith replaces the field with raw content
	ReplaceWith string `yaml:"replaceWith,omitempty"`
	// WrapValue wraps the field value with conditions
	WrapValue *WrapValue `yaml:"wrapValue,omitempty"`
}

// Wrap wraps a resource or field with before/after content
type Wrap struct {
	Before string `yaml:"before"`
	After  string `yaml:"after"`
}

// WrapValue wraps a field value with conditional template
type WrapValue struct {
	Before   string `yaml:"before"`
	Template string `yaml:"template"`
	After    string `yaml:"after"`
}

// InjectRaw injects raw content at a specific position
type InjectRaw struct {
	// Position: before, after, replace
	Position string `yaml:"position"`
	// Content to inject (can be multi-line Helm template)
	Content string `yaml:"content"`
}
