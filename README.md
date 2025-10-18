# TAKL

> ⚠️ **EXPERIMENTAL:** This is an experimental project exploring whether git-native issue tracking makes sense in the era of CLI AI tools.

TAKL (pronounced "tackle") is a git-native issue tracker.

## Requirements

- **Unix-like OS** (Linux, macOS, BSD) - Windows not supported
- Go 1.25+ for building from source

## Features

- 📝 **Git-native issue storage** - Issues stored as markdown files (source of truth)
- 🏗️ **Daemon architecture** - Background service with HTTP API over Unix sockets
- 🌐 **Multi-project support** - Centralized registry managing multiple projects

## Architecture

TAKL operates with a **daemon-first architecture**:

```
CLI Commands → HTTP API → Daemon → Registry + Files + Git
```

- **CLI**: Thin client that communicates with daemon via Unix socket
- **Daemon**: Background service managing project registry and git operations
- **Registry**: YAML-based project registry with atomic file operations
- **Files**: Markdown files remain the authoritative source of truth

## Quick Start

### 1. Start the daemon
```bash
takl daemon start
```

### 2. Register a project
```bash
takl projects register --name "My Project" --path ~/src/myproject
```

### 3. List registered projects
```bash
takl projects list
```

## Commands

### Issue Management

```bash
# List issues with filters
takl list                              # List all issues
takl list --status "In Progress"       # Filter by status
takl list --assignee "John"            # Filter by assignee (name or email substring)
takl list --labels bug,urgent          # Filter by labels (must match all)
takl list --search "database error"    # Search in title and description
takl list --json                       # Output JSON for piping

# Show issue details
takl show PROJ-123                     # Display full issue details
takl show PROJ-456 --json              # Output as JSON

# Unix pipeline composition (using --json flag)
takl list --json | jq -r '.issues[] | "\(.jira_key): \(.title)"'
takl list --status Open --json | jq '.count'
takl list --labels bug --json | jq '.issues[].jira_key'
```

**Note:** The `--assignee` filter supports case-insensitive substring matching on both display names and email addresses.

### Jira Bridge

**Configuration:** Create `.takl/jira.json` in your project directory:

```json
{
  "base_url": "https://your-domain.atlassian.net",
  "email": "your-email@example.com",
  "api_token": "your-api-token",
  "project": "PROJ"
}
```

To create an API token, visit: https://id.atlassian.com/manage-profile/security/api-tokens

**Commands:**

```bash
# Pull issues from Jira to local markdown files
takl jira pull

# Fetch and cache project members (for assignee resolution)
takl jira members          # Table output
takl jira members --json   # JSON output

# Fetch and cache project workflow statuses
takl jira workflow         # Table output grouped by category
takl jira workflow --json  # JSON output
```

**Caches:**
- `.takl/jira-members.json` - Project members cache (used for assignee resolution)
- `.takl/jira-workflow.json` - Workflow statuses cache (includes status categories)

### Daemon Management

```bash
takl daemon start   # Start the daemon (foreground)
takl daemon status  # Check daemon status
takl daemon stop    # Stop the daemon gracefully
```

For background operation:
```bash
nohup takl daemon start > /tmp/takl-daemon.log 2>&1 &
```

### Project Management

```bash
# Register a project
takl projects register --name "Project Name" --path ~/src/project
takl projects register -n "Short Name" -p .  # Short flags

# List all registered projects
takl projects list          # Tabular output
takl projects list --json   # JSON output

# Remove a project
takl projects remove <project-id>               # By ID (with confirmation)
takl projects remove <project-id> -y            # Skip confirmation
```

## File Locations

TAKL follows the XDG Base Directory specification:

- **Runtime files** (socket, PID):
  - `$XDG_RUNTIME_DIR/takl/` (Linux)
  - `~/.takl/` (fallback)
- **State files** (project registry):
  - `$XDG_STATE_HOME/takl/projects.yaml` (Linux)
  - `~/.local/state/takl/projects.yaml` (fallback)

Socket permissions: `0600` (owner-only)
Directory permissions: `0700` (owner-only)

## Development

### Build

```bash
make build  # Build the takl binary
```

TAKL requires a Unix-like operating system. The build will fail on Windows due to `//go:build unix` constraints.

### Code Quality

```bash
make fmt    # Format code with gofmt
make vet    # Run go vet for correctness checks
make check  # Run fmt + vet
```

### Linting

Requires [golangci-lint](https://golangci-lint.run/):

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
make lint
```

### Testing

```bash
make test   # Run tests
```

### Full Development Workflow

```bash
make dev    # Clean, format, vet, and build
```
