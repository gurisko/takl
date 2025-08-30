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
