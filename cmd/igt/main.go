package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mather/igt/internal/fetcher"
	"github.com/mather/igt/internal/merger"
	"github.com/mather/igt/internal/template"
	"github.com/mather/igt/internal/ui"
)

const version = "0.1.0"

var (
	outputPath = flag.String("o", ".gitignore", "output file path")
	output     = flag.String("output", ".gitignore", "output file path")
	refresh    = flag.Bool("r", false, "force refresh cache")
	refreshLong = flag.Bool("refresh", false, "force refresh cache")
	listMode   = flag.Bool("l", false, "list all templates")
	listLong   = flag.Bool("list", false, "list all templates")
	dryRun     = flag.Bool("n", false, "dry run mode (preview changes)")
	dryRunLong = flag.Bool("dry-run", false, "dry run mode (preview changes)")
	showHelp   = flag.Bool("h", false, "show help")
	helpLong   = flag.Bool("help", false, "show help")
	showVersion = flag.Bool("v", false, "show version")
	versionLong = flag.Bool("version", false, "show version")
)

func main() {
	flag.Parse()

	// Consolidate flags
	if *output != ".gitignore" {
		*outputPath = *output
	}
	if *refreshLong {
		*refresh = true
	}
	if *listLong {
		*listMode = true
	}
	if *dryRunLong {
		*dryRun = true
	}
	if *helpLong {
		*showHelp = true
	}
	if *versionLong {
		*showVersion = true
	}

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("igt version %s\n", version)
		os.Exit(0)
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Create fetcher
	f, err := fetcher.NewGitHubFetcher()
	if err != nil {
		return fmt.Errorf("failed to create fetcher: %w", err)
	}

	// Fetch templates
	fmt.Fprintln(os.Stderr, "Fetching templates...")
	templates, err := f.FetchTemplates(ctx, *refresh)
	if err != nil {
		return fmt.Errorf("failed to fetch templates: %w", err)
	}

	if len(templates) == 0 {
		return fmt.Errorf("no templates found")
	}

	// List mode
	if *listMode {
		printTemplateList(templates)
		return nil
	}

	// Get template names from arguments
	args := flag.Args()
	var selectedTemplates []template.Template

	if len(args) > 0 {
		// Non-interactive mode: select templates by name
		selectedTemplates, err = selectTemplatesByName(templates, args)
		if err != nil {
			return err
		}
	} else {
		// Interactive mode: show TUI
		fmt.Fprintln(os.Stderr, "Starting interactive selection...")
		selectedTemplates, err = ui.Run(templates)
		if err != nil {
			return fmt.Errorf("UI error: %w", err)
		}
	}

	if len(selectedTemplates) == 0 {
		fmt.Fprintln(os.Stderr, "No templates selected")
		return nil
	}

	// Create merger
	m := merger.NewMerger(*outputPath)

	// Dry run mode
	if *dryRun {
		preview, err := m.PreviewChanges(selectedTemplates)
		if err != nil {
			return fmt.Errorf("failed to generate preview: %w", err)
		}
		fmt.Print(preview)
		return nil
	}

	// Merge templates
	if err := m.MergeTemplates(selectedTemplates); err != nil {
		return fmt.Errorf("failed to merge templates: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully updated %s with %d template(s)\n",
		*outputPath, len(selectedTemplates))

	return nil
}

func printHelp() {
	fmt.Println("igt - Interactive gitignore CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  igt [flags] [<template>...]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -o, --output string   Output file path (default: .gitignore)")
	fmt.Println("  -r, --refresh         Force refresh cache")
	fmt.Println("  -l, --list            List all templates")
	fmt.Println("  -n, --dry-run         Dry run mode (preview changes)")
	fmt.Println("  -h, --help            Show help")
	fmt.Println("  -v, --version         Show version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  igt                   # Interactive mode")
	fmt.Println("  igt Go Node           # Non-interactive mode")
	fmt.Println("  igt -l | grep -i go   # List and search templates")
	fmt.Println("  igt -n Go             # Preview changes")
	fmt.Println("  igt -r                # Refresh cache and select")
}

func printTemplateList(templates []template.Template) {
	// Group by category
	byCategory := make(map[template.Category][]template.Template)
	for _, tmpl := range templates {
		byCategory[tmpl.Category] = append(byCategory[tmpl.Category], tmpl)
	}

	// Print each category
	categories := []template.Category{
		template.CategoryLanguage,
		template.CategoryGlobal,
		template.CategoryCommunity,
	}

	for _, cat := range categories {
		templates := byCategory[cat]
		if len(templates) == 0 {
			continue
		}

		fmt.Printf("=== %s ===\n", cat)
		for _, tmpl := range templates {
			fmt.Println(tmpl.GetDisplayName())
		}
		fmt.Println()
	}
}

func selectTemplatesByName(templates []template.Template, names []string) ([]template.Template, error) {
	// Build a map for case-insensitive lookup
	templateMap := make(map[string]template.Template)
	for _, tmpl := range templates {
		key := strings.ToLower(tmpl.GetDisplayName())
		templateMap[key] = tmpl
	}

	var selected []template.Template
	var notFound []string

	for _, name := range names {
		key := strings.ToLower(name)
		if tmpl, ok := templateMap[key]; ok {
			selected = append(selected, tmpl)
		} else {
			notFound = append(notFound, name)
		}
	}

	if len(notFound) > 0 {
		return nil, fmt.Errorf("templates not found: %s", strings.Join(notFound, ", "))
	}

	return selected, nil
}
