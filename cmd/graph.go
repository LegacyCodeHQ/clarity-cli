package cmd

import (
	"fmt"

	"sanity/git"
	"sanity/parser"

	"github.com/spf13/cobra"
)

var outputFormat string
var repoPath string

// graphCmd represents the graph command
var graphCmd = &cobra.Command{
	Use:   "graph [files...]",
	Short: "Generate dependency graph for project imports",
	Long: `Analyzes Dart files and generates a dependency graph showing relationships
between project files (excluding external package: and dart: imports).

Supports two modes:
  1. Explicit files: Analyze specific files
  2. Git mode: Analyze all uncommitted .dart files in a repository

Output formats:
  - list: Simple text list (default)
  - json: JSON format
  - dot: Graphviz DOT format for visualization

Example usage:
  sanity graph file1.dart file2.dart file3.dart
  sanity graph --repo .
  sanity graph --repo /path/to/repo --format=json
  sanity graph lib/**/*.dart --format=dot > deps.dot`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var filePaths []string
		var err error

		if repoPath != "" {
			// Ensure --repo and explicit files are not both provided
			if len(args) > 0 {
				return fmt.Errorf("cannot use --repo flag with explicit file arguments")
			}

			// Git discovery mode
			filePaths, err = git.GetUncommittedDartFiles(repoPath)
			if err != nil {
				return fmt.Errorf("failed to get uncommitted files: %w", err)
			}

			if len(filePaths) == 0 {
				return fmt.Errorf("no uncommitted Dart files found in repository")
			}
		} else {
			// Explicit file mode
			if len(args) == 0 {
				return fmt.Errorf("no files specified (use --repo or provide file paths)")
			}
			filePaths = args
		}

		// Build the dependency graph
		graph, err := parser.BuildDependencyGraph(filePaths)
		if err != nil {
			return fmt.Errorf("failed to build dependency graph: %w", err)
		}

		// Output based on format
		switch outputFormat {
		case "json":
			jsonData, err := graph.ToJSON()
			if err != nil {
				return fmt.Errorf("failed to generate JSON: %w", err)
			}
			fmt.Println(string(jsonData))

		case "dot":
			fmt.Print(graph.ToDOT())

		case "list":
			fmt.Print(graph.ToList())

		default:
			return fmt.Errorf("unknown output format: %s (valid options: list, json, dot)", outputFormat)
		}

		return nil
	},
}

func init() {
	// Add format flag
	graphCmd.Flags().StringVarP(&outputFormat, "format", "f", "list", "Output format (list, json, dot)")
	// Add repo flag
	graphCmd.Flags().StringVarP(&repoPath, "repo", "r", "", "Git repository path to analyze uncommitted files")
}
