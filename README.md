# Sanity (experimental)

[![License](https://img.shields.io/github/license/LegacyCodeHQ/sanity)](LICENSE)
[![Release](https://img.shields.io/github/v/release/LegacyCodeHQ/sanity)](https://github.com/LegacyCodeHQ/sanity/releases)

Sanity helps you assess risk and confidently review AI-generated code before you commit it.

## Visuals

<p align="center">
  <img src="docs/images/go-example.png" alt="Go">
  <small>Fig. 1: Relationships between impacted files are shown and tests are highlighted in green.</small>
</p>

<p align="center">
  <img src="docs/images/kotlin-example.png" alt="Kotlin">
  <small>Fig. 2: New files are marked with a 🪴 emoji.</small>
</p>

## Supported Languages

- Dart
- Go
- Kotlin

## Quick Start

### Installation

Install on Linux and Mac using Homebrew:

```bash
brew install LegacyCodeHQ/tap/sanity
```

For other installation methods (pre-built binaries, build from source, Go install), see
the [Installation Guide](docs/usage/installation.md).

## Usage

### Commands

#### `sanity graph`

Generate dependency graphs for Dart and Go files.

**Flags**:

| Flag             | Description               | Notes          |
|------------------|---------------------------|----------------|
| `--repo`, `-r`   | Git repository path       | Default: "."   |
| `--commit`, `-c` | Git commit to analyze     |                |
| `--format`, `-f` | Output format (dot, json) | Default: "dot" |

**Examples**:

```bash
# Analyze uncommitted files in current repository (most common use case)
sanity graph

# Output dependency graph in JSON format
sanity graph --format=json

# Analyze files changed in a specific commit
sanity graph --commit 8d4f78

# Analyze uncommitted files in a different repository
sanity graph --repo /path/to/repo --commit HEAD~1

# Analyze specific files directly
sanity graph file1.dart file2.dart file3.dart
```

#### Help

- **List all commands**: `sanity --help`
- **Command-specific help**: `sanity <command> --help`
- **Help command alias**: `sanity help <command>`
