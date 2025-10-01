# TAKL

> ⚠️ **EXPERIMENTAL:** This is an experimental project exploring whether git-native issue tracking makes sense in the era of CLI AI tools.

TAKL (pronounced "tackle") is a git-native issue tracker.

## Features

- 📝 **Git-native issue storage** - Issues stored as markdown files (source of truth)
- 🏗️ **Daemon architecture** - Background service with HTTP API over Unix sockets
- 🌐 **Multi-project support** - Centralized registry managing multiple projects
- 🔍 **Global search** - Search across all registered projects simultaneously
- 🔄 **Auto-commit** - Background git operations via daemon

## Architecture

TAKL operates with a **daemon-first architecture**:

```
CLI Commands → HTTP API → Daemon → Database + Files + Git
```

- **CLI**: Thin client that communicates with daemon via Unix socket
- **Daemon**: Background service managing databases, files, and git operations
- **Database**: Per-project SQLite with full-text search for fast queries
- **Files**: Markdown files remain the authoritative source of truth

## Daemon Management

Start the daemon:
```bash
takl daemon start
```

Check daemon status:
```bash
takl daemon status
```

Stop the daemon:
```bash
takl daemon stop
```

The daemon uses Unix socket at `~/.takl/daemon.sock` with secure permissions (0600).

## Development

### Build

```bash
make build  # Build the takl binary
```

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
