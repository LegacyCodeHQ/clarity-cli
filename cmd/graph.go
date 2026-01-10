package cmd

import (
	"fmt"

	"sanity/parser"

	"github.com/spf13/cobra"
)

var outputFormat string

// graphCmd represents the graph command
var graphCmd = &cobra.Command{
	Use:   "graph [files...]",
	Short: "Generate dependency graph for project imports",
	Long: `Analyzes Dart files and generates a dependency graph showing relationships
between project files (excluding external package: and dart: imports).

Supports multiple output formats:
  - list: Simple text list (default)
  - json: JSON format
  - dot: Graphviz DOT format for visualization

Example usage:
  sanity graph file1.dart file2.dart file3.dart
  sanity graph lib/**/*.dart --format=json
  sanity graph lib/**/*.dart --format=dot > deps.dot`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Build the dependency graph
		graph, err := parser.BuildDependencyGraph(args)
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
}
