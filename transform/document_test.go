package transform

import (
	"strings"
	"testing"
)

func TestParseDocuments_SingleDocument(t *testing.T) {
	input := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test")
	docs, err := ParseDocuments(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}
}

func TestParseDocuments_MultipleDocuments(t *testing.T) {
	input := []byte("kind: ConfigMap\nmetadata:\n  name: a\n---\nkind: Service\nmetadata:\n  name: b")
	docs, err := ParseDocuments(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
}

func TestParseDocuments_EmptyInput(t *testing.T) {
	docs, err := ParseDocuments([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 documents, got %d", len(docs))
	}
}

func TestParseDocuments_InvalidYAML(t *testing.T) {
	input := []byte(":\n  :\n    - [invalid")
	_, err := ParseDocuments(input)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestGetKind(t *testing.T) {
	docs, _ := ParseDocuments([]byte("kind: Deployment\nmetadata:\n  name: test"))
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document")
	}
	if kind := docs[0].GetKind(); kind != "Deployment" {
		t.Errorf("expected Deployment, got %s", kind)
	}
}

func TestGetKind_NoKind(t *testing.T) {
	docs, _ := ParseDocuments([]byte("apiVersion: v1\nmetadata:\n  name: test"))
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document")
	}
	if kind := docs[0].GetKind(); kind != "" {
		t.Errorf("expected empty string, got %s", kind)
	}
}

func TestGetName(t *testing.T) {
	docs, _ := ParseDocuments([]byte("kind: ConfigMap\nmetadata:\n  name: my-config"))
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document")
	}
	if name := docs[0].GetName(); name != "my-config" {
		t.Errorf("expected my-config, got %s", name)
	}
}

func TestGetName_NoMetadata(t *testing.T) {
	docs, _ := ParseDocuments([]byte("kind: ConfigMap\ndata:\n  key: value"))
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document")
	}
	if name := docs[0].GetName(); name != "" {
		t.Errorf("expected empty string, got %s", name)
	}
}

func TestGetLabels(t *testing.T) {
	docs, _ := ParseDocuments([]byte("metadata:\n  labels:\n    app: myapp\n    version: v1"))
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document")
	}
	labels := docs[0].GetLabels()
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels["app"] != "myapp" {
		t.Errorf("expected app=myapp, got %s", labels["app"])
	}
	if labels["version"] != "v1" {
		t.Errorf("expected version=v1, got %s", labels["version"])
	}
}

func TestGetLabels_NoLabels(t *testing.T) {
	docs, _ := ParseDocuments([]byte("metadata:\n  name: test"))
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document")
	}
	if labels := docs[0].GetLabels(); labels != nil {
		t.Errorf("expected nil, got %v", labels)
	}
}

func TestUnquoteHelmTemplates_DoubleQuotes(t *testing.T) {
	input := `namespace: "{{ .Release.Namespace }}"`
	result := unquoteHelmTemplates(input)
	expected := `namespace: {{ .Release.Namespace }}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestUnquoteHelmTemplates_SingleQuotes(t *testing.T) {
	input := `namespace: '{{ .Release.Namespace }}'`
	result := unquoteHelmTemplates(input)
	expected := `namespace: {{ .Release.Namespace }}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestUnquoteHelmTemplates_NoTemplates(t *testing.T) {
	input := `name: "regular-value"`
	result := unquoteHelmTemplates(input)
	// Should not modify lines without {{ }}
	if result != input {
		t.Errorf("expected unchanged, got %q", result)
	}
}

func TestSerialize_RoundTrip(t *testing.T) {
	input := "kind: ConfigMap\nmetadata:\n  name: test\ndata:\n  key: value"
	docs, err := ParseDocuments([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document")
	}

	output, err := docs[0].Serialize()
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}

	if !strings.Contains(output, "kind: ConfigMap") {
		t.Errorf("expected output to contain 'kind: ConfigMap'\nGot:\n%s", output)
	}
	if !strings.Contains(output, "name: test") {
		t.Errorf("expected output to contain 'name: test'\nGot:\n%s", output)
	}
}
