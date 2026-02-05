package init

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const agentsSnippet = `## Dependency Visualization

This project uses **sanity** for dependency visualization and design analysis.
Run ` + "`sanity prime`" + ` for workflow context.

**Quick reference:**
- ` + "`sanity graph -u`" + `          - Visualize and open in browser
- ` + "`sanity graph -f mermaid`" + `  - Output mermaid diagram
- ` + "`sanity graph -c HEAD~3`" + `   - Graph files from recent commits

For full workflow details: ` + "`sanity prime`" + `
`

var (
	flagCopilot bool
	flagForce   bool
	flagQuiet   bool
)

// Cmd represents the init command
var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize sanity in the current directory",
	Long: `Initialize sanity in the current directory by adding instructions to AGENTS.md.

This creates or updates AGENTS.md with a minimal snippet that helps AI agents
understand how to use sanity for dependency visualization.

With --copilot: Also creates .github/copilot-instructions.md for GitHub Copilot.

With --force: Overwrites existing files instead of appending.`,
	RunE: runInit,
}

func init() {
	Cmd.Flags().BoolVar(&flagCopilot, "copilot", false, "Also create .github/copilot-instructions.md")
	Cmd.Flags().BoolVar(&flagForce, "force", false, "Overwrite existing files instead of appending")
	Cmd.Flags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress output")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository (no .git directory found)")
	}

	// Create/update AGENTS.md
	if err := writeAgentsFile("AGENTS.md"); err != nil {
		return err
	}

	// Optionally create .github/copilot-instructions.md
	if flagCopilot {
		if err := os.MkdirAll(".github", 0755); err != nil {
			return fmt.Errorf("failed to create .github directory: %w", err)
		}
		if err := writeAgentsFile(".github/copilot-instructions.md"); err != nil {
			return err
		}
	}

	if !flagQuiet {
		fmt.Println("Sanity initialized successfully!")
		fmt.Println("")
		fmt.Println("Files updated:")
		fmt.Println("  - AGENTS.md")
		if flagCopilot {
			fmt.Println("  - .github/copilot-instructions.md")
		}
		fmt.Println("")
		fmt.Println("What this command does:")
		fmt.Println("  - Adds a minimal sanity snippet to AGENTS.md")
		fmt.Println("  - (Optional) Adds the same snippet to .github/copilot-instructions.md")
		fmt.Println("")
		fmt.Println("How it relates to other commands:")
		fmt.Println("  - sanity onboard  -> shows the minimal snippet without changing files")
		fmt.Println("  - sanity prime    -> prints the full workflow context for this tool")
		fmt.Println("")
		fmt.Println("Quick start:")
		fmt.Println("  - Run 'sanity prime' after a new session or context reset")
		fmt.Println("  - Run 'sanity graph -u' to visualize dependencies in a browser")
		fmt.Println("  - Run 'sanity graph -f mermaid' to render in IDEs / applications that support mermaid")
		fmt.Println("  - Run 'sanity graph' to output Graphviz (dot) for tools that can render it")
		fmt.Println("")
		fmt.Println("Re-running init:")
		fmt.Println("  - Use --force to overwrite files instead of appending")
	}

	return nil
}

func writeAgentsFile(filename string) error {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	_, err = os.Stat(filename)
	fileExists := !os.IsNotExist(err)

	if fileExists && !flagForce {
		// Append to existing file
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", filename, err)
		}
		defer f.Close()

		// Add newlines before appending
		if _, err := f.WriteString("\n\n" + agentsSnippet); err != nil {
			return fmt.Errorf("failed to append to %s: %w", filename, err)
		}

		if !flagQuiet {
			fmt.Printf("Appended sanity instructions to %s\n", absPath)
		}
	} else {
		// Create new file or overwrite
		if err := os.WriteFile(filename, []byte(agentsSnippet), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}

		if !flagQuiet {
			if fileExists {
				fmt.Printf("Overwrote %s with sanity instructions\n", absPath)
			} else {
				fmt.Printf("Created %s with sanity instructions\n", absPath)
			}
		}
	}

	return nil
}
