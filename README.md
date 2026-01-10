# Sanity

Sanity is a CLI tool for analyzing and visualizing dependency graphs in your codebase, with support for Dart and Go files.

## Commands

### `sanity graph`

Generate dependency graphs for project files. Analyzes Dart and Go files to show import relationships.

**Flags**:
- `--format, -f`: Output format (list, json, dot) - default: "list"
- `--repo, -r`: Git repository path to analyze uncommitted files
- `--commit, -c`: Git commit to analyze (requires --repo)

**Examples**:
```bash
# Analyze specific files
sanity graph file1.dart file2.dart file3.dart

# Analyze uncommitted files in current repository
sanity graph --repo .

# Analyze files changed in a specific commit
sanity graph --repo . --commit abc123

# Output in JSON format
sanity graph --repo . --commit HEAD~1 --format=json

# Output in Graphviz DOT format for visualization
sanity graph --repo /path/to/repo --commit main --format=dot
```

## Getting Help

- **List all commands**: `sanity --help`
- **Command-specific help**: `sanity <command> --help`
- **Help command alias**: `sanity help <command>`
