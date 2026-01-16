package graph

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters"
	"github.com/LegacyCodeHQ/sanity/git"
	"github.com/LegacyCodeHQ/sanity/parsers"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var outputFormat string
var repoPath string
var commitID string
var generateURL bool

// GraphCmd represents the graph command
var GraphCmd = &cobra.Command{
	Use:   "graph [files...]",
	Short: "Generate dependency graph for project imports",
	Long: `Analyzes files and generates a dependency graph showing relationships
between project files (excluding external package: and dart: imports).

All files are included in the graph. Dart files will show their dependencies,
while non-Dart files appear as standalone nodes with no connections.

Supports three modes:
  1. Explicit files: Analyze specific files
  2. Uncommitted files: Analyze all uncommitted files (default: current directory)
  3. Commit analysis: Analyze files changed in a commit (--commit)

Output formats:
  - dot: Graphviz DOT format for visualization (default)
  - json: JSON format
  - mermaid: Mermaid.js flowchart format

Example usage:
  sanity graph
  sanity graph --url
  sanity graph --commit 8d4f78
  sanity graph --commit 8d4f78 --format=json
  sanity graph --commit 8d4f78 --format=mermaid
  sanity graph --format=mermaid --url
  sanity graph --repo /path/to/repo --commit 8d4f78 --format=dot
  sanity graph file1.dart file2.dart file3.dart
  sanity graph --url --commit 8d4f78`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var filePaths []string
		var err error

		// If no explicit files provided and no repo path specified, default to current directory
		if len(args) == 0 && repoPath == "" {
			repoPath = "."
		}

		if repoPath != "" {
			// Ensure --repo and explicit files are not both provided
			if len(args) > 0 {
				return fmt.Errorf("cannot use --repo flag with explicit file arguments")
			}

			if commitID != "" {
				// Commit analysis mode
				filePaths, err = git.GetCommitDartFiles(repoPath, commitID)
				if err != nil {
					return fmt.Errorf("failed to get files from commit: %w", err)
				}

				if len(filePaths) == 0 {
					return fmt.Errorf("no files changed in commit %s", commitID)
				}
			} else {
				// Uncommitted files mode
				filePaths, err = git.GetUncommittedDartFiles(repoPath)
				if err != nil {
					return fmt.Errorf("failed to get uncommitted files: %w", err)
				}

				if len(filePaths) == 0 {
					fmt.Println("Working directory is clean (no uncommitted changes).")
					fmt.Println()
					fmt.Println("To visualize the most recent commit:")
					fmt.Println("  sanity graph -c HEAD")
					fmt.Println()
					fmt.Println("To visualize a specific commit:")
					fmt.Println("  sanity graph -c <commit-hash>")
					return nil
				}
			}
		} else {
			// Validate --commit cannot be used with explicit files
			if commitID != "" {
				return fmt.Errorf("--commit flag cannot be used with explicit file arguments")
			}

			// Explicit file mode
			filePaths = args
		}

		// Build the dependency graph
		// Pass repoPath and commitID if we're analyzing a commit (otherwise pass empty strings)
		graph, err := parsers.BuildDependencyGraph(filePaths, repoPath, commitID)
		if err != nil {
			return fmt.Errorf("failed to build dependency graph: %w", err)
		}

		// Get file statistics for DOT/Mermaid formats
		var fileStats map[string]git.FileStats
		if (outputFormat == "dot" || outputFormat == "mermaid") && repoPath != "" {
			if commitID != "" {
				// Get stats for committed changes
				fileStats, err = git.GetCommitFileStats(repoPath, commitID)
				if err != nil {
					// Don't fail if we can't get stats, just log and continue without them
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to get file statistics: %v\n", err)
				}
			} else {
				// Get stats for uncommitted changes
				fileStats, err = git.GetUncommittedFileStats(repoPath)
				if err != nil {
					// Don't fail if we can't get stats, just log and continue without them
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to get file statistics: %v\n", err)
				}
			}
		}

		// Build label with commit hash and dirty status for DOT/Mermaid formats
		var label string
		if outputFormat == "dot" || outputFormat == "mermaid" {
			// Determine the repo path to use (use current directory if not specified)
			labelRepoPath := repoPath
			if labelRepoPath == "" {
				labelRepoPath = "."
			}

			// Get repository root and extract directory name
			repoRoot, err := git.GetRepositoryRoot(labelRepoPath)
			if err == nil {
				projectName := filepath.Base(repoRoot)
				label = fmt.Sprintf("%s • ", projectName)
			}

			// Get current commit hash
			var commitHash string
			if commitID != "" {
				// When analyzing a specific commit, show that commit's hash
				commitHash, err = git.GetShortCommitHash(labelRepoPath, commitID)
			} else {
				// When analyzing uncommitted changes, show current HEAD
				commitHash, err = git.GetCurrentCommitHash(labelRepoPath)
			}
			if err == nil {
				label += commitHash

				// Only check for uncommitted changes when analyzing current state (not a specific commit)
				if commitID == "" {
					isDirty, err := git.HasUncommittedChanges(labelRepoPath)
					if err == nil && isDirty {
						label += "-dirty"
					}
				}

				// Add number of files changed
				fileCount := len(filePaths)
				if fileCount == 1 {
					label += fmt.Sprintf(" • %d file", fileCount)
				} else {
					label += fmt.Sprintf(" • %d files", fileCount)
				}
			}
		}

		// Output based on format
		var output string
		switch outputFormat {
		case "json":
			jsonData, err := formatters.ToJSON(graph)
			if err != nil {
				return fmt.Errorf("failed to generate JSON: %w", err)
			}
			output = string(jsonData)
			fmt.Println(output)

		case "dot":
			output = formatters.ToDOT(graph, label, fileStats)

			// Generate GraphvizOnline URL if requested
			if generateURL {
				vizURL := generateGraphvizOnlineURL(output)
				fmt.Println(vizURL)
			} else {
				fmt.Print(output)
			}

		case "mermaid":
			output = formatters.ToMermaid(graph, label, fileStats)

			// Generate Mermaid.live URL if requested
			if generateURL {
				vizURL := generateMermaidLiveURL(output)
				fmt.Println(vizURL)
			} else {
				fmt.Print(output)
			}

		default:
			return fmt.Errorf("unknown output format: %s (valid options: dot, json, mermaid)", outputFormat)
		}

		// Copy to clipboard if flag is enabled
		copyToClipboard, _ := cmd.Root().PersistentFlags().GetBool("clipboard")
		if copyToClipboard {
			if err := clipboard.WriteAll(output); err != nil {
				return fmt.Errorf("failed to copy to clipboard: %w", err)
			}
			fmt.Println("\n✅ Content copied to your clipboard.")
		}

		return nil
	},
}

