package transform

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// pathSegmentRe matches quoted key ["..."], key=value filter [key=value], wildcard [*], array index [N], or unquoted key
var pathSegmentRe = regexp.MustCompile(`\["([^"]+)"\]|\[([^=\]]+)=([^\]]+)\]|\[(\*)\]|\[(\d+)\]|([^.\[\]]+)`)

// PathSegment represents a segment in a YAML path
type PathSegment struct {
	Key       string // For map access
	Index     int    // For array access (-1 if not array)
	FilterKey string // For array filter access: [key=value]
	FilterVal string // For array filter access: [key=value]
	Wildcard  bool   // For wildcard array access: [*]
}

// ParsePath parses a JSONPath-like string into segments
// Examples:
//   - .metadata.name -> [{Key: "metadata"}, {Key: "name"}]
//   - .spec.containers[0].image -> [{Key: "spec"}, {Key: "containers"}, {Index: 0}, {Key: "image"}]
//   - .metadata.annotations["helm.sh/resource-policy"] -> [
//     {Key: "metadata"}, {Key: "annotations"}, {Key: "helm.sh/resource-policy"}]
//   - .spec.containers[0].env[name=FOO].value -> [
//     {Key: "spec"}, {Key: "containers"}, {Index: 0}, {Key: "env"}, {FilterKey: "name"}, {FilterVal: "FOO"}, {Key: "value"}]
func ParsePath(path string) ([]PathSegment, error) {
	if path == "" {
		return nil, nil
	}

	// Remove leading dot
	path = strings.TrimPrefix(path, ".")

	var segments []PathSegment
	matches := pathSegmentRe.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		switch {
		case match[1] != "":
			// Quoted key segment: ["key.with.dots"]
			segments = append(segments, PathSegment{Key: match[1], Index: -1})
		case match[2] != "" && match[3] != "":
			// Key=value filter segment: [name=FOO]
			segments = append(segments, PathSegment{Index: -1, FilterKey: match[2], FilterVal: match[3]})
		case match[4] != "":
			// Wildcard segment: [*]
			segments = append(segments, PathSegment{Index: -1, Wildcard: true})
		case match[5] != "":
			// Array index segment: [0]
			idx, err := strconv.Atoi(match[5])
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", match[5])
			}
			segments = append(segments, PathSegment{Index: idx})
		case match[6] != "":
			// Unquoted key segment
			segments = append(segments, PathSegment{Key: match[6], Index: -1})
		}
	}

	return segments, nil
}

// isFilter returns true if the segment uses key=value array filtering
func (s PathSegment) isFilter() bool {
	return s.FilterKey != ""
}

// containsWildcard returns true if any segment in the path uses [*]
func containsWildcard(segments []PathSegment) bool {
	for _, s := range segments {
		if s.Wildcard {
			return true
		}
	}
	return false
}

// splitAtWildcard splits segments into: before the wildcard, and after the wildcard.
// It navigates to the sequence node before the wildcard and calls fn for each element
// with the remaining segments.
func forEachInWildcard(root *yaml.Node, segments []PathSegment, fn func(elem *yaml.Node, rest []PathSegment) error) error {
	// Find the first wildcard
	wildcardIdx := -1
	for i, s := range segments {
		if s.Wildcard {
			wildcardIdx = i
			break
		}
	}
	if wildcardIdx < 0 {
		return fmt.Errorf("no wildcard segment found")
	}

	// Navigate to the sequence node before the wildcard
	var seqNode *yaml.Node
	if wildcardIdx == 0 {
		seqNode = root
		if seqNode.Kind == yaml.DocumentNode && len(seqNode.Content) > 0 {
			seqNode = seqNode.Content[0]
		}
	} else {
		var err error
		seqNode, _, _, err = GetNodeAtPath(root, segments[:wildcardIdx])
		if err != nil {
			return err
		}
		if seqNode == nil {
			return fmt.Errorf("node not found at path before wildcard")
		}
	}

	if seqNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("expected sequence for wildcard [*], got node kind %d", seqNode.Kind)
	}

	rest := segments[wildcardIdx+1:]
	for _, elem := range seqNode.Content {
		if err := fn(elem, rest); err != nil {
			return err
		}
	}
	return nil
}

