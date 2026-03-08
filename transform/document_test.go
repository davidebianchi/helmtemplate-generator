package transform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDocuments_SingleDocument(t *testing.T) {
	input := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test")
	docs, err := ParseDocuments(input)
	require.NoError(t, err)
	require.Len(t, docs, 1)
}

func TestParseDocuments_MultipleDocuments(t *testing.T) {
	input := []byte("kind: ConfigMap\nmetadata:\n  name: a\n---\nkind: Service\nmetadata:\n  name: b")
	docs, err := ParseDocuments(input)
	require.NoError(t, err)
	require.Len(t, docs, 2)
}

func TestParseDocuments_EmptyInput(t *testing.T) {
	docs, err := ParseDocuments([]byte(""))
	require.NoError(t, err)
	require.Len(t, docs, 0)
}

func TestParseDocuments_InvalidYAML(t *testing.T) {
	input := []byte(":\n  :\n    - [invalid")
	_, err := ParseDocuments(input)
	require.Error(t, err)
}

func TestGetKind(t *testing.T) {
	docs, _ := ParseDocuments([]byte("kind: Deployment\nmetadata:\n  name: test"))
	require.NotEmpty(t, docs, "expected at least 1 document")
	require.Equal(t, kindDeployment, docs[0].GetKind())
}

func TestGetKind_NoKind(t *testing.T) {
	docs, _ := ParseDocuments([]byte("apiVersion: v1\nmetadata:\n  name: test"))
	require.NotEmpty(t, docs, "expected at least 1 document")
	require.Equal(t, "", docs[0].GetKind())
}

func TestGetName(t *testing.T) {
	docs, _ := ParseDocuments([]byte("kind: ConfigMap\nmetadata:\n  name: my-config"))
	require.NotEmpty(t, docs, "expected at least 1 document")
	require.Equal(t, "my-config", docs[0].GetName())
}

func TestGetName_NoMetadata(t *testing.T) {
	docs, _ := ParseDocuments([]byte("kind: ConfigMap\ndata:\n  key: value"))
	require.NotEmpty(t, docs, "expected at least 1 document")
	require.Equal(t, "", docs[0].GetName())
}

func TestGetLabels(t *testing.T) {
	docs, _ := ParseDocuments([]byte("metadata:\n  labels:\n    app: myapp\n    version: v1"))
	require.NotEmpty(t, docs, "expected at least 1 document")
	labels := docs[0].GetLabels()
	require.NotNil(t, labels)
	require.Equal(t, "myapp", labels["app"])
	require.Equal(t, "v1", labels["version"])
}

func TestGetLabels_NoLabels(t *testing.T) {
	docs, _ := ParseDocuments([]byte("metadata:\n  name: test"))
	require.NotEmpty(t, docs, "expected at least 1 document")
	require.Nil(t, docs[0].GetLabels())
}

func TestUnquoteHelmTemplates_DoubleQuotes(t *testing.T) {
	input := `namespace: "{{ .Release.Namespace }}"`
	result := unquoteHelmTemplates(input)
	expected := `namespace: {{ .Release.Namespace }}`
	require.Equal(t, expected, result)
}

func TestUnquoteHelmTemplates_SingleQuotes(t *testing.T) {
	input := `namespace: '{{ .Release.Namespace }}'`
	result := unquoteHelmTemplates(input)
	expected := `namespace: {{ .Release.Namespace }}`
	require.Equal(t, expected, result)
}

func TestUnquoteHelmTemplates_NoTemplates(t *testing.T) {
	input := `name: "regular-value"`
	result := unquoteHelmTemplates(input)
	// Should not modify lines without {{ }}
	require.Equal(t, input, result)
}

func TestSerialize_RoundTrip(t *testing.T) {
	input := "kind: ConfigMap\nmetadata:\n  name: test\ndata:\n  key: value"
	docs, err := ParseDocuments([]byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, docs, "expected at least 1 document")

	output, err := docs[0].Serialize()
	require.NoError(t, err)

	require.Contains(t, output, "kind: ConfigMap")
	require.Contains(t, output, "name: test")
}
