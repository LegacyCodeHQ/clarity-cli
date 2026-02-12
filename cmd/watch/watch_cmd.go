package watch

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/LegacyCodeHQ/clarity/cmd/show/formatters"
	"github.com/LegacyCodeHQ/clarity/depgraph"
	"github.com/LegacyCodeHQ/clarity/depgraph/registry"
	"github.com/LegacyCodeHQ/clarity/vcs"
	"github.com/spf13/cobra"
)

type watchOptions struct {
	repoPath   string
	port       int
	includeExt string
	excludeExt string
	includes   []string
	excludes   []string
}

// Cmd represents the watch command.
var Cmd = NewCommand()

// NewCommand returns a new watch command instance.
func NewCommand() *cobra.Command {
	opts := &watchOptions{
		port: 4900,
	}

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch for file changes and serve a live dependency graph",
		Long:  `Watch a project directory for file changes, rebuild the dependency graph, and serve a live-updating visualization at localhost.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repoPath, "repo", "r", "", "Git repository path (default: current directory)")
	cmd.Flags().IntVarP(&opts.port, "port", "P", opts.port, "HTTP server port")
	cmd.Flags().StringSliceVarP(&opts.includes, "input", "i", nil, "Watch specific files and/or directories (comma-separated)")
	cmd.Flags().StringSliceVar(&opts.excludes, "exclude", nil, "Exclude specific files and/or directories (comma-separated)")
	cmd.Flags().StringVar(&opts.includeExt, "include-ext", "", "Include only files with these extensions (comma-separated, e.g. .go,.java)")
	cmd.Flags().StringVar(&opts.excludeExt, "exclude-ext", "", "Exclude files with these extensions (comma-separated, e.g. .go,.java)")

	return cmd
}

func runWatch(cmd *cobra.Command, opts *watchOptions) error {
	repoPath := opts.repoPath
	if repoPath == "" {
		repoPath = "."
	}

	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve repo path: %w", err)
	}
	repoPath = absRepoPath

	dot, err := buildDOTGraph(repoPath, opts)
	if err != nil {
		return fmt.Errorf("initial graph build failed: %w", err)
	}

	b := newBroker()
	srv := newServer(b, opts.port)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", opts.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", opts.port, err)
	}

	go srv.Serve(ln)

	b.publish(dot)

	fmt.Fprintf(cmd.OutOrStdout(), "Watching %s\n", repoPath)
	fmt.Fprintf(cmd.OutOrStdout(), "Serving at http://localhost:%d\n", opts.port)
	fmt.Fprintf(cmd.OutOrStdout(), "Press Ctrl+C to stop\n")

	err = watchAndRebuild(ctx, repoPath, opts, b)

	srv.Close()
	return err
}

func buildDOTGraph(repoPath string, opts *watchOptions) (string, error) {
	filePaths, err := collectWatchFiles(repoPath, opts)
	if err != nil {
		return "", err
	}

	if len(filePaths) == 0 {
		return "", fmt.Errorf("no supported files found in %s", repoPath)
	}

	contentReader := vcs.FilesystemContentReader()

	graph, err := depgraph.BuildDependencyGraph(filePaths, contentReader)
	if err != nil {
		return "", fmt.Errorf("failed to build dependency graph: %w", err)
	}

	fileGraph, err := depgraph.NewFileDependencyGraph(graph, nil, contentReader)
	if err != nil {
		return "", fmt.Errorf("failed to build file graph metadata: %w", err)
	}

	formatter, err := formatters.NewFormatter("dot")
	if err != nil {
		return "", err
	}

	renderOpts := formatters.RenderOptions{
		Label: "clarity watch",
	}

	return formatter.Format(fileGraph, renderOpts)
}

func collectWatchFiles(repoPath string, opts *watchOptions) ([]string, error) {
	var roots []string
	if len(opts.includes) > 0 {
		for _, include := range opts.includes {
			absInclude, err := filepath.Abs(include)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve input path %q: %w", include, err)
			}
			roots = append(roots, absInclude)
		}
	} else {
		roots = []string{repoPath}
	}

	var filePaths []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			return nil, fmt.Errorf("failed to access %s: %w", root, err)
		}

		if info.IsDir() {
			err := filepath.Walk(root, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if fi.IsDir() {
					if skippedDirs[fi.Name()] {
						return filepath.SkipDir
					}
					return nil
				}
				ext := filepath.Ext(path)
				if registry.IsSupportedLanguageExtension(ext) {
					filePaths = append(filePaths, path)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory %s: %w", root, err)
			}
		} else {
			filePaths = append(filePaths, root)
		}
	}

	filePaths, err := applyWatchExtensionFilters(opts, filePaths)
	if err != nil {
		return nil, err
	}

	if len(opts.excludes) > 0 {
		filePaths, err = applyWatchExcludeFilter(opts, filePaths)
		if err != nil {
			return nil, err
		}
	}

	return filePaths, nil
}

func applyWatchExtensionFilters(opts *watchOptions, filePaths []string) ([]string, error) {
	if opts.includeExt != "" {
		exts := parseExtensions(opts.includeExt)
		filtered := make([]string, 0, len(filePaths))
		for _, fp := range filePaths {
			if exts[strings.ToLower(filepath.Ext(fp))] {
				filtered = append(filtered, fp)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("no files remain after applying --include-ext %q", opts.includeExt)
		}
		filePaths = filtered
	}

	if opts.excludeExt != "" {
		exts := parseExtensions(opts.excludeExt)
		filtered := make([]string, 0, len(filePaths))
		for _, fp := range filePaths {
			if !exts[strings.ToLower(filepath.Ext(fp))] {
				filtered = append(filtered, fp)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("no files remain after applying --exclude-ext %q", opts.excludeExt)
		}
		filePaths = filtered
	}

	return filePaths, nil
}

func applyWatchExcludeFilter(opts *watchOptions, filePaths []string) ([]string, error) {
	excludePaths := make([]string, 0, len(opts.excludes))
	for _, exclude := range opts.excludes {
		absExclude, err := filepath.Abs(exclude)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve exclude path %q: %w", exclude, err)
		}
		excludePaths = append(excludePaths, absExclude)
	}

	filtered := make([]string, 0, len(filePaths))
	for _, fp := range filePaths {
		excluded := false
		for _, ep := range excludePaths {
			if fp == ep || strings.HasPrefix(fp, ep+string(filepath.Separator)) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, fp)
		}
	}

	return filtered, nil
}

func parseExtensions(raw string) map[string]bool {
	exts := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		ext := strings.TrimSpace(part)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		exts[strings.ToLower(ext)] = true
	}
	return exts
}
