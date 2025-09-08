# TAKL

TAKL (pronounced "tackle") is a git-native issue tracker with a modern daemon-first architecture that stores issues as markdown files with YAML frontmatter while providing fast database-backed operations.

## Features

- 📝 **Git-native issue storage** - Issues stored as markdown files (source of truth)
- ⚡ **Fast SQLite database** - Instant search and queries with full-text search (FTS5)
- 🏗️ **Daemon architecture** - Background service with HTTP API over Unix sockets
- 🌐 **Multi-project support** - Centralized registry managing multiple projects
- 🔍 **Global search** - Search across all registered projects simultaneously
- 📋 **Rich metadata** - Priority, assignee, labels, timestamps, content
- 🔄 **Auto-commit** - Background git operations via daemon
- 🏷️ **Issue types** - bug, feature, task, epic with full workflow support

## Architecture

TAKL operates with a **daemon-first architecture**:

```
CLI Commands → HTTP API → Daemon → Database + Files + Git
```

- **CLI**: Thin client that communicates with daemon via Unix socket
- **Daemon**: Background service managing databases, files, and git operations  
- **Database**: Per-project SQLite with full-text search for fast queries
- **Files**: Markdown files remain the authoritative source of truth

## Installation

### From Source
```bash
git clone https://github.com/takl/takl
cd takl
make build
# Binary available at ./takl
```

## Quick Start

### 1. Register your project
```bash
cd /path/to/your/project
takl register . "My Project" --description="Main development project"
```

### 2. Initialize TAKL in the project
```bash
takl init
```

This creates:
- `.takl/issues/` directory (embedded mode)
- SQLite database at `~/.takl/projects/{project-id}.db`
- Configuration and project registry

### 3. Create issues (daemon auto-starts)

#### Git-style (like `git commit -m`)
```bash
takl create bug -m "Login button not working on mobile"
takl create feature -m "Add dark mode" --assignee=john@example.com --priority=high
takl create task -m "Update docs" --labels=docs,urgent --content="Need to update API documentation"
```

#### Interactive mode (prompts for details)
```bash
takl create bug
# Prompts for title, description, priority, assignee, labels
```

#### Traditional (still supported)
```bash
takl create feature "Add user authentication"
```

### 4. List and search issues (powered by database)
```bash
# List issues in current project
takl list
takl list --status=open --type=bug --assignee=dev@example.com

# Search within current project
takl search "authentication"

# Global search across all projects
takl search "login" --global
```

### 5. View specific issues
```bash
takl show ISS-001                # Smart ID resolution
takl show ISS-001 --verbose     # Shows full content
takl show ISS-001 --json        # Machine-readable JSON output
takl show ISS-001 --json -v     # JSON with content included
```

## Project Management

### Registry Operations
```bash
# List all registered projects
takl register --list

# Register a new project
takl register /path/to/project "Project Name" --description="Description"

# Clean up stale projects (with confirmation prompts)
takl register --cleanup

# Remove specific project
takl register --remove proj-abc123
```

### Daemon Management
```bash
# Check daemon status
takl daemon status

# Start daemon manually (usually auto-starts)
takl daemon start

# Stop daemon
takl daemon stop
```

## Commands

| Command | Description | Examples |
|---------|-------------|----------|
| `register` | Manage project registry | `takl register . "My Project"` |
| `init` | Initialize TAKL in project | `takl init` |
| `create` | Create new issues | `takl create bug -m "Fix login"` |
| `list` | List issues with filtering | `takl list --status=open --type=bug` |
| `search` | Search issues (FTS) | `takl search "authentication"` |
| `show` | Display issue details | `takl show ISS-001 --json` |
| `check` | Validate issue files | `takl check --global --fix` |
| `status` | Show project status | `takl status` |
| `context` | Show current project context | `takl context` |
| `daemon` | Manage background daemon | `takl daemon status` |

## Issue Types & Workflow

- **bug** 🐛 - Something is broken or not working correctly
- **feature** ✨ - New functionality or enhancement
- **task** ✅ - Work that needs to be done (maintenance, docs, etc.)
- **epic** 🎯 - Large feature or initiative spanning multiple tasks

## File Structure

### Embedded Mode (default for git repos)
```
your-project/
├── .takl/
│   ├── config.yaml         # Project configuration
│   └── issues/             # Issue files (source of truth)
│       ├── bug/
│       │   └── iss-001-login-broken.md
│       ├── feature/  
│       │   └── iss-002-add-dark-mode.md
│       └── task/
│           └── iss-003-update-docs.md
├── src/
└── README.md
```

