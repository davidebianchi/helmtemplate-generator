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

func TestParsePath_QuotedKey(t *testing.T) {
	segments, err := ParsePath(`.metadata.annotations["helm.sh/resource-policy"]`)
	require.NoError(t, err)
	require.Len(t, segments, 3)
	require.Equal(t, "metadata", segments[0].Key)
	require.Equal(t, "annotations", segments[1].Key)
	require.Equal(t, "helm.sh/resource-policy", segments[2].Key)
}

func TestSetValueAtPath_QuotedKeyWithDots(t *testing.T) {
	root := parseYAML(t, "metadata:\n  annotations:\n    existing: value")
	segments, _ := ParsePath(`.metadata.annotations["helm.sh/resource-policy"]`)

	err := SetValueAtPath(root, segments, "keep")
	require.NoError(t, err)

	node, _, _, err := GetNodeAtPath(root, segments)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, "keep", node.Value)
}

func TestSetValueAtPath_QuotedKeyCreatesIntermediateMap(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(`.metadata.annotations["helm.sh/resource-policy"]`)

	err := SetValueAtPath(root, segments, "keep")
	require.NoError(t, err)

	node, _, _, err := GetNodeAtPath(root, segments)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, "keep", node.Value)
}

func TestSetValueAtPath_CreatesIntermediateMap(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(".metadata.annotations.mykey")

	err := SetValueAtPath(root, segments, "myvalue")
	require.NoError(t, err)

	node, _, _, err := GetNodeAtPath(root, segments)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, "myvalue", node.Value)
}

func TestSetValueAtPath_CreatesMultipleIntermediateMaps(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, _ := ParsePath(".metadata.annotations.deep.nested")

	err := SetValueAtPath(root, segments, "value")
	require.NoError(t, err)

	node, _, _, err := GetNodeAtPath(root, segments)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, "value", node.Value)
}

func TestParsePath_KeyValueFilter(t *testing.T) {
	segments, err := ParsePath(".spec.containers[0].env[name=FOO].value")
	require.NoError(t, err)
	require.Len(t, segments, 6)
	require.Equal(t, "spec", segments[0].Key)
	require.Equal(t, "containers", segments[1].Key)
	require.Equal(t, 0, segments[2].Index)
	require.Equal(t, "env", segments[3].Key)
	require.Equal(t, "name", segments[4].FilterKey)
	require.Equal(t, "FOO", segments[4].FilterVal)
	require.Equal(t, -1, segments[4].Index)
	require.Equal(t, "value", segments[5].Key)
}

func TestGetNodeAtPath_KeyValueFilter(t *testing.T) {
	root := parseYAML(t, `
env:
  - name: FOO
    value: bar
  - name: BAZ
    value: qux
`)
	segments, err := ParsePath(".env[name=BAZ].value")
	require.NoError(t, err)

	node, _, _, err := GetNodeAtPath(root, segments)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, "qux", node.Value)
}

func TestGetNodeAtPath_KeyValueFilter_NotFound(t *testing.T) {
	root := parseYAML(t, `
env:
  - name: FOO
    value: bar
`)
	segments, err := ParsePath(".env[name=MISSING].value")
	require.NoError(t, err)

	_, _, _, err = GetNodeAtPath(root, segments)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no element with name=MISSING found")
}

func TestSetValueAtPath_KeyValueFilter(t *testing.T) {
	root := parseYAML(t, `
env:
  - name: FOO
    value: bar
  - name: DATABASE_URL
    value: postgres://localhost
`)
	segments, err := ParsePath(".env[name=DATABASE_URL].value")
	require.NoError(t, err)

	err = SetValueAtPath(root, segments, "{{ .Values.databaseURL }}")
	require.NoError(t, err)

	node, _, _, err := GetNodeAtPath(root, segments)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, "{{ .Values.databaseURL }}", node.Value)
}

func TestSetValueAtPath_FilterAsLastSegment_SetValue(t *testing.T) {
	root := parseYAML(t, `
env:
  - name: FOO
    value: bar
  - name: TARGET_NS
    value: old-value
`)
	segments, err := ParsePath(".env[name=TARGET_NS]")
	require.NoError(t, err)

	// Setting a value on a filter-matched element should replace the element
	err = SetValueAtPath(root, segments, "replaced")
	require.NoError(t, err)

	envSegs, _ := ParsePath(".env")
	envNode, _, _, _ := GetNodeAtPath(root, envSegs)
	require.NotNil(t, envNode)
	require.Equal(t, "replaced", envNode.Content[1].Value)
}

