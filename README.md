# helmtemplate-generator

A CLI tool that transforms Kubernetes YAML manifests into Helm templates using configuration-driven rules.

## Overview

`helmtemplate-generator` reads Kubernetes resource manifests (YAML) and applies transformation rules from a configuration file to produce Helm chart templates. It supports:

- Replacing field values with Helm template expressions
- Appending raw content to existing YAML arrays
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
      excludeKinds:            # Exclude specific kinds (takes precedence over kinds)
        - Job
        - CronJob
      names:                   # Wildcard name matching (matches ANY)
        - "my-*"
      labels:                  # All labels must match
        app: myapp
```

#### Setting values

Replace a field's value with a Helm template expression.

Config:
```yaml
rules:
  - path: .metadata.namespace
    value: '{{ .Release.Namespace }}'
```

<details>
<summary>Before / After</summary>

Before:
```yaml
metadata:
  name: my-config
  namespace: default
```

After:
```yaml
metadata:
  name: my-config
  namespace: {{ .Release.Namespace }}
```
</details>

You can group multiple changes in a single rule:

```yaml
rules:
  - match:
      kinds:
        - Deployment
    changes:
      - path: .spec.replicas
        value: '{{ .Values.replicas | default 1 }}'
      - path: .spec.template.spec.containers[0].image
        value: '{{ .Values.image.repository }}:{{ .Values.image.tag }}'
```

#### Adding map keys

Set a value at a path that doesn't exist yet. Intermediate mapping nodes are created automatically. This is useful for adding annotations, labels, or any new map entries.

Config:
```yaml
rules:
  - path: .metadata.annotations.my-annotation
    value: '{{ .Values.myAnnotation }}'
```

<details>
<summary>Before / After</summary>

Before:
```yaml
metadata:
  name: my-config
```

After:
```yaml
metadata:
  name: my-config
  annotations:
    my-annotation: {{ .Values.myAnnotation }}
```
</details>

For keys containing dots (e.g., Kubernetes annotations), use the quoted bracket syntax `["key"]`:

Config:
```yaml
rules:
  - path: '.metadata.annotations["helm.sh/resource-policy"]'
    value: keep
```

<details>
<summary>Before / After</summary>

Before:
```yaml
metadata:
  name: my-config
```

After:
```yaml
metadata:
  name: my-config
  annotations:
    helm.sh/resource-policy: keep
```
</details>

If the map already exists, the new key is appended:

Config:
```yaml
rules:
  - path: .metadata.annotations.new-key
    value: new-value
```

<details>
<summary>Before / After</summary>

Before:
```yaml
metadata:
  name: my-config
  annotations:
    existing: value
```

After:
```yaml
metadata:
  name: my-config
  annotations:
    existing: value
    new-key: new-value
```
</details>

#### Deleting fields

Config:
```yaml
rules:
  - path: .metadata.annotations
    action: delete
```

<details>
<summary>Before / After</summary>

Before:
```yaml
metadata:
  name: my-config
  annotations:
    key: value
data:
  foo: bar
```

After:
```yaml
metadata:
  name: my-config
data:
  foo: bar
```
</details>

#### Wrapping resources with conditionals

Wrap an entire resource with Helm conditionals.

Config:
```yaml
rules:
  - match:
      kinds:
        - CustomResourceDefinition
    wrap:
      before: '{{- if .Values.installCRDs }}'
      after: '{{- end }}'
```

<details>
<summary>Before / After</summary>

Before:
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: my-crd
```

After:
```yaml
{{- if .Values.installCRDs }}
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: my-crd
{{- end }}
```
</details>

#### Replacing fields with raw Helm blocks

Replace a field and its value with arbitrary Helm template content.

Config:
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

<details>
<summary>Before / After</summary>

Before:
```yaml
spec:
  template:
    spec:
      imagePullSecrets:
        - name: my-secret
      containers:
        - name: app
```

After:
```yaml
spec:
  template:
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      containers:
        - name: app
```
</details>

#### Appending to arrays

Append raw content after the last element of an existing YAML array. Unlike `replaceWith`, this preserves all existing array elements.

Config:
```yaml
rules:
  - match:
      kinds:
        - Deployment
    path: .spec.template.spec.containers[0].env
    appendWith: |
      {{- include "mychart.extraEnv" . | nindent 12 }}
```

<details>
<summary>Before / After</summary>

Before:
```yaml
containers:
  - name: app
    env:
      - name: FOO
        value: bar
```

After:
```yaml
containers:
  - name: app
    env:
      - name: FOO
        value: bar
      {{- include "mychart.extraEnv" . | nindent 12 }}
```
</details>

#### Filtering array elements by key

Update a specific element in an array by matching a key-value pair, without needing to know its index. This is useful for targeting specific environment variables, containers, volumes, etc.

Config:
```yaml
rules:
  - match:
      kinds:
        - Deployment
    path: .spec.template.spec.containers[0].env[name=DATABASE_URL].value
    value: '{{ .Values.databaseURL }}'
```

<details>
<summary>Before / After</summary>

Before:
```yaml
containers:
  - name: app
    env:
      - name: FOO
        value: bar
      - name: DATABASE_URL
        value: postgres://localhost
```

After:
```yaml
containers:
  - name: app
    env:
      - name: FOO
        value: bar
      - name: DATABASE_URL
        value: {{ .Values.databaseURL }}
```
</details>

You can also delete a specific array element by filter:

```yaml
rules:
  - match:
      kinds:
        - Deployment
    path: .spec.template.spec.containers[0].env[name=TO_REMOVE]
    action: delete
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
| `.metadata.annotations["helm.sh/resource-policy"]` | Access keys containing dots |
| `.spec.containers[0].env[name=FOO].value` | Filter array elements by key=value |

## Development

```sh
make test     # Run tests
make lint     # Run linter
make build    # Build binary
make fmt      # Format code
```

## License

[Add your license here]