### Global Configuration & Data
```
~/.takl/
├── daemon.sock           # Unix socket for API communication
├── daemon.pid           # Daemon process ID
├── projects.yaml        # Centralized project registry
└── projects/           # Per-project SQLite databases
    ├── proj-abc123.db  # Fast queries & full-text search
    └── proj-def456.db
```

## Issue Format

Issues are stored as markdown files with YAML frontmatter:

```markdown
---
id: ISS-001
type: bug
title: Login button not working
status: open
priority: high
assignee: dev@example.com
labels: ["ui", "mobile", "critical"]
created: 2025-09-01T15:30:00Z
updated: 2025-09-01T15:30:00Z
---

# Login button not working

The login button on the main page doesn't respond to clicks on mobile devices.

## Steps to Reproduce
1. Open the app on mobile Safari
2. Navigate to login page  
3. Tap login button
4. Nothing happens - no visual feedback

## Expected Behavior
Button should show loading state and submit the form to log user in.

## Additional Context
- Works fine on desktop browsers
- Affects iOS Safari and Chrome mobile
- Started after recent CSS changes
```

## Database Integration

TAKL maintains **dual persistence**:

1. **Markdown Files** (source of truth)
   - Human-readable, git-trackable
   - Can be edited directly with any editor
   - Committed to version control

2. **SQLite Database** (performance index)
   - Fast queries and filtering
   - Full-text search with FTS5
   - Cross-project analytics
   - Automatic sync from file changes

## Multi-Project Workflow

Work seamlessly across multiple projects:

```bash
# Register projects once
takl register ~/work/frontend "Frontend App"
takl register ~/work/backend "API Service"  
takl register ~/work/mobile "Mobile App"

# Context-aware operations
cd ~/work/frontend && takl create bug -m "Button styling"  # → Frontend project
cd ~/work/backend && takl create bug -m "API timeout"     # → Backend project

# Global operations
takl search "authentication" --global  # Search across all projects
takl list --global --assignee=me       # My issues across all projects
```

## Development

```bash
# Format and lint code
make fmt
make lint

# Run tests with coverage
make test
make test-cover  # Generates coverage.html

# Build and development
make build
make dev         # Full workflow: clean, fmt, vet, lint, test, build

# CI pipeline
make ci-test     # Tests with 80% coverage gate
```

## API Architecture

The daemon exposes a REST API over Unix sockets:

- **Project Operations**: `/api/projects/{id}/issues`
- **Global Search**: `/api/search?q=query`  
- **Registry**: `/api/registry/projects`
- **Health**: `/health`, `/stats`

CLI commands communicate with this API for all operations, providing:
- Consistent performance (database-backed)
- Centralized business logic
- Background processing capabilities
- Multi-project coordination

### Security

TAKL daemon binds to a Unix socket with `0600` permissions under `~/.takl/` (0700). This prevents other local users from accessing the API. TCP is not exposed by default, ensuring that only the owner can interact with the daemon and access project data.

## Configuration

Project-level config (`.takl/config.yaml`):
```yaml
project:
  name: "My Project"
paradigm:
  id: kanban
  options:
    wip_limits:
      doing: 3
      review: 2
    block_on_downstream_full: true
ui:
  date_format: "2006-01-02 15:04"
notifications:
  enabled: false
git:
  auto_commit: true
  commit_message: "Update issue: %s"
```

Global config (`~/.takl/projects.yaml`):
```yaml
projects:
  proj-abc123:
    id: proj-abc123
    name: Frontend App
    path: /home/user/work/frontend
    mode: embedded
    registered: 2025-09-01T10:00:00Z
    last_access: 2025-09-01T15:30:00Z
    active: true
    database_path: /home/user/.takl/projects/proj-abc123.db
```

## Performance

- **Issue Creation**: ~5ms (database + file + git commit)
- **Search**: ~1ms (SQLite FTS5 full-text search)  
- **List Operations**: ~2ms (indexed database queries)
- **Global Search**: ~10ms across dozens of projects
- **Cold Start**: Daemon starts in ~100ms when needed

## Coming Soon

- Issue updates and status transitions
- Advanced workflow automation  
- GitHub/GitLab integration
- Web dashboard UI
- Team collaboration features
- Issue templates and forms
- Advanced analytics and reporting

---

*TAKL combines the simplicity of markdown files with the power of modern database technology, giving you both human-readable issues and lightning-fast operations.*
