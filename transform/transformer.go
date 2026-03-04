package transform

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/davidebianchi/helmtemplate-generator/config"
	"gopkg.in/yaml.v3"
)

const actionSet = "set"

// Transformer applies configuration rules to YAML documents
type Transformer struct {
	config *config.Config
}

// New creates a new transformer with the given config
func New(cfg *config.Config) *Transformer {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &Transformer{config: cfg}
}

// TransformedDocument represents a transformed document with its metadata
type TransformedDocument struct {
	Kind    string
	Name    string
	Content string
}

// TransformDocuments processes all documents and returns them individually with metadata
func (t *Transformer) TransformDocuments(input []byte) ([]TransformedDocument, error) {
	docs, err := ParseDocuments(input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse documents: %w", err)
	}

	// Apply top-level filter before processing
	docs = FilterDocuments(docs, t.config.Filter)

	results := make([]TransformedDocument, 0, len(docs))
	for _, doc := range docs {
		kind := doc.GetKind()
		name := doc.GetName()

		// Skip documents without kind or name
		if kind == "" || name == "" {
			continue
		}

		output, err := t.transformDocument(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to transform document %s/%s: %w", kind, name, err)
		}

		results = append(results, TransformedDocument{
			Kind:    kind,
			Name:    name,
			Content: output,
		})
	}

	return results, nil
}

// Transform processes all documents and returns the transformed output
func (t *Transformer) Transform(input []byte) (string, error) {
	docs, err := ParseDocuments(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse documents: %w", err)
	}

	// Apply top-level filter before processing
	docs = FilterDocuments(docs, t.config.Filter)

	outputs := make([]string, 0, len(docs))
	for _, doc := range docs {
		output, err := t.transformDocument(doc)
		if err != nil {
			return "", fmt.Errorf("failed to transform document: %w", err)
		}
		outputs = append(outputs, output)
	}

	return strings.Join(outputs, "\n---\n"), nil
}

func (t *Transformer) transformDocument(doc *Document) (string, error) {
	// Apply global deletions first
	if t.config.Global != nil {
		for _, path := range t.config.Global.Delete {
			segments, err := ParsePath(path)
			if err != nil {
				return "", fmt.Errorf("invalid global delete path %s: %w", path, err)
			}
			if err := DeleteAtPath(doc.Root, segments); err != nil {
				return "", fmt.Errorf("failed to delete path %s: %w", path, err)
			}
		}
	}

	// Track wraps to apply after serialization
	var docWrap *config.Wrap
	var fieldReplacements []fieldReplacement

	// Apply rules
	for _, rule := range t.config.Rules {
		if !MatchesDocument(doc, rule.Match) {
			continue
		}

		// Document-level wrap
		if rule.Wrap != nil && rule.Path == "" {
			docWrap = rule.Wrap
		}

		// Single path/value change
		if rule.Path != "" {
			if err := t.applyPathChange(doc, &rule, &fieldReplacements); err != nil {
				return "", err
			}
		}

		// Multiple changes
		for _, change := range rule.Changes {
			if err := t.applyChange(doc, &change, &fieldReplacements); err != nil {
				return "", err
			}
		}
	}

	// Serialize the document
	output, err := doc.Serialize()
	if err != nil {
		return "", err
	}

	// Apply field replacements (for replaceWith and complex injections)
	for _, fr := range fieldReplacements {
		output = applyFieldReplacement(output, fr)
	}

	// Apply document wrap
	if docWrap != nil {
		output = docWrap.Before + "\n" + output + "\n" + docWrap.After
	}

	return output, nil
}

type fieldReplacement struct {
	placeholder string
	content     string
	fieldKey    string
	wrapValue   *config.WrapValue
	isAppend    bool
}

