# helmtemplate-generator

A CLI tool that transforms Kubernetes YAML manifests into Helm templates using configuration-driven rules.

## Overview

`helmtemplate-generator` reads Kubernetes resource manifests (YAML) and applies transformation rules from a configuration file to produce Helm chart templates. It supports:

- Replacing field values with Helm template expressions
- Deleting fields
- Wrapping entire resources or field values with Helm conditionals
- Injecting raw Helm template blocks
- Splitting multi-document YAML into separate files per resource
- Generating chart support files (values.yaml, \_helpers.tpl) from templates

## Installation

### From source

Requires Go 1.25+.

```sh
go install github.com/davidebianchi/helmtemplate-generator@latest
```

### Build locally

```sh
git clone https://github.com/davidebianchi/helmtemplate-generator.git
cd helmtemplate-generator
make build
```

## Usage

```sh
helmtemplate-generator -c config.yaml [-i input.yaml] [-o output/]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to config file (required) |
| `--input` | `-i` | Input YAML file or directory (default: stdin) |
| `--output` | `-o` | Output file or directory (default: stdout) |
| `--template-dir` | | Directory for resolving template paths (defaults to config file directory) |
| `--chart-name` | | Chart name for template substitution |
| `--default-namespace` | | Default namespace for template substitution |
| `--chart-description` | | Chart description for template substitution |
| `--app-version` | | App version for template substitution |

### Examples

Transform a single file:

```sh
helmtemplate-generator -c config.yaml -i deployment.yaml -o templates/deployment.yaml
```

Transform a directory of manifests:

```sh
helmtemplate-generator -c config.yaml -i manifests/ -o templates/
```

Transform from stdin (pipe from kubectl):

```sh
kubectl get deployment my-app -o yaml | helmtemplate-generator -c config.yaml
```

Split multi-document YAML into separate files:

```sh
kubectl get all -o yaml | helmtemplate-generator -c config.yaml -o chart/
```

Generate chart files with metadata substitution:

```sh
helmtemplate-generator -c config.yaml -o chart/ --chart-name my-chart --default-namespace my-ns
```

## Configuration

The configuration file is YAML with the following top-level keys:

### `global`

Global rules applied to all resources before per-resource rules.

```yaml
global:
  delete:
    - .status
    - .metadata.uid
    - .metadata.resourceVersion
    - .metadata.creationTimestamp
```

### `rules`

A list of transformation rules. Each rule can optionally match specific resources and apply changes.

#### Matching resources

```yaml
rules:
  - match:
      kinds:
        - Deployment
        - StatefulSet
      names:                   # Wildcard name matching (matches ANY)
        - "my-*"
      labels:                  # All labels must match
        app: myapp
```

#### Setting values

```yaml
rules:
  - path: .metadata.namespace
    value: '{{ .Release.Namespace }}'

  - match:
      kinds:
        - Deployment
    changes:
      - path: .spec.replicas
        value: '{{ .Values.replicas | default 1 }}'
      - path: .spec.template.spec.containers[0].image
        value: '{{ .Values.image.repository }}:{{ .Values.image.tag }}'
```

#### Deleting fields

```yaml
rules:
  - path: .metadata.annotations
    action: delete
```

#### Wrapping resources with conditionals

```yaml
rules:
  - match:
      kinds:
        - CustomResourceDefinition
    wrap:
      before: '{{- if .Values.installCRDs }}'
      after: '{{- end }}'
```

#### Replacing fields with raw Helm blocks

```yaml
rules:
  - match:
      kinds:
        - Deployment
    path: .spec.template.spec.imagePullSecrets
    replaceWith: |
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

### `output`

Controls how multi-document output is organized.

```yaml
output:
  splitByDocument: true
  directoryRules:
    - match:
        kinds:
          - CustomResourceDefinition
      directory: crds
```

### `chartFiles`

Generate support files from templates with variable substitution.

```yaml
chartFiles:
  - template: templates/values.yaml.tpl
    output: values.yaml
  - template: templates/_helpers.tpl
    output: templates/_helpers.tpl
```

Template variables available: `CHART_NAME`, `DEFAULT_NAMESPACE`, `CHART_DESCRIPTION`, `APP_VERSION`.

## Path Syntax

Paths use a JSONPath-like dot notation:

| Syntax | Description |
|--------|-------------|
| `.metadata.name` | Access map keys |
| `.spec.containers[0].image` | Access array elements by index |

## Development

```sh
make test     # Run tests
make lint     # Run linter
make build    # Build binary
make fmt      # Format code
```

## License

[Add your license here]
