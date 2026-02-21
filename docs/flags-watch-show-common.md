# Clarity: Common and Unique Flags Between `watch` and `show`

This document is generated from Lens output of the command files:
- `cmd/show/show_cmd.go`
- `cmd/watch/watch_cmd.go`

## Common Flags

| Flag | `watch` description | `show` description |
|---|---|---|
| `--direction` | Graph direction (lr, rl, tb, bt) | Graph direction (lr, rl, tb, bt) |
| `--exclude` | Exclude specific files and/or directories (comma-separated) | Exclude specific files and/or directories from graph inputs (comma-separated) |
| `--exclude-ext` | Exclude files with these extensions (comma-separated, e.g. .go,.java) | Exclude files with these extensions (comma-separated, e.g. .go,.java) |
| `--include-ext` | Include only files with these extensions (comma-separated, e.g. .go,.java) | Include only files with these extensions (comma-separated, e.g. .go,.java) |
| `--input` | Watch specific files and/or directories (comma-separated) | Build graph from specific files and/or directories (comma-separated) |
| `--repo` | Git repository path (default: current directory) | Git repository path (default: current directory) |


## `watch`-Only Flags

| Flag | Description |
|---|---|
| `--port` | HTTP server port |


## `show`-Only Flags

| Flag | Description |
|---|---|
| `--allow-outside-repo` | Allow input paths outside the repo root |
| `--between` | Find all paths between specified files (comma-separated) |
| `--commit` | Git commit or range to analyze (e.g., f0459ec, HEAD~3, f0459ec...be3d11a) |
| `--file` | Show dependencies for a specific file |
| `--format` | fmt.Sprintf("Output format (%s)", formatters.SupportedFormats()) |
| `--level` | Depth level for dependencies (used with --file) |
| `--url` | Generate visualization URL (supported formats: dot, mermaid) |
