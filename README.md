# Sanity

Sanity is a CLI tool for analyzing and visualizing dependency graphs in your codebase, with support for Dart and Go files.

## Usage

### Commands

#### `sanity graph`

Generate dependency graphs for project files. Analyzes Dart and Go files to show import relationships.

**Flags**:
- `--format, -f`: Output format (list, json, dot) - default: "list"
- `--repo, -r`: Git repository path to analyze uncommitted files
- `--commit, -c`: Git commit to analyze (requires --repo)

**Examples**:
```bash
# Analyze uncommitted files in current repository (most common use case)
sanity graph --repo .

# Analyze files changed in a specific commit
sanity graph --repo . --commit 8d4f78

# Output in JSON format
sanity graph --repo . --format=json

# Analyze uncommitted files in a different repository
sanity graph --repo /path/to/repo

# Output in Graphviz DOT format for visualization
sanity graph --repo /path/to/repo --commit 8d4f78 --format=dot

# Analyze specific files directly
sanity graph file1.dart file2.dart file3.dart
```

#### Help

- **List all commands**: `sanity --help`
- **Command-specific help**: `sanity <command> --help`
- **Help command alias**: `sanity help <command>`

---

## Development

### Testing and Code Coverage

This project uses Go's built-in testing framework with code coverage support.

**Available make targets**:

- `make test` - Run all tests
- `make test-coverage` - Run tests and show coverage percentage
- `make coverage` - Generate coverage profile (coverage.out)
- `make coverage-html` - Generate HTML coverage report (coverage.html)
- `make clean` - Remove coverage files and binary
