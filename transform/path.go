package transform

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// PathSegment represents a segment in a YAML path
type PathSegment struct {
	Key   string // For map access
	Index int    // For array access (-1 if not array)
}

// ParsePath parses a JSONPath-like string into segments
// Examples:
//   - .metadata.name -> [{Key: "metadata"}, {Key: "name"}]
//   - .spec.containers[0].image -> [{Key: "spec"}, {Key: "containers"}, {Index: 0}, {Key: "image"}]
func ParsePath(path string) ([]PathSegment, error) {
	if path == "" {
		return nil, nil
	}

	// Remove leading dot
	path = strings.TrimPrefix(path, ".")

	var segments []PathSegment
	// Match either a key or an array index
	re := regexp.MustCompile(`([^.\[\]]+)|\[(\d+)\]`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if match[1] != "" {
			// Key segment
			segments = append(segments, PathSegment{Key: match[1], Index: -1})
		} else if match[2] != "" {
			// Array index segment
			idx, err := strconv.Atoi(match[2])
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", match[2])
			}
			segments = append(segments, PathSegment{Index: idx})
		}
	}

	return segments, nil
}

// GetNodeAtPath traverses the YAML tree and returns the node at the given path
func GetNodeAtPath(root *yaml.Node, segments []PathSegment) (*yaml.Node, *yaml.Node, int, error) {
	if len(segments) == 0 {
		return root, nil, -1, nil
	}

	node := root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	var parent *yaml.Node
	var indexInParent = -1

	for i, seg := range segments {
		parent = node
		isLast := i == len(segments)-1

		if seg.Index >= 0 {
			// Array access
			if node.Kind != yaml.SequenceNode {
				return nil, nil, -1, fmt.Errorf("expected sequence at path segment [%d]", seg.Index)
			}
			if seg.Index >= len(node.Content) {
				return nil, nil, -1, fmt.Errorf("index %d out of bounds", seg.Index)
			}
			indexInParent = seg.Index
			node = node.Content[seg.Index]
		} else {
			// Map access
			if node.Kind != yaml.MappingNode {
				return nil, nil, -1, fmt.Errorf("expected mapping at path segment %s", seg.Key)
			}

			found := false
			for j := 0; j < len(node.Content); j += 2 {
				if j+1 < len(node.Content) && node.Content[j].Value == seg.Key {
					indexInParent = j + 1 // Index of the value node
					node = node.Content[j+1]
					found = true
					break
				}
			}
			if !found {
				if isLast {
					// Return nil node but valid parent for insertion
					return nil, parent, -1, nil
				}
				return nil, nil, -1, fmt.Errorf("key %s not found", seg.Key)
			}
		}
	}

	return node, parent, indexInParent, nil
}

// SetValueAtPath sets a value at the given path
func SetValueAtPath(root *yaml.Node, segments []PathSegment, value string) error {
	if len(segments) == 0 {
		return fmt.Errorf("empty path")
	}

	// Navigate to parent
	parentSegments := segments[:len(segments)-1]
	lastSeg := segments[len(segments)-1]

	var parent *yaml.Node
	if len(parentSegments) == 0 {
		parent = root
		if parent.Kind == yaml.DocumentNode && len(parent.Content) > 0 {
			parent = parent.Content[0]
		}
	} else {
		var err error
		parent, _, _, err = GetNodeAtPath(root, parentSegments)
		if err != nil {
			return fmt.Errorf("failed to navigate to parent: %w", err)
		}
	}

	if lastSeg.Index >= 0 {
		// Array element
		if parent.Kind != yaml.SequenceNode {
			return fmt.Errorf("expected sequence for array index")
		}
		if lastSeg.Index >= len(parent.Content) {
			return fmt.Errorf("index out of bounds")
		}
		parent.Content[lastSeg.Index] = createScalarNode(value)
	} else {
		// Map key
		if parent.Kind != yaml.MappingNode {
			return fmt.Errorf("expected mapping for key access")
		}

		// Find existing key
		for i := 0; i < len(parent.Content); i += 2 {
			if i+1 < len(parent.Content) && parent.Content[i].Value == lastSeg.Key {
				parent.Content[i+1] = createScalarNode(value)
				return nil
			}
		}

		// Key not found, add it
		parent.Content = append(parent.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: lastSeg.Key, Tag: "!!str"},
			createScalarNode(value),
		)
	}

	return nil
}

// createScalarNode creates a YAML scalar node with the appropriate style
// For values containing Helm template syntax ({{ }}), we need to avoid quoting
func createScalarNode(value string) *yaml.Node {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: value,
	}

	// If the value contains Helm template syntax, use no tag to let YAML
	// output it without quotes. We'll post-process to ensure it's unquoted.
	if strings.Contains(value, "{{") {
		node.Tag = "!!str"
		node.Style = 0 // Default style, will be handled in post-processing
	} else {
		node.Tag = "!!str"
	}

	return node
}

// DeleteAtPath deletes the node at the given path
func DeleteAtPath(root *yaml.Node, segments []PathSegment) error {
	if len(segments) == 0 {
		return fmt.Errorf("empty path")
	}

	// Navigate to parent
	parentSegments := segments[:len(segments)-1]
	lastSeg := segments[len(segments)-1]

	var parent *yaml.Node
	if len(parentSegments) == 0 {
		parent = root
		if parent.Kind == yaml.DocumentNode && len(parent.Content) > 0 {
			parent = parent.Content[0]
		}
	} else {
		var err error
		parent, _, _, err = GetNodeAtPath(root, parentSegments)
		if err != nil {
			// Parent doesn't exist, nothing to delete
			return nil
		}
	}

	if parent == nil {
		return nil
	}

	if lastSeg.Index >= 0 {
		// Delete array element
		if parent.Kind != yaml.SequenceNode {
			return nil
		}
		if lastSeg.Index >= len(parent.Content) {
			return nil
		}
		parent.Content = append(parent.Content[:lastSeg.Index], parent.Content[lastSeg.Index+1:]...)
	} else {
		// Delete map key
		if parent.Kind != yaml.MappingNode {
			return nil
		}
		for i := 0; i < len(parent.Content); i += 2 {
			if parent.Content[i].Value == lastSeg.Key {
				// Remove both key and value
				parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
				return nil
			}
		}
	}

	return nil
}