func (t *Transformer) applyPathChange(doc *Document, rule *config.Rule, replacements *[]fieldReplacement) error {
	segments, err := ParsePath(rule.Path)
	if err != nil {
		return fmt.Errorf("invalid path %s: %w", rule.Path, err)
	}

	action := rule.Action
	if action == "" {
		action = actionSet
	}

	switch action {
	case "delete":
		return DeleteAtPath(doc.Root, segments)
	case actionSet:
		if rule.Value != "" {
			return SetValueAtPath(doc.Root, segments, rule.Value)
		}
		if rule.ReplaceWith != "" {
			// Use placeholder for complex replacement
			placeholder := fmt.Sprintf("__HELMGEN_REPLACE_%d__", len(*replacements))
			*replacements = append(*replacements, fieldReplacement{
				placeholder: placeholder,
				content:     rule.ReplaceWith,
				fieldKey:    getLastPathKey(segments),
			})
			return SetValueAtPath(doc.Root, segments, placeholder)
		}
		if rule.AppendWith != "" {
			placeholder := fmt.Sprintf("__HELMGEN_APPEND_%d__", len(*replacements))
			*replacements = append(*replacements, fieldReplacement{
				placeholder: placeholder,
				content:     rule.AppendWith,
				fieldKey:    getLastPathKey(segments),
				isAppend:    true,
			})
			return appendPlaceholderToSequence(doc.Root, segments, placeholder)
		}
	case "inject":
		if rule.InjectRaw != nil {
			return t.applyInjectRaw(doc, segments, rule.InjectRaw, replacements)
		}
	default:
		return fmt.Errorf("unknown action %q for path %s (valid actions: set, delete, inject)", action, rule.Path)
	}

	return nil
}

func (t *Transformer) applyChange(doc *Document, change *config.Change, replacements *[]fieldReplacement) error {
	segments, err := ParsePath(change.Path)
	if err != nil {
		return fmt.Errorf("invalid path %s: %w", change.Path, err)
	}

	action := change.Action
	if action == "" {
		action = actionSet
	}

	switch action {
	case "delete":
		return DeleteAtPath(doc.Root, segments)
	case actionSet:
		if change.Value != "" {
			return SetValueAtPath(doc.Root, segments, change.Value)
		}
		if change.ReplaceWith != "" {
			placeholder := fmt.Sprintf("__HELMGEN_REPLACE_%d__", len(*replacements))
			*replacements = append(*replacements, fieldReplacement{
				placeholder: placeholder,
				content:     change.ReplaceWith,
				fieldKey:    getLastPathKey(segments),
			})
			return SetValueAtPath(doc.Root, segments, placeholder)
		}
		if change.AppendWith != "" {
			placeholder := fmt.Sprintf("__HELMGEN_APPEND_%d__", len(*replacements))
			*replacements = append(*replacements, fieldReplacement{
				placeholder: placeholder,
				content:     change.AppendWith,
				fieldKey:    getLastPathKey(segments),
				isAppend:    true,
			})
			return appendPlaceholderToSequence(doc.Root, segments, placeholder)
		}
		if change.WrapValue != nil {
			placeholder := fmt.Sprintf("__HELMGEN_WRAP_%d__", len(*replacements))
			*replacements = append(*replacements, fieldReplacement{
				placeholder: placeholder,
				fieldKey:    getLastPathKey(segments),
				wrapValue:   change.WrapValue,
			})
			return SetValueAtPath(doc.Root, segments, placeholder)
		}
	default:
		return fmt.Errorf("unknown action %q for path %s (valid actions: set, delete)", action, change.Path)
	}

	return nil
}

func (t *Transformer) applyInjectRaw(
	doc *Document, segments []PathSegment, inject *config.InjectRaw,
	replacements *[]fieldReplacement,
) error {
	placeholder := fmt.Sprintf("__HELMGEN_INJECT_%d__", len(*replacements))

	*replacements = append(*replacements, fieldReplacement{
		placeholder: placeholder,
		content:     inject.Content,
		fieldKey:    getLastPathKey(segments),
	})

	switch inject.Position {
	case "replace":
		return SetValueAtPath(doc.Root, segments, placeholder)
	case "before", "after":
		// For before/after, we need special handling in the output
		node, _, _, err := GetNodeAtPath(doc.Root, segments)
		if err != nil {
			return err
		}
		if node != nil {
			// Mark the node for injection
			node.LineComment = placeholder
		}
	}

	return nil
}

