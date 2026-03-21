package transform

import (
	"strings"
	"testing"

	"github.com/davidebianchi/helmtemplate-generator/config"
	"github.com/stretchr/testify/require"
)

const (
	kindDeployment = "Deployment"
	kindConfigMap  = "ConfigMap"
	kindService    = "Service"
	nameAppA       = "app-a"
)

func TestTransform_SimpleReplacement(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:  ".metadata.namespace",
						Value: `{{ .Release.Namespace }}`,
					},
				},
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
	require.NoError(t, err)

	require.Contains(t, output, `namespace: {{ .Release.Namespace }}`)
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
	require.NoError(t, err)

	require.NotContains(t, output, "uid:")
	require.NotContains(t, output, "status:")
}

func TestTransform_MatchKind(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Changes: []config.Change{
					{
						Path:  ".spec.replicas",
						Value: `{{ .Values.replicas }}`,
					},
				},
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
	require.NoError(t, err)

	require.Contains(t, output, `replicas: {{ .Values.replicas }}`)
}

func TestTransform_MatchKindNoMatch(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Changes: []config.Change{
					{
						Path:  ".spec.replicas",
						Value: `{{ .Values.replicas }}`,
					},
				},
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
	require.NoError(t, err)

	// Should not modify ConfigMap since rule matches only Deployment
	require.NotContains(t, output, `{{ .Values.replicas }}`)
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
	require.NoError(t, err)

	require.True(t,
		strings.HasPrefix(output, "{{- if .Values.enabled }}"),
		"Expected to start with if condition\nGot:\n%s",
		output,
	)
	require.True(t,
		strings.HasSuffix(strings.TrimSpace(output), "{{- end }}"),
		"Expected to end with end\nGot:\n%s",
		output,
	)
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
	require.NoError(t, err)

	require.Contains(t, output, `replicas: {{ .Values.replicas }}`)
	require.Contains(t, output, `namespace: {{ .Release.Namespace }}`)
}

func TestTransform_MultiDocument(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:  ".metadata.namespace",
						Value: `{{ .Release.Namespace }}`,
					},
				},
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
	require.NoError(t, err)

	// Should have two documents separated by ---
	docs := strings.Split(output, "---")
	require.Len(t, docs, 2)

	// Both should have the namespace replaced
	count := strings.Count(output, `{{ .Release.Namespace }}`)
	require.Equal(t, 2, count, "Expected 2 namespace replacements\nOutput:\n%s", output)
}

func TestTransform_MatchNameWildcard(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{
					Kinds: []string{"Deployment"},
					Names: []string{"my-*"},
				},
				Changes: []config.Change{
					{
						Path:  ".spec.replicas",
						Value: `{{ .Values.replicas }}`,
					},
				},
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
	require.NoError(t, err)

	// Only first deployment should have replicas replaced
	count := strings.Count(output, `{{ .Values.replicas }}`)
	require.Equal(t, 1, count, "Expected 1 replacement (only my-*)\nOutput:\n%s", output)
}

func TestTransform_ReplaceWith(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Changes: []config.Change{
					{
						Path: ".spec.template.spec.imagePullSecrets",
						ReplaceWith: `{{- with .Values.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 8 }}
{{- end }}`,
					},
				},
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
	require.NoError(t, err)

	require.Contains(t, output, "{{- with .Values.imagePullSecrets }}")
	require.NotContains(t, output, "my-secret")
}

func TestTransform_ReplaceWith_RootPath(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Changes: []config.Change{
					{
						Path:        ".",
						ReplaceWith: `{{- include "mychart.deployment" . }}`,
					},
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
	require.NoError(t, err)

	require.Contains(t, output, `{{- include "mychart.deployment" . }}`)
	require.NotContains(t, output, "replicas:")
}

func TestTransform_AppendWith_RootPath(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{"Deployment"}},
				Changes: []config.Change{
					{
						Path:       ".",
						AppendWith: `{{- include "mychart.extra" . | nindent 0 }}`,
					},
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
	require.NoError(t, err)

	// Existing content should be preserved
	require.Contains(t, output, "replicas: 1")
	require.Contains(t, output, "name: my-deployment")
	// Appended content should be at the end
	require.Contains(t, output, `{{- include "mychart.extra" . | nindent 0 }}`)
	require.True(t,
		strings.HasSuffix(strings.TrimSpace(output), `{{- include "mychart.extra" . | nindent 0 }}`),
		"Expected appended content at end\nGot:\n%s", output,
	)
}

func TestTransform_UnknownAction(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:   ".metadata.name",
						Action: "patch",
						Value:  "test",
					},
				},
			},
		},
	}

	input := `kind: ConfigMap
metadata:
  name: test`

	transformer := New(cfg)
	_, err := transformer.Transform([]byte(input))
	require.Error(t, err)
	require.ErrorContains(t, err, "unknown action")
}

func TestTransform_NilConfig(t *testing.T) {
	transformer := New(nil)
	input := `kind: ConfigMap
metadata:
  name: test
data:
  key: value`

	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)
	require.Contains(t, output, "name: test")
}

func TestTransformDocuments(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:  ".metadata.namespace",
						Value: `{{ .Release.Namespace }}`,
					},
				},
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
	require.NoError(t, err)

	require.Len(t, docs, 2)

	require.Equal(t, kindConfigMap, docs[0].Kind)
	require.Equal(t, "config1", docs[0].Name)
	require.Equal(t, kindDeployment, docs[1].Kind)
	require.Equal(t, "my-deploy", docs[1].Name)
}

func TestTransform_FilterIncludeKind(t *testing.T) {
	cfg := &config.Config{
		Filter: &config.Filter{
			Include: []config.Match{
				{Kinds: []string{kindDeployment}, Names: []string{"my-deploy"}},
			},
		},
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:  ".metadata.namespace",
						Value: `{{ .Release.Namespace }}`,
					},
				},
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
  namespace: default