// findInSequence finds the first element in a SequenceNode where the child
// map key FilterKey has value FilterVal. Returns the element and its index.
func findInSequence(node *yaml.Node, seg PathSegment) (*yaml.Node, int) {
	for i, elem := range node.Content {
		if elem.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j < len(elem.Content); j += 2 {
			if j+1 < len(elem.Content) &&
				elem.Content[j].Value == seg.FilterKey &&
				elem.Content[j+1].Value == seg.FilterVal {
				return elem, i
			}
		}
	}
	return nil, -1
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

		switch {
		case seg.Wildcard:
			return nil, nil, -1, fmt.Errorf("wildcard [*] is not supported in GetNodeAtPath; use SetValueAtPath or DeleteAtPath")
		case seg.isFilter():
			// Array filter access: [key=value]
			if node.Kind != yaml.SequenceNode {
				return nil, nil, -1, fmt.Errorf(
					"expected sequence at path segment [%s=%s]", seg.FilterKey, seg.FilterVal,
				)
			}
			found, idx := findInSequence(node, seg)
			if found == nil {
				return nil, nil, -1, fmt.Errorf("no element with %s=%s found", seg.FilterKey, seg.FilterVal)
			}
			indexInParent = idx
			node = found
		case seg.Index >= 0:
			// Array access
			if node.Kind != yaml.SequenceNode {
				return nil, nil, -1, fmt.Errorf("expected sequence at path segment [%d]", seg.Index)
			}
			if seg.Index >= len(node.Content) {
				return nil, nil, -1, fmt.Errorf("index %d out of bounds", seg.Index)
			}
			indexInParent = seg.Index
			node = node.Content[seg.Index]
		default:
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

// SetValueAtPath sets a value at the given path, creating intermediate mapping
// nodes as needed. For example, setting ".metadata.annotations.newKey" will
// create the "annotations" map if it doesn't exist.
func SetValueAtPath(root *yaml.Node, segments []PathSegment, value string) error {
	if len(segments) == 0 {
		return fmt.Errorf("empty path")
	}

	if containsWildcard(segments) {
		return forEachInWildcard(root, segments, func(elem *yaml.Node, rest []PathSegment) error {
			// Wrap element in a temporary document for recursive call
			wrapper := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{elem}}
			return SetValueAtPath(wrapper, rest, value)
		})
	}

	// Navigate to parent, creating intermediate maps as needed
	parentSegments := segments[:len(segments)-1]
	lastSeg := segments[len(segments)-1]

	var parent *yaml.Node
	if len(parentSegments) == 0 {
		parent = root
		if parent.Kind == yaml.DocumentNode && len(parent.Content) > 0 {
			parent = parent.Content[0]
		}
	} else {
		parent = navigateOrCreate(root, parentSegments)
		if parent == nil {
			return fmt.Errorf("failed to navigate to parent path")
		}
	}

	switch {
	case lastSeg.isFilter():
		// Array filter access: find matching element and replace it
		if parent.Kind != yaml.SequenceNode {
			return fmt.Errorf(
				"expected sequence for filter [%s=%s], got node kind %d",
				lastSeg.FilterKey, lastSeg.FilterVal, parent.Kind,
			)
		}
		found, idx := findInSequence(parent, lastSeg)
		if found == nil {
			return fmt.Errorf("no element with %s=%s found", lastSeg.FilterKey, lastSeg.FilterVal)
		}
		parent.Content[idx] = createScalarNode(value)
	case lastSeg.Index >= 0:
		// Array element
		if parent.Kind != yaml.SequenceNode {
			return fmt.Errorf(
				"expected sequence for array index at segment [%d], got node kind %d",
				lastSeg.Index, parent.Kind,
			)
		}
		if lastSeg.Index >= len(parent.Content) {
			return fmt.Errorf("index %d out of bounds (length %d)", lastSeg.Index, len(parent.Content))
		}
		parent.Content[lastSeg.Index] = createScalarNode(value)
	default:
		// Map key
		if parent.Kind != yaml.MappingNode {
			return fmt.Errorf(
				"expected mapping for key %q, got node kind %d (tag: %s, value: %q)",
				lastSeg.Key, parent.Kind, parent.Tag, parent.Value,
			)
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

// navigateOrCreate traverses the YAML tree following the segments, creating
// intermediate MappingNodes for missing keys.
func navigateOrCreate(root *yaml.Node, segments []PathSegment) *yaml.Node {
	node := root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	for _, seg := range segments {
		switch {
		case seg.Wildcard:
			return nil
		case seg.isFilter():
			// Array filter access — cannot create elements
			if node.Kind != yaml.SequenceNode {
				return nil
			}
			found, _ := findInSequence(node, seg)
			if found == nil {
				return nil
			}
			node = found
		case seg.Index >= 0:
			// Array access — cannot create intermediate arrays
			if node.Kind != yaml.SequenceNode || seg.Index >= len(node.Content) {
				return nil
			}
			node = node.Content[seg.Index]
		default:
			if node.Kind != yaml.MappingNode {
				return nil
			}
			found := false
			for j := 0; j < len(node.Content); j += 2 {
				if j+1 < len(node.Content) && node.Content[j].Value == seg.Key {
					child := node.Content[j+1]
					if child.Kind == yaml.ScalarNode {
						// Replace scalar/null value with a new mapping node
						newMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
						node.Content[j+1] = newMap
						child = newMap
					}
					node = child
					found = true
					break
				}
			}
			if !found {
				// Create a new mapping node for the missing key
				newMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
				node.Content = append(node.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: seg.Key, Tag: "!!str"},
					newMap,
				)
				node = newMap
			}
		}
	}

	return node
}

// createScalarNode creates a YAML scalar node.
// Values containing Helm template syntax ({{ }}) are post-processed by
// unquoteHelmTemplates to remove YAML quotes around them.
func createScalarNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: value,
		Tag:   "!!str",
	}
}

// DeleteAtPath deletes the node at the given path
func DeleteAtPath(root *yaml.Node, segments []PathSegment) error {
	if len(segments) == 0 {
		return fmt.Errorf("empty path")
	}

	if containsWildcard(segments) {
		return forEachInWildcard(root, segments, func(elem *yaml.Node, rest []PathSegment) error {
			wrapper := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{elem}}
			return DeleteAtPath(wrapper, rest)
		})
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

	switch {
	case lastSeg.isFilter():
		// Delete array element by key=value filter
		if parent.Kind != yaml.SequenceNode {
			return nil
		}
		_, idx := findInSequence(parent, lastSeg)
		if idx < 0 {
			return nil
		}
		parent.Content = append(parent.Content[:idx], parent.Content[idx+1:]...)
	case lastSeg.Index >= 0:
		// Delete array element
		if parent.Kind != yaml.SequenceNode {
			return nil
		}
		if lastSeg.Index >= len(parent.Content) {
			return nil
		}
		parent.Content = append(parent.Content[:lastSeg.Index], parent.Content[lastSeg.Index+1:]...)
	default:
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