// appendPlaceholderToSequence navigates to the SequenceNode at the given path
// and appends a placeholder scalar as the last element.
func appendPlaceholderToSequence(root *yaml.Node, segments []PathSegment, placeholder string) error {
	node, _, _, err := GetNodeAtPath(root, segments)
	if err != nil {
		return fmt.Errorf("failed to navigate to path for appendWith: %w", err)
	}
	if node == nil {
		return fmt.Errorf("node not found at path for appendWith")
	}
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("appendWith requires a sequence (array) node, got kind %d", node.Kind)
	}

	node.Content = append(node.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: placeholder,
		Tag:   "!!str",
	})

	return nil
}

func getLastPathKey(segments []PathSegment) string {
	if len(segments) == 0 {
		return ""
	}
	last := segments[len(segments)-1]
	if last.Key != "" {
		return last.Key
	}
	return fmt.Sprintf("[%d]", last.Index)
}

func applyFieldReplacement(output string, fr fieldReplacement) string {
	if fr.wrapValue != nil {
		// Handle wrap value - find the line with placeholder and replace
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			if strings.Contains(line, fr.placeholder) {
				indent := getIndent(line)
				// Build the wrapped content
				var wrapped strings.Builder
				wrapped.WriteString(fr.wrapValue.Before + "\n")
				wrapped.WriteString(indent + fr.fieldKey + ": " + fr.wrapValue.Template + "\n")
				wrapped.WriteString(indent + strings.TrimPrefix(fr.wrapValue.After, "\n"))
				lines[i] = wrapped.String()
				break
			}
		}
		return strings.Join(lines, "\n")
	}

	if fr.content != "" {
		// Find the line with placeholder and replace entire field
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			if strings.Contains(line, fr.placeholder) {
				indent := getIndent(line)
				// Indent the replacement content
				contentLines := strings.Split(strings.TrimSuffix(fr.content, "\n"), "\n")
				var indented []string
				for j, cl := range contentLines {
					if j == 0 {
						// First line gets the field's indent (key is removed)
						indented = append(indented, indent+strings.TrimSpace(cl))
					} else {
						indented = append(indented, indent+cl)
					}
				}
				lines[i] = strings.Join(indented, "\n")
				break
			}
		}
		return strings.Join(lines, "\n")
	}

	return output
}

func getIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// GetOutputDirectory returns the output directory for a document based on config rules
func (t *Transformer) GetOutputDirectory(doc TransformedDocument) string {
	if t.config.Output == nil {
		return ""
	}

	for _, rule := range t.config.Output.DirectoryRules {
		if matchesDirectoryRule(doc, rule.Match) {
			return rule.Directory
		}
	}

	return ""
}

// GetOutputFileName returns the output filename for a document
func (t *Transformer) GetOutputFileName(doc TransformedDocument) string {
	// Default template: lowercase kind-name.yaml
	return strings.ToLower(doc.Kind) + "-" + doc.Name + ".yaml"
}

// ShouldSplitDocuments returns true if documents should be split into separate files
func (t *Transformer) ShouldSplitDocuments() bool {
	return t.config.Output != nil && t.config.Output.SplitByDocument
}

func matchesDirectoryRule(doc TransformedDocument, match *config.Match) bool {
	if match == nil {
		return true
	}

	// Check kinds (list)
	if len(match.Kinds) > 0 {
		found := false
		for _, k := range match.Kinds {
			if k == doc.Kind || matchWildcard(k, doc.Kind) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check name patterns (matches if ANY pattern matches)
	if len(match.Names) > 0 {
		nameMatched := false
		for _, pattern := range match.Names {
			if matchWildcard(pattern, doc.Name) {
				nameMatched = true
				break
			}
		}
		if !nameMatched {
			return false
		}
	}

	return true
}

func matchWildcard(pattern, value string) bool {
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		return false
	}
	return matched
}
