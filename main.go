package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidebianchi/helmtemplate-generator/config"
	"github.com/davidebianchi/helmtemplate-generator/transform"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		configPath       string
		templateDir      string
		inputPath        string
		outputPath       string
		chartName        string
		defaultNamespace string
		chartDescription string
		appVersion       string
	)

	flag.StringVar(&configPath, "c", "", "Path to config file (required)")
	flag.StringVar(&configPath, "config", "", "Path to config file (required)")
	flag.StringVar(&templateDir, "template-dir", "",
		"Directory for resolving template paths (defaults to config file directory)")
	flag.StringVar(&inputPath, "i", "", "Input YAML file or directory (default: stdin)")
	flag.StringVar(&inputPath, "input", "", "Input YAML file or directory (default: stdin)")
	flag.StringVar(&outputPath, "o", "", "Output file or directory (default: stdout)")
	flag.StringVar(&outputPath, "output", "", "Output file or directory (default: stdout)")
	flag.StringVar(&chartName, "chart-name", "", "Chart name for template substitution (replaces CHART_NAME)")
	flag.StringVar(&defaultNamespace, "default-namespace", "",
		"Default namespace for template substitution (replaces DEFAULT_NAMESPACE)")
	flag.StringVar(&chartDescription, "chart-description", "",
		"Chart description for template substitution (replaces CHART_DESCRIPTION)")
	flag.StringVar(&appVersion, "app-version", "", "App version for template substitution (replaces APP_VERSION)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `helmtemplate-generator - Transform Kubernetes manifests into Helm templates

Usage:
  helmtemplate-generator -c config.yaml [-i input.yaml] [-o output.yaml]
  cat manifest.yaml | helmtemplate-generator -c config.yaml > template.yaml

Options:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Transform single file
  helmgen -c config.yaml -i deployment.yaml -o templates/deployment.yaml

  # Transform directory
  helmgen -c config.yaml -i manifests/ -o templates/

  # Transform from stdin
  cat manifest.yaml | helmgen -c config.yaml

  # Transform from OLM bundle extraction
  kubectl get deployment -o yaml | helmgen -c config.yaml

  # Generate chart files (values.yaml, _helpers.tpl) from templates
  helmgen -c config.yaml -o chart/ --chart-name my-chart --default-namespace my-ns
`)
	}

	flag.Parse()

	if configPath == "" {
		return fmt.Errorf("config file is required (-c)")
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Generate chart files from templates if configured
	if len(cfg.ChartFiles) > 0 {
		// Use explicit template-dir if provided, otherwise use config file directory
		tplDir := templateDir
		if tplDir == "" {
			tplDir = filepath.Dir(configPath)
		}
		subs := templateSubstitutions{
			ChartName:        chartName,
			DefaultNamespace: defaultNamespace,
			ChartDescription: chartDescription,
			AppVersion:       appVersion,
		}
		if err := generateChartFiles(cfg, tplDir, outputPath, subs); err != nil {
			return fmt.Errorf("failed to generate chart files: %w", err)
		}
	}

	transformer := transform.New(cfg)

	// Handle input
	if inputPath == "" {
		// Read from stdin
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}

		// Check if we should split documents into separate files
		if transformer.ShouldSplitDocuments() && outputPath != "" {
			return transformAndSplit(transformer, input, outputPath)
		}

		output, err := transformer.Transform(input)
		if err != nil {
			return fmt.Errorf("failed to transform: %w", err)
		}

		return writeOutput(output, outputPath)
	}

	// Check if input is a directory
	info, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to stat input: %w", err)
	}

	if info.IsDir() {
		return transformDirectory(transformer, inputPath, outputPath)
	}

	// Single file
	input, err := os.ReadFile(filepath.Clean(inputPath))
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	output, err := transformer.Transform(input)
	if err != nil {
		return fmt.Errorf("failed to transform: %w", err)
	}

	return writeOutput(output, outputPath)
}

func transformDirectory(t *transform.Transformer, inputDir, outputDir string) error {
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Create output directory if needed
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Recursively process subdirectories
			subInput := filepath.Join(inputDir, entry.Name())
			subOutput := ""
			if outputDir != "" {
				subOutput = filepath.Join(outputDir, entry.Name())
			}
			if err := transformDirectory(t, subInput, subOutput); err != nil {
				return err
			}
			continue
		}

		// Only process YAML files
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		inputPath := filepath.Join(inputDir, name)
		input, err := os.ReadFile(filepath.Clean(inputPath))
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", inputPath, err)
		}

		output, err := t.Transform(input)
		if err != nil {
			return fmt.Errorf("failed to transform %s: %w", inputPath, err)
		}

		outputPath := ""
		if outputDir != "" {
			outputPath = filepath.Join(outputDir, name)
		}

		if err := writeOutput(output, outputPath); err != nil {
			return fmt.Errorf("failed to write output for %s: %w", inputPath, err)
		}
	}

	return nil
}

func transformAndSplit(t *transform.Transformer, input []byte, outputDir string) error {
	docs, err := t.TransformDocuments(input)
	if err != nil {
		return fmt.Errorf("failed to transform documents: %w", err)
	}

	// Templates go in the templates/ subdirectory
	templatesDir := filepath.Join(outputDir, "templates")

	// Create templates directory
	if err := os.MkdirAll(templatesDir, 0750); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	for _, doc := range docs {
		// Get output subdirectory based on config rules
		subDir := t.GetOutputDirectory(doc)
		fileName := t.GetOutputFileName(doc)

		var outputPath string
		if subDir != "" {
			outputPath = filepath.Join(templatesDir, subDir, fileName)
		} else {
			outputPath = filepath.Join(templatesDir, fileName)
		}

		if err := writeOutput(doc.Content, outputPath); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, err)
		}

		fmt.Fprintf(os.Stderr, "  Created: %s\n", outputPath)
	}

	fmt.Fprintf(os.Stderr, "\nProcessed %d resources\n", len(docs))
	return nil
}

func writeOutput(content, outputPath string) error {
	// Ensure content ends with newline
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if outputPath == "" {
		fmt.Print(content)
		return nil
	}

	// Create parent directories if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(outputPath, []byte(content), 0600)
}

// templateSubstitutions holds all template variable substitutions
type templateSubstitutions struct {
	ChartName        string
	DefaultNamespace string
	ChartDescription string
	AppVersion       string
}
