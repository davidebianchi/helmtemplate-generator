package transform

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func parseYAML(t *testing.T, input string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(input), &node); err != nil {
		t.Fatalf("failed to parse test YAML: %v", err)
	}
	return &node
}

func TestParsePath_DotNotation(t *testing.T) {
	segments, err := ParsePath(".metadata.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segments))
	}
	if segments[0].Key != "metadata" || segments[0].Index != -1 {
		t.Errorf("segment 0: expected Key=metadata Index=-1, got Key=%s Index=%d", segments[0].Key, segments[0].Index)
	}
	if segments[1].Key != "name" || segments[1].Index != -1 {
		t.Errorf("segment 1: expected Key=name Index=-1, got Key=%s Index=%d", segments[1].Key, segments[1].Index)
	}
}

func TestParsePath_WithArrayIndex(t *testing.T) {
	segments, err := ParsePath(".spec.containers[0].image")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(segments))
	}
	if segments[2].Index != 0 {
		t.Errorf("segment 2: expected Index=0, got Index=%d", segments[2].Index)
	}
	if segments[3].Key != "image" {
		t.Errorf("segment 3: expected Key=image, got Key=%s", segments[3].Key)
	}
}

func TestParsePath_EmptyPath(t *testing.T) {
	segments, err := ParsePath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if segments != nil {
		t.Errorf("expected nil segments for empty path, got %v", segments)
	}
}

func TestSetValueAtPath_ExistingKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: old-value")
	segments, _ := ParsePath(".metadata.name")

	err := SetValueAtPath(root, segments, "new-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node, _, _, _ := GetNodeAtPath(root, segments)
	if node == nil || node.Value != "new-value" {
		t.Errorf("expected new-value, got %v", node)
	}
}

func TestSetValueAtPath_NewKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(".metadata.namespace")

	err := SetValueAtPath(root, segments, "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node, _, _, _ := GetNodeAtPath(root, segments)
	if node == nil || node.Value != "default" {
		t.Errorf("expected 'default', got %v", node)
	}
}

func TestSetValueAtPath_EmptyPath(t *testing.T) {
	root := parseYAML(t, "key: value")
	err := SetValueAtPath(root, nil, "test")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestDeleteAtPath_ExistingKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test\n  namespace: default")
	segments, _ := ParsePath(".metadata.namespace")

	err := DeleteAtPath(root, segments)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node, _, _, _ := GetNodeAtPath(root, segments)
	if node != nil {
		t.Errorf("expected key to be deleted, but found node with value %s", node.Value)
	}
}

func TestDeleteAtPath_NonExistentKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(".metadata.namespace")

	err := DeleteAtPath(root, segments)
	if err != nil {
		t.Errorf("expected no error when deleting non-existent key, got %v", err)
	}
}

func TestDeleteAtPath_ArrayElement(t *testing.T) {
	root := parseYAML(t, "items:\n  - first\n  - second\n  - third")
	segments, _ := ParsePath(".items[1]")

	err := DeleteAtPath(root, segments)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that the sequence now has 2 elements
	itemsSegs, _ := ParsePath(".items")
	node, _, _, _ := GetNodeAtPath(root, itemsSegs)
	if node == nil {
		t.Fatal("items node not found")
	}
	if len(node.Content) != 2 {
		t.Errorf("expected 2 items after deletion, got %d", len(node.Content))
	}
}

func TestGetNodeAtPath_MissingIntermediateKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(".spec.replicas")

	_, _, _, err := GetNodeAtPath(root, segments)
	if err == nil {
		t.Error("expected error for missing intermediate key")
	}
}

func TestGetNodeAtPath_ArrayOutOfBounds(t *testing.T) {
	root := parseYAML(t, "items:\n  - first")
	segments, _ := ParsePath(".items[5]")

	_, _, _, err := GetNodeAtPath(root, segments)
	if err == nil {
		t.Error("expected error for out-of-bounds index")
	}
}
