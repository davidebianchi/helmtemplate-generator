package transform

import (
	"testing"

	"github.com/davidebianchi/helmtemplate-generator/config"
	"github.com/stretchr/testify/require"
)

func parseMultiDoc(t *testing.T, input string) []*Document {
	t.Helper()
	docs, err := ParseDocuments([]byte(input))
	require.NoError(t, err)
	return docs
}

func TestFilterDocuments_NilFilter(t *testing.T) {
	docs := parseMultiDoc(t, "kind: Deployment\nmetadata:\n  name: test")
	result := FilterDocuments(docs, nil)
	require.Len(t, result, 1)
}

func TestFilterDocuments_EmptyDocs(t *testing.T) {
	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}},
		},
	}
	result := FilterDocuments(nil, filter)
	require.Len(t, result, 0)
}

func TestFilterDocuments_IncludeScopedByKind(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: my-app
---
kind: Deployment
metadata:
  name: other-deploy
---
kind: Service
metadata:
  name: my-svc
---
kind: ConfigMap
metadata:
  name: my-config`)

	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{"my-app"}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 3, "expected 3 documents (my-app + Service + ConfigMap)")

	// my-app Deployment should be included
	require.Equal(t, kindDeployment, result[0].GetKind())
	require.Equal(t, "my-app", result[0].GetName())
	// Service should pass through (kind not in filter)
	require.Equal(t, kindService, result[1].GetKind())
	// ConfigMap should pass through (kind not in filter)
	require.Equal(t, kindConfigMap, result[2].GetKind())
}

func TestFilterDocuments_IncludeByNameWildcard(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: my-app
---
kind: Deployment
metadata:
  name: other-deploy`)

	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{"my-*"}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 1)
	require.Equal(t, "my-app", result[0].GetName())
}

func TestFilterDocuments_IncludeByLabels(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: app1
  labels:
    app: frontend
---
kind: Deployment
metadata:
  name: app2
  labels:
    app: backend`)

	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}, Labels: map[string]string{"app": "frontend"}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 1)
	require.Equal(t, "app1", result[0].GetName())
}

func TestFilterDocuments_IncludeMultipleEntries(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: app-a
---
kind: Deployment
metadata:
  name: app-b
---
kind: Deployment
metadata:
  name: app-c`)

	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{nameAppA}},
			{Kinds: []string{kindDeployment}, Names: []string{"app-b"}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 2)
	require.Equal(t, nameAppA, result[0].GetName())
	require.Equal(t, "app-b", result[1].GetName())
}

func TestFilterDocuments_IncludeNoMatch(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: my-deploy
---
kind: Service
metadata:
  name: my-svc`)

	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{"nonexistent"}},
		},
	}

	result := FilterDocuments(docs, filter)
	// Deployment is scoped but none match; Service passes through
	require.Len(t, result, 1, "expected 1 document (Service)")
	require.Equal(t, kindService, result[0].GetKind())
}

func TestFilterDocuments_IncludeByMultipleNames(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: app-a
---
kind: Deployment
metadata:
  name: app-b
---
kind: Deployment
metadata:
  name: app-c`)

	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{"app-a", "app-b"}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 2)
	require.Equal(t, "app-a", result[0].GetName())
	require.Equal(t, "app-b", result[1].GetName())
}

func TestFilterDocuments_ExcludeByKind(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: my-deploy
---
kind: ConfigMap
metadata:
  name: my-config
---
kind: Service
metadata:
  name: my-svc`)

	filter := &config.Filter{
		Exclude: []config.Match{
			{Kinds: []string{kindConfigMap}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 2)
	require.Equal(t, kindDeployment, result[0].GetKind())
	require.Equal(t, kindService, result[1].GetKind())
}

func TestFilterDocuments_ExcludeByName(t *testing.T) {
	docs := parseMultiDoc(t, `kind: ConfigMap
metadata:
  name: kube-root-ca.crt
---
kind: ConfigMap
metadata:
  name: my-config`)

	filter := &config.Filter{
		Exclude: []config.Match{
			{Kinds: []string{kindConfigMap}, Names: []string{"kube-root-ca*"}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 1)
	require.Equal(t, "my-config", result[0].GetName())
}

func TestFilterDocuments_ExcludeNoneMatch(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: my-deploy
---
kind: Service
metadata:
  name: my-svc`)

	filter := &config.Filter{
		Exclude: []config.Match{
			{Kinds: []string{kindConfigMap}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 2)
}

func TestFilterDocuments_IncludeAndExclude(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: app-a
---
kind: Deployment
metadata:
  name: app-b
---
kind: Service
metadata:
  name: my-svc`)

	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{"app-*"}},
		},
		Exclude: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{"app-b"}},
		},
	}

	result := FilterDocuments(docs, filter)
	// app-a matches include, app-b matches exclude, Service passes through
	require.Len(t, result, 2, "expected 2 documents (app-a + Service)")
	require.Equal(t, nameAppA, result[0].GetName())
	require.Equal(t, kindService, result[1].GetKind())
}

func TestFilterDocuments_ExcludeWinsOverInclude(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: my-app`)

	filter := &config.Filter{
		Include: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{"my-app"}},
		},
		Exclude: []config.Match{
			{Kinds: []string{kindDeployment}, Names: []string{"my-app"}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 0, "expected 0 documents when both include and exclude match")
}

func TestFilterDocuments_MultipleExcludeEntries(t *testing.T) {
	docs := parseMultiDoc(t, `kind: Deployment
metadata:
  name: deploy
---
kind: ConfigMap
metadata:
  name: config
---
kind: Service
metadata:
  name: svc`)

	filter := &config.Filter{
		Exclude: []config.Match{
			{Kinds: []string{kindConfigMap}},
			{Kinds: []string{kindService}},
		},
	}

	result := FilterDocuments(docs, filter)
	require.Len(t, result, 1)
	require.Equal(t, kindDeployment, result[0].GetKind())
}
