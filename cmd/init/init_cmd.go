package init

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed INIT.md
var initTemplate string

// Cmd represents the init command
var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize sanity in the current directory",
	Long: `Initialize sanity in the current directory by adding instructions to AGENTS.md.

This creates or updates AGENTS.md with a minimal snippet that helps AI agents
understand how to use sanity.`,
	RunE: runInit,
}

func runInit(_ *cobra.Command, _ []string) error {
	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository (no .git directory found)")
	}

	// Create/update AGENTS.md
	if err := writeAgentsFile("AGENTS.md"); err != nil {
		return err
	}

	return nil
}

func writeAgentsFile(filename string) error {
	_, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	_, err = os.Stat(filename)
	fileExists := !os.IsNotExist(err)

	if fileExists {
		// Append to existing file
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", filename, err)
		}
		defer f.Close()

		// Add newline before appending
		if _, err := f.WriteString("\n" + initTemplate); err != nil {
			return fmt.Errorf("failed to append to %s: %w", filename, err)
		}
	} else {
		// Create new file or overwrite
		if err := os.WriteFile(filename, []byte(initTemplate), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}
