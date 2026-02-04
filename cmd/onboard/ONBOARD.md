Sanity Onboarding

Add this minimal snippet to AGENTS.md (or create it):

--- BEGIN AGENTS.MD CONTENT ---
---
## Sanity Usage

This project uses **sanity** for the following:

1. Auditing AI-generated code
2. Examining, identifying, and fixing software design

Both use cases require the agent to run the `graph` subcommand, either after making changes or if the user requests to visualize the design.

### Agent Instructions: Using the `graph` Subcommand

#### Implementation Based on Agent Environment:

**For shell/terminal agents (macOS):**
```bash
open $(sanity graph -u)  # Or equivalent command on other shells/operating systems
```

This generates a visualization URL and opens it directly in the default browser.

**For IDE/desktop agents with mermaid rendering support:**
```bash
sanity graph -f mermaid
```

Render the mermaid diagram output directly within the IDE or desktop application interface.

---
--- END AGENTS.MD CONTENT ---
