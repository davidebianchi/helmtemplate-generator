package transform

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func parseYAML(t *testing.T, input string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	err := yaml.Unmarshal([]byte(input), &node)
	require.NoError(t, err)
	return &node
}

func TestParsePath_DotNotation(t *testing.T) {
	segments, err := ParsePath(".metadata.name")
	require.NoError(t, err)
	require.Len(t, segments, 2)
	require.Equal(t, "metadata", segments[0].Key)
	require.Equal(t, -1, segments[0].Index)
	require.Equal(t, "name", segments[1].Key)
	require.Equal(t, -1, segments[1].Index)
}

func TestParsePath_WithArrayIndex(t *testing.T) {
	segments, err := ParsePath(".spec.containers[0].image")
	require.NoError(t, err)
	require.Len(t, segments, 4)
	require.Equal(t, 0, segments[2].Index)
	require.Equal(t, "image", segments[3].Key)
}

func TestParsePath_EmptyPath(t *testing.T) {
	segments, err := ParsePath("")
	require.NoError(t, err)
	require.Nil(t, segments)
}

func TestSetValueAtPath_ExistingKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: old-value")
	segments, _ := ParsePath(".metadata.name")

	err := SetValueAtPath(root, segments, "new-value")
	require.NoError(t, err)

	node, _, _, _ := GetNodeAtPath(root, segments)
	require.NotNil(t, node)
	require.Equal(t, "new-value", node.Value)
}

func TestSetValueAtPath_NewKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(".metadata.namespace")

	err := SetValueAtPath(root, segments, "default")
	require.NoError(t, err)

	node, _, _, _ := GetNodeAtPath(root, segments)
	require.NotNil(t, node)
	require.Equal(t, "default", node.Value)
}

func TestSetValueAtPath_EmptyPath(t *testing.T) {
	root := parseYAML(t, "key: value")
	err := SetValueAtPath(root, nil, "test")
	require.Error(t, err)
}

func TestDeleteAtPath_ExistingKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test\n  namespace: default")
	segments, _ := ParsePath(".metadata.namespace")

	err := DeleteAtPath(root, segments)
	require.NoError(t, err)

	node, _, _, _ := GetNodeAtPath(root, segments)
	require.Nil(t, node)
}

func TestDeleteAtPath_NonExistentKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(".metadata.namespace")

	err := DeleteAtPath(root, segments)
	require.NoError(t, err)
}

func TestDeleteAtPath_ArrayElement(t *testing.T) {
	root := parseYAML(t, "items:\n  - first\n  - second\n  - third")
	segments, _ := ParsePath(".items[1]")

	err := DeleteAtPath(root, segments)
	require.NoError(t, err)

	// Check that the sequence now has 2 elements
	itemsSegs, _ := ParsePath(".items")
	node, _, _, _ := GetNodeAtPath(root, itemsSegs)
	require.NotNil(t, node)
	require.Len(t, node.Content, 2)
}

func TestGetNodeAtPath_MissingIntermediateKey(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(".spec.replicas")

	_, _, _, err := GetNodeAtPath(root, segments)
	require.Error(t, err)
}

func TestGetNodeAtPath_ArrayOutOfBounds(t *testing.T) {
	root := parseYAML(t, "items:\n  - first")
	segments, _ := ParsePath(".items[5]")

	_, _, _, err := GetNodeAtPath(root, segments)
	require.Error(t, err)
}
