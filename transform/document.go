package transform

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document represents a parsed YAML document with its root node
type Document struct {
	Root *yaml.Node
}

// ParseDocuments parses multi-document YAML input into separate documents
func ParseDocuments(input []byte) ([]*Document, error) {
	var docs []*Document
	decoder := yaml.NewDecoder(bytes.NewReader(input))

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML document: %w", err)
		}

		// Skip empty documents
		if node.Kind == 0 || (node.Kind == yaml.DocumentNode && len(node.Content) == 0) {
			continue
		}

		docs = append(docs, &Document{Root: &node})
	}

	return docs, nil
}

// GetKind returns the kind field of a Kubernetes resource
func (d *Document) GetKind() string {
	return d.getStringField("kind")
}

// GetName returns the metadata.name field
func (d *Document) GetName() string {
	metadata := d.getMapField("metadata")
	if metadata == nil {
		return ""
	}
	return getStringFromMap(metadata, "name")
}

// GetLabels returns the metadata.labels map
func (d *Document) GetLabels() map[string]string {
	metadata := d.getMapField("metadata")
	if metadata == nil {
		return nil
	}
	labels := getMapFromMap(metadata, "labels")
	if labels == nil {
		return nil
	}

	result := make(map[string]string)
	for i := 0; i < len(labels.Content); i += 2 {
		if i+1 < len(labels.Content) {
			key := labels.Content[i].Value
			value := labels.Content[i+1].Value
			result[key] = value
		}
	}
	return result
}

// Serialize converts the document back to YAML string
func (d *Document) Serialize() (string, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	// Get the actual content node (unwrap DocumentNode if present)
	node := d.Root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	if err := encoder.Encode(node); err != nil {
		return "", fmt.Errorf("failed to serialize document: %w", err)
	}
	err := encoder.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close encoder: %w", err)
	}

	result := strings.TrimSuffix(buf.String(), "\n")

	// Post-process to remove quotes around Helm template expressions
	result = unquoteHelmTemplates(result)

	return result, nil
}

// unquoteHelmTemplates removes quotes around values containing Helm template syntax
func unquoteHelmTemplates(input string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		// Match: key: '{{ ... }}' or key: "{{ ... }}"
		// Replace with: key: {{ ... }}
		if strings.Contains(line, "{{") {
			// Handle single quotes: 'value with {{ }}'
			line = unquoteLine(line, "'")
			// Handle double quotes: "value with {{ }}"
			line = unquoteLine(line, "\"")
			lines[i] = line
		}
	}
	return strings.Join(lines, "\n")
}

func unquoteLine(line, quote string) string {
	// Find pattern: key: 'value' or key: "value"
	// We need to find the colon, then the quoted value
	colonIdx := strings.Index(line, ": ")
	if colonIdx == -1 {
		return line
	}

	afterColon := line[colonIdx+1:]
	trimmed := strings.TrimSpace(afterColon)

	// Check if it starts and ends with the quote
	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, quote) && strings.HasSuffix(trimmed, quote) {
		// Extract the value without quotes
		unquoted := trimmed[1 : len(trimmed)-1]
		// Rebuild the line preserving leading whitespace after colon
		leadingSpace := afterColon[:len(afterColon)-len(strings.TrimLeft(afterColon, " "))]
		return line[:colonIdx+1] + leadingSpace + unquoted
	}

	return line
}

func (d *Document) getStringField(name string) string {
	node := d.Root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return ""
	}
	return getStringFromMap(node, name)
}

func (d *Document) getMapField(name string) *yaml.Node {
	node := d.Root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	return getMapFromMap(node, name)
}

func getStringFromMap(node *yaml.Node, key string) string {
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Value == key {
			return node.Content[i+1].Value
		}
	}
	return ""
}

func getMapFromMap(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Value == key {
			if node.Content[i+1].Kind == yaml.MappingNode {
				return node.Content[i+1]
			}
		}
	}
	return nil
}
