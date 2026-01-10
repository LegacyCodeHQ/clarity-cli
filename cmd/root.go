package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sanity",
	Short: "Analyze and visualize dependency graphs in your codebase",
	Long: `Sanity is a CLI tool for analyzing and visualizing dependency graphs
in your codebase. It supports Dart and Go files, showing relationships
between project files while excluding external dependencies.

Use 'sanity --help' to see all available commands, or 'sanity <command> --help'
for detailed information about a specific command.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Register subcommands
	rootCmd.AddCommand(graphCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sanity.yaml)")
}
