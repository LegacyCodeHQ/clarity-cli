Sanity Onboarding

Add this minimal snippet to AGENTS.md (or create it):

--- BEGIN AGENTS.MD CONTENT ---
---
## Sanity

This project uses `sanity` to audit AI-generated code, visualize software design, and help developers understand unfamiliar codebases.

### When to Use Sanity

Agents should use **sanity** in these scenarios:

1. **After code generation/modification** - Audit AI-generated changes
2. **When analyzing software design/architecture** - Examine and understand software design
3. **When fixing design issues** - Iteratively refactor to address design concerns
4. **Upon user request** - When explicitly asked to visualize dependencies

### When NOT to Run Sanity

- For trivial changes (typo fixes, comments)
- When user explicitly requests no visualization

## Quick Start for Agents

### Most common workflow:

1. Make code changes
2. Run: `open $(sanity graph -u)` (or platform equivalent)
3. Review visualization with user
4. Iterate if needed

### Choosing the Right Output Format

#### Use URL format (`-u`) when

- Agent has browser access
- Working in terminal/shell environment
- Need to share visualization with user

#### For shell/terminal agents

```bash
open $(sanity graph -u)      # macOS
start $(sanity graph -u)     # Windows
xdg-open $(sanity graph -u)  # Linux
```

This generates a visualization URL and opens it directly in the default browser.

#### Use Mermaid format (`-f mermaid`) when

- IDE/editor supports mermaid rendering
- Want inline visualization
- Working in environments with mermaid support

#### For IDE/desktop agents with mermaid rendering support:

```bash
sanity graph -f mermaid
```

Render the mermaid diagram output directly within the IDE or desktop application interface.

#### Platform Notes:

- **Windows**: Use `start` instead of `open`
- **Linux**: Use `xdg-open` instead of `open`
- **WSL**: May need `wslview` for browser opening

## Sanity Use Cases

### Auditing AI-Generated Code

After generating or modifying code, run `sanity graph` to visualize the relationships between changed files. This helps:

- Review the scope, impact, and blast radius of changes
- Identify unintended dependencies
- Verify that modifications follow the project's architectural patterns
- Catch potential issues before committing code

**Example Workflow:**

```bash
open $(sanity graph -u)
```

**Quick Reference:**

- `sanity graph`             - Output Graphviz (dot) format
- `sanity graph -u`          - Generate URL for online Graphviz viewer
- `sanity graph -f mermaid`  - Output mermaid diagram for IDE/desktop rendering
- `sanity graph -c HEAD~3`   - Graph files from recent commits

### Examining Software Design

Use `sanity graph` to understand and analyze your codebase architecture. This is useful for:

- Understanding how specific files or modules interact with each other
- Identifying coupling and dependency patterns
- Spotting cyclic dependencies
- Finding dependency paths between components
- Analyzing the impact of potential refactoring changes
- Onboarding to unfamiliar codebases

**Quick Reference:**

- `sanity graph -i ./src/auth,./src/api`         - Graph specific files/directories
- `sanity graph -w ./src/api.go,./src/db.go`     - Find all dependency paths between two files (**Why?** Understand if
  api.go depends on db.go directly or transitively)
- `sanity graph -c d2c2965`                      - View design changes in a single commit
- `sanity graph -c d2c2965...0de124f`            - View design changes across a range of commits
- `sanity graph -p ./src/core/engine.go`         - Show outgoing dependencies for a specific file (1 level) (**Why?**
  See what this file imports/uses)
- `sanity graph -p ./src/core/engine.go -l 3`    - Show dependencies up to 3 levels deep

**Note:** For all options: `sanity graph --help`

### Fixing Design Issues

Use `sanity` as a reliable design feedback tool to iteratively refactor code and address architectural concerns. This is an iterative process where the agent makes changes and uses `sanity graph` to verify improvements.

**Common Design Issues to Address:**

- Breaking cyclic dependencies
- Reducing coupling between components
- Introducing abstraction layers (interfaces, types)
- Enforcing dependency direction
- Separating concerns across module boundaries

**Workflow for Design Fixes:**

#### When user requests fixes for specific files/directories (`-i` flag):

```bash
# 1. Visualize the current design
sanity graph -i ./src/auth,./src/api

# 2. Identify the design issue (e.g., cyclic dependency, tight coupling)

# 3. Ask user for their opinion on the concerns

# 4. Make the requested changes (introduce types, break dependencies, etc.)

# 5. Verify the fix
sanity graph -i ./src/auth,./src/api

# 6. Repeat steps 4-5 until design concerns are addressed
```

#### When user requests to analyze dependency paths (`-w` flag):

```bash
# 1. Check if problematic dependency exists
sanity graph -w ./src/ui/handler.go,./src/db/repo.go

# 2. Ask user for their opinion about concerns with the design

# 3. If user wants to make changes, refactor (e.g., introduce interface, move code, add indirection layer)

# 4. Verify if the desired changes were made
sanity graph -w ./src/ui/handler.go,./src/db/repo.go
```

**Iterative Refinement Process:**

1. **Visualize** - Use appropriate flags to see current design
2. **Identify** - Spot the architectural issue (cyclic deps, wrong direction, tight coupling)
3. **Refactor** - Make targeted changes:

- Introduce interfaces/abstractions
- Extract shared types
- Move code to appropriate modules
- Break direct dependencies
- Apply dependency inversion

4. **Verify** - Run `sanity graph` again with same flags
5. **Repeat** - Continue until design concerns are resolved

**Key Principle:** Always use `sanity graph` with the same flags after making changes to verify that the design issue has been properly addressed. Continue iterating until the visualization confirms the desired architecture.

## Troubleshooting

If `sanity graph` fails:

- Ensure you're in a git repository root
- Verify the tool is installed: `sanity --version`
- Check that target files/commits exist
- For commit ranges, ensure commits are in repository history
---
--- END AGENTS.MD CONTENT ---