func TestSetValueAtPath_NullIntermediateValue(t *testing.T) {
	root := parseYAML(t, "metadata:\n  annotations: null")
	segments, _ := ParsePath(".metadata.annotations.mykey")

	err := SetValueAtPath(root, segments, "myvalue")
	require.NoError(t, err)

	node, _, _, err := GetNodeAtPath(root, segments)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, "myvalue", node.Value)
}

func TestSetValueAtPath_ScalarIntermediateValue(t *testing.T) {
	root := parseYAML(t, "metadata:\n  annotations: some-scalar")
	segments, _ := ParsePath(".metadata.annotations.mykey")

	err := SetValueAtPath(root, segments, "myvalue")
	require.NoError(t, err)

	node, _, _, err := GetNodeAtPath(root, segments)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, "myvalue", node.Value)
}

func TestParsePath_Wildcard(t *testing.T) {
	segments, err := ParsePath(".webhooks[*].clientConfig.service.namespace")
	require.NoError(t, err)
	require.Len(t, segments, 5)
	require.Equal(t, "webhooks", segments[0].Key)
	require.True(t, segments[1].Wildcard)
	require.Equal(t, -1, segments[1].Index)
	require.Equal(t, "clientConfig", segments[2].Key)
	require.Equal(t, "service", segments[3].Key)
	require.Equal(t, "namespace", segments[4].Key)
}

func TestSetValueAtPath_Wildcard(t *testing.T) {
	root := parseYAML(t, `
webhooks:
  - name: webhook1
    clientConfig:
      service:
        namespace: ns1
  - name: webhook2
    clientConfig:
      service:
        namespace: ns2
`)
	segments, err := ParsePath(".webhooks[*].clientConfig.service.namespace")
	require.NoError(t, err)

	err = SetValueAtPath(root, segments, "{{ .Release.Namespace }}")
	require.NoError(t, err)

	// Verify both elements were updated
	s1, _ := ParsePath(".webhooks[0].clientConfig.service.namespace")
	node1, _, _, err := GetNodeAtPath(root, s1)
	require.NoError(t, err)
	require.Equal(t, "{{ .Release.Namespace }}", node1.Value)

	s2, _ := ParsePath(".webhooks[1].clientConfig.service.namespace")
	node2, _, _, err := GetNodeAtPath(root, s2)
	require.NoError(t, err)
	require.Equal(t, "{{ .Release.Namespace }}", node2.Value)
}

func TestDeleteAtPath_Wildcard(t *testing.T) {
	root := parseYAML(t, `
items:
  - name: a
    extra: remove-me
  - name: b
    extra: remove-me-too
`)
	segments, err := ParsePath(".items[*].extra")
	require.NoError(t, err)

	err = DeleteAtPath(root, segments)
	require.NoError(t, err)

	// Verify extra was deleted from both
	s1, _ := ParsePath(".items[0].extra")
	node1, _, _, _ := GetNodeAtPath(root, s1)
	require.Nil(t, node1)

	s2, _ := ParsePath(".items[1].extra")
	node2, _, _, _ := GetNodeAtPath(root, s2)
	require.Nil(t, node2)

	// Verify name fields still exist
	n1, _ := ParsePath(".items[0].name")
	nameNode, _, _, err := GetNodeAtPath(root, n1)
	require.NoError(t, err)
	require.Equal(t, "a", nameNode.Value)
}

func TestSetValueAtPath_Wildcard_NotSequence(t *testing.T) {
	root := parseYAML(t, "metadata:\n  name: test")
	segments, err := ParsePath(".metadata[*].name")
	require.NoError(t, err)

	err = SetValueAtPath(root, segments, "value")
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected sequence for wildcard")
}

func TestDeleteAtPath_KeyValueFilter(t *testing.T) {
	root := parseYAML(t, `
env:
  - name: FOO
    value: bar
  - name: TO_DELETE
    value: gone
  - name: BAZ
    value: qux
`)
	segments, err := ParsePath(".env[name=TO_DELETE]")
	require.NoError(t, err)

	err = DeleteAtPath(root, segments)
	require.NoError(t, err)

	envSegs, _ := ParsePath(".env")
	node, _, _, _ := GetNodeAtPath(root, envSegs)
	require.NotNil(t, node)
	require.Len(t, node.Content, 2)
}
