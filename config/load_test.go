package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `rules:
  - path: .metadata.namespace
    value: '{{ .Release.Namespace }}'
`
	err := os.WriteFile(cfgPath, []byte(content), 0600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	require.Len(t, cfg.Rules, 1)
	require.Equal(t, ".metadata.namespace", cfg.Rules[0].Path)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	require.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.yaml")
	err := os.WriteFile(cfgPath, []byte(":\n  :\n    - [invalid"), 0600)
	require.NoError(t, err)

	_, err = Load(cfgPath)
	require.Error(t, err)
}

func TestLoad_Validation_UnknownAction(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `rules:
  - path: .metadata.name
    action: patch
`
	err := os.WriteFile(cfgPath, []byte(content), 0600)
	require.NoError(t, err)

	_, err = Load(cfgPath)
	require.Error(t, err)
	require.ErrorContains(t, err, "unknown action")
}

func TestLoad_ValidConfigWithFilterInclude(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `filter:
  include:
    - kinds:
        - Deployment
      names:
        - "my-app"
rules:
  - path: .metadata.namespace
    value: '{{ .Release.Namespace }}'
`
	err := os.WriteFile(cfgPath, []byte(content), 0600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	require.NotNil(t, cfg.Filter)
	require.Len(t, cfg.Filter.Include, 1)
}

func TestLoad_ValidConfigWithFilterExclude(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `filter:
  exclude:
    - kinds:
        - ConfigMap
      names:
        - "kube-root-ca*"
rules:
  - path: .metadata.namespace
    value: '{{ .Release.Namespace }}'
`
	err := os.WriteFile(cfgPath, []byte(content), 0600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	require.NotNil(t, cfg.Filter)
	require.Len(t, cfg.Filter.Exclude, 1)
}

func TestLoad_Validation_EmptyFilter(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `filter: {}
rules:
  - path: .metadata.namespace
    value: '{{ .Release.Namespace }}'
`
	err := os.WriteFile(cfgPath, []byte(content), 0600)
	require.NoError(t, err)

	_, err = Load(cfgPath)
	require.Error(t, err)
	require.ErrorContains(t, err, "filter")
}

func TestLoad_Validation_RuleWithoutPathOrChanges(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `rules:
  - match:
      kinds:
        - Deployment
`
	err := os.WriteFile(cfgPath, []byte(content), 0600)
	require.NoError(t, err)

	_, err = Load(cfgPath)
	require.Error(t, err)
}