// generateGraphvizOnlineURL creates a URL for GraphvizOnline with the DOT graph embedded
func generateGraphvizOnlineURL(dotGraph string) string {
	// URL encode the DOT graph for use in fragment (spaces as %20, not +)
	encoded := url.PathEscape(dotGraph)

	// Create the GraphvizOnline URL with the encoded graph
	return fmt.Sprintf("https://dreampuf.github.io/GraphvizOnline/?engine=dot#%s", encoded)
}

// generateMermaidLiveURL creates a URL for mermaid.live with the diagram embedded
func generateMermaidLiveURL(mermaidCode string) string {
	// Mermaid.live uses a JSON payload encoded in base64
	payload := map[string]interface{}{
		"code": mermaidCode,
		"mermaid": map[string]interface{}{
			"theme": "default",
		},
		"autoSync":      true,
		"updateDiagram": true,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		// Fallback: just return the code URL-encoded
		return fmt.Sprintf("https://mermaid.live/edit#%s", url.PathEscape(mermaidCode))
	}

	// Base64 encode the JSON payload
	encoded := base64.URLEncoding.EncodeToString(jsonBytes)

	return fmt.Sprintf("https://mermaid.live/edit#base64:%s", encoded)
}

func init() {
	// Add format flag
	GraphCmd.Flags().StringVarP(&outputFormat, "format", "f", "dot", "Output format (dot, json, mermaid)")
	// Add repo flag
	GraphCmd.Flags().StringVarP(&repoPath, "repo", "r", "", "Git repository path (default: current directory)")
	// Add commit flag
	GraphCmd.Flags().StringVarP(&commitID, "commit", "c", "", "Git commit to analyze")
	// Add URL flag
	GraphCmd.Flags().BoolVarP(&generateURL, "url", "u", false, "Generate GraphvizOnline URL for visualization")
}
