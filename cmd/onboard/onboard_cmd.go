package onboard

import (
	"fmt"

	"github.com/spf13/cobra"
)

// OnboardCmd represents the onboard command
var OnboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Show how to use sanity with your project",
	Long: `Show how to use sanity with your project.

Provides guidance on integrating sanity into your workflow and
a minimal snippet to add to AGENTS.md for AI assistant awareness.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(onboardingContent)
	},
}

const onboardingContent = `Sanity Onboarding

Add this minimal snippet to AGENTS.md (or create it):

--- BEGIN AGENTS.MD CONTENT ---
## Code Analysis

This project uses **sanity** for dependency graph analysis.
It helps audit AI-generated code and understand file relationships.

**Quick reference:**
- ` + "`sanity graph`" + `              - Graph uncommitted changes
- ` + "`sanity graph -c HEAD~3`" + `   - Graph files from recent commits
- ` + "`sanity graph -i ./src`" + `    - Graph specific files/directories
- ` + "`sanity graph -p ./main.go`" + ` - Show dependencies of a file
- ` + "`sanity graph -u`" + `          - Generate visualization URL

**Supported languages:** Dart, Go, Kotlin, TypeScript

For full options: ` + "`sanity graph --help`" + `
--- END AGENTS.MD CONTENT ---

For GitHub Copilot users:
Add the same content to .github/copilot-instructions.md

How it works:
   - sanity graph analyzes file imports to build dependency graphs
   - Output formats: DOT (default), Mermaid
   - Use -u flag to get a visualization URL for quick sharing
   - Review files right-to-left in the graph (dependencies first)

Common workflows:
   - Before code review: sanity graph -c <commit> -u
   - Understanding a file: sanity graph -p ./path/to/file.go -l 2
   - Finding connections: sanity graph -w ./file1.go,./file2.go`
