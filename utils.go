package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidebianchi/helmtemplate-generator/config"
)

func generateChartFiles(cfg *config.Config, configDir, outputDir string, subs templateSubstitutions) error {
	for _, cf := range cfg.ChartFiles {
		// Resolve template path relative to config file
		templatePath := cf.Template
		if !filepath.IsAbs(templatePath) {
			templatePath = filepath.Join(configDir, templatePath)
		}

		// Read template file
		content, err := os.ReadFile(filepath.Clean(templatePath))
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", cf.Template, err)
		}

		// Apply substitutions
		output := string(content)
		if subs.ChartName != "" {
			output = strings.ReplaceAll(output, "CHART_NAME", subs.ChartName)
		}
		if subs.DefaultNamespace != "" {
			output = strings.ReplaceAll(output, "DEFAULT_NAMESPACE", subs.DefaultNamespace)
		}
		if subs.ChartDescription != "" {
			output = strings.ReplaceAll(output, "CHART_DESCRIPTION", subs.ChartDescription)
		}
		if subs.AppVersion != "" {
			output = strings.ReplaceAll(output, "APP_VERSION", subs.AppVersion)
		}

		// Resolve output path relative to output directory
		outputPath := cf.Output
		if outputDir != "" && !filepath.IsAbs(outputPath) {
			outputPath = filepath.Join(outputDir, outputPath)
		}

		if err := writeOutput(output, outputPath); err != nil {
			return fmt.Errorf("failed to write %s: %w", cf.Output, err)
		}

		fmt.Fprintf(os.Stderr, "  Generated: %s\n", outputPath)
	}

	return nil
}