spec:
  replicas: 1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-deploy
  namespace: default
spec:
  replicas: 1`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)

	// ConfigMap should pass through (kind not in include filter)
	require.Contains(t, output, kindConfigMap)
	// my-deploy should be included
	require.Contains(t, output, "my-deploy")
	// other-deploy should be excluded (kind is scoped, doesn't match name)
	require.NotContains(t, output, "other-deploy")
	// Rules should be applied to remaining docs
	count := strings.Count(output, `{{ .Release.Namespace }}`)
	require.Equal(t, 2, count, "Expected 2 namespace replacements (ConfigMap + my-deploy)\nOutput:\n%s", output)
}

func TestTransform_FilterExcludeKind(t *testing.T) {
	cfg := &config.Config{
		Filter: &config.Filter{
			Exclude: []config.Match{
				{Kinds: []string{kindConfigMap}},
			},
		},
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:  ".metadata.namespace",
						Value: `{{ .Release.Namespace }}`,
					},
				},
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
  namespace: default
spec:
  replicas: 1`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)

	require.NotContains(t, output, kindConfigMap)
	require.Contains(t, output, kindDeployment)
}

func TestTransformDocuments_FilterIncludeAndExclude(t *testing.T) {
	cfg := &config.Config{
		Filter: &config.Filter{
			Include: []config.Match{
				{Kinds: []string{kindDeployment}, Names: []string{"app-*"}},
			},
			Exclude: []config.Match{
				{Kinds: []string{kindDeployment}, Names: []string{"app-b"}},
			},
		},
	}

	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-a
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-b
---
apiVersion: v1
kind: Service
metadata:
  name: my-svc`

	transformer := New(cfg)
	docs, err := transformer.TransformDocuments([]byte(input))
	require.NoError(t, err)

	// app-a matches include, app-b matches exclude, Service passes through
	require.Len(t, docs, 2, "expected 2 documents (app-a + Service)")
	require.Equal(t, kindDeployment, docs[0].Kind)
	require.Equal(t, nameAppA, docs[0].Name)
	require.Equal(t, kindService, docs[1].Kind)
	require.Equal(t, "my-svc", docs[1].Name)
}

func TestTransform_DeleteAction(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:   ".metadata.annotations",
						Action: "delete",
					},
				},
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
	require.NoError(t, err)

	require.NotContains(t, output, "annotations:")
}

func TestTransform_AppendWith(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{kindDeployment}},
				Changes: []config.Change{
					{
						Path:       ".spec.template.spec.containers[0].env",
						AppendWith: `{{- include "myapp.extraEnv" . | nindent 12 }}`,
					},
				},
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
      containers:
        - name: my-container
          env:
            - name: FOO
              value: bar`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)

	// Should preserve existing env var
	require.Contains(t, output, "name: FOO")
	require.Contains(t, output, "value: bar")
	// Should contain appended content
	require.Contains(t, output, `{{- include "myapp.extraEnv" . | nindent 12 }}`)
}

func TestTransform_AppendWith_PreservesExistingElements(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{kindDeployment}},
				Changes: []config.Change{
					{
						Path:       ".spec.template.spec.containers[0].env",
						AppendWith: `{{- include "myapp.extraEnv" . | nindent 12 }}`,
					},
				},
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
      containers:
        - name: my-container
          env:
            - name: FOO
              value: bar
            - name: BAZ
              value: qux`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)

	// Both existing env vars should be preserved
	require.Contains(t, output, "name: FOO")
	require.Contains(t, output, "name: BAZ")
	// Appended content should follow
	require.Contains(t, output, `{{- include "myapp.extraEnv" . | nindent 12 }}`)
}

func TestTransform_AppendWith_NoMatch(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Match: &config.Match{Kinds: []string{kindDeployment}},
				Changes: []config.Change{
					{
						Path:       ".spec.template.spec.containers[0].env",
						AppendWith: `{{- include "myapp.extraEnv" . | nindent 12 }}`,
					},
				},
			},
		},
	}

	const input = `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)

	// Should not contain the appended content since it's a ConfigMap
	require.NotContains(t, output, "extraEnv")
}

func TestTransform_AddMapKey_NewAnnotation(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:  ".metadata.annotations.my-annotation",
						Value: "keep",
					},
				},
			},
		},
	}

	const input = `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)

	require.Contains(t, output, "annotations:")
	require.Contains(t, output, "my-annotation: keep")
}

func TestTransform_AddMapKey_ExistingAnnotations(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:  ".metadata.annotations.new-annotation",
						Value: `{{ .Values.myAnnotation }}`,
					},
				},
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  annotations:
    existing: value
data:
  key: value`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)

	require.Contains(t, output, "existing: value")
	require.Contains(t, output, "new-annotation: {{ .Values.myAnnotation }}")
}

func TestTransform_AddMapKey_QuotedDottedKey(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:  `.metadata.annotations["helm.sh/resource-policy"]`,
						Value: "keep",
					},
				},
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value`

	transformer := New(cfg)
	output, err := transformer.Transform([]byte(input))
	require.NoError(t, err)

	require.Contains(t, output, "annotations:")
	require.Contains(t, output, "helm.sh/resource-policy: keep")
}

func TestTransform_AppendWith_NonSequenceError(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Changes: []config.Change{
					{
						Path:       ".metadata.name",
						AppendWith: "some content",
					},
				},
			},
		},
	}

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value`

	transformer := New(cfg)
	_, err := transformer.Transform([]byte(input))
	require.Error(t, err)
	require.ErrorContains(t, err, "sequence")
}
