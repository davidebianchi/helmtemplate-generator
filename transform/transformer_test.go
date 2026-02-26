package transform

import (
	"strings"
	"testing"

	"github.com/davidebianchi/helmtemplate-generator/config"
)

func TestTransform_SimpleReplacement(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Path:  ".metadata.namespace",
				Value: `{{ .Release.Namespace }}`,
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  key: value`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(output, `namespace: {{ .Release.Namespace }}`) {
		t.Errorf("Expected namespace to be replaced with template\nGot:\n%s", output)
	}
}

func TestTransform_GlobalDelete(t *testing.T) {
	cfg := &config.Config{
		Global: &config.GlobalRules{
			Delete: []string{".status", ".metadata.uid"},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  uid: abc-123
data:
  key: value
status:
  ready: true`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if strings.Contains(output, "uid:") {
		t.Errorf("Expected uid to be deleted\nGot:\n%s", output)
	}
	if strings.Contains(output, "status:") {
		t.Errorf("Expected status to be deleted\nGot:\n%s", output)
	}
}

func TestTransform_MatchKind(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Path:  ".spec.replicas",
				Value: `{{ .Values.replicas }}`,
			},
		},
	}

	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 3`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(output, `replicas: {{ .Values.replicas }}`) {
		t.Errorf("Expected replicas to be replaced with template\nGot:\n%s", output)
	}
}

func TestTransform_MatchKindNoMatch(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Path:  ".spec.replicas",
				Value: `{{ .Values.replicas }}`,
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  replicas: "3"`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Should not modify ConfigMap since rule matches only Deployment
	if strings.Contains(output, `{{ .Values.replicas }}`) {
		t.Errorf("Expected ConfigMap to be unchanged\nGot:\n%s", output)
	}
}

func TestTransform_DocumentWrap(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Wrap: &config.Wrap{
					Before: "{{- if .Values.enabled }}",
					After:  "{{- end }}",
				},
			},
		},
	}

	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 1`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.HasPrefix(output, "{{- if .Values.enabled }}") {
		t.Errorf("Expected output to start with if condition\nGot:\n%s", output)
	}
	if !strings.HasSuffix(strings.TrimSpace(output), "{{- end }}") {
		t.Errorf("Expected output to end with end\nGot:\n%s", output)
	}
}

func TestTransform_MultipleChanges(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Changes: []config.Change{
					{
						Path:  ".spec.replicas",
						Value: `{{ .Values.replicas }}`,
					},
					{
						Path:  ".metadata.namespace",
						Value: `{{ .Release.Namespace }}`,
					},
				},
			},
		},
	}

	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  namespace: default
spec:
  replicas: 3`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(output, `replicas: {{ .Values.replicas }}`) {
		t.Errorf("Expected replicas to be replaced\nGot:\n%s", output)
	}
	if !strings.Contains(output, `namespace: {{ .Release.Namespace }}`) {
		t.Errorf("Expected namespace to be replaced\nGot:\n%s", output)
	}
}

func TestTransform_MultiDocument(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Path:  ".metadata.namespace",
				Value: `{{ .Release.Namespace }}`,
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  namespace: ns1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
  namespace: ns2`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Should have two documents separated by ---
	docs := strings.Split(output, "---")
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d\nOutput:\n%s", len(docs), output)
	}

	// Both should have the namespace replaced
	count := strings.Count(output, `{{ .Release.Namespace }}`)
	if count != 2 {
		t.Errorf("Expected 2 namespace replacements, got %d\nOutput:\n%s", count, output)
	}
}

func TestTransform_MatchNameWildcard(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{
					Kinds: []string{"Deployment"},
					Name:  "my-*",
				},
				Path:  ".spec.replicas",
				Value: `{{ .Values.replicas }}`,
			},
		},
	}

	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-deployment
spec:
  replicas: 3`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Only first deployment should have replicas replaced
	count := strings.Count(output, `{{ .Values.replicas }}`)
	if count != 1 {
		t.Errorf("Expected 1 replacement (only my-*), got %d\nOutput:\n%s", count, output)
	}
}

func TestTransform_ReplaceWith(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Path:  ".spec.template.spec.imagePullSecrets",
				ReplaceWith: `{{- with .Values.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 8 }}
{{- end }}`,
			},
		},
	}

	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  template:
    spec:
      imagePullSecrets:
        - name: my-secret`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(output, "{{- with .Values.imagePullSecrets }}") {
		t.Errorf("Expected replaceWith content in output\nGot:\n%s", output)
	}
	if strings.Contains(output, "my-secret") {
		t.Errorf("Expected original value to be replaced\nGot:\n%s", output)
	}
}

func TestTransform_UnknownAction(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Path:   ".metadata.name",
				Action: "patch",
			},
		},
	}

	input := `kind: ConfigMap
metadata:
  name: test`

	transformer := New(cfg)
	_, err := transformer.Transform([]byte(input))
	if err == nil {
		t.Error("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("expected 'unknown action' in error, got: %v", err)
	}
}

func TestTransform_NilConfig(t *testing.T) {
	transformer := New(nil)
	input := `kind: ConfigMap
metadata:
  name: test
data:
  key: value`

	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "name: test") {
		t.Errorf("expected pass-through output\nGot:\n%s", output)
	}
}

func TestTransformDocuments(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Path:  ".metadata.namespace",
				Value: `{{ .Release.Namespace }}`,
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  namespace: ns1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
  namespace: ns2
spec:
  replicas: 1`

	transformer := New(cfg)
	docs, err := transformer.TransformDocuments([]byte(input))
	if err != nil {
		t.Fatalf("TransformDocuments failed: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}

	if docs[0].Kind != "ConfigMap" || docs[0].Name != "config1" {
		t.Errorf("doc 0: expected ConfigMap/config1, got %s/%s", docs[0].Kind, docs[0].Name)
	}
	if docs[1].Kind != "Deployment" || docs[1].Name != "my-deploy" {
		t.Errorf("doc 1: expected Deployment/my-deploy, got %s/%s", docs[1].Kind, docs[1].Name)
	}
}

func TestTransform_DeleteAction(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Path:   ".metadata.annotations",
				Action: "delete",
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  annotations:
    key: value
data:
  foo: bar`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if strings.Contains(output, "annotations:") {
		t.Errorf("Expected annotations to be deleted\nGot:\n%s", output)
	}
}
