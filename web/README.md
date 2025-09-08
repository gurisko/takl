# TAKL Web Interface

A modern SvelteKit-based web interface for the TAKL git-native issue tracker.

## Features

- **Dashboard**: Overview of project issues and activity
- **Issue Management**: Create, view, edit, and manage issues
- **Real-time Updates**: WebSocket integration for live updates
- **Template System**: Create issues using predefined templates
- **Import/Export**: Bulk operations with CSV/YAML support
- **Paradigm Support**: Kanban boards and Scrum workflows
- **Responsive Design**: Mobile-friendly interface
- **Dark Mode**: Automatic dark/light theme support

## Architecture

The web interface communicates with the TAKL daemon via HTTP API and WebSocket connections:

```
Web Browser ←→ SvelteKit App ←→ TAKL Daemon ←→ Git Repository
```

### Key Components

- **`src/routes/`**: Page components and routing
- **`src/lib/api.js`**: API client for daemon communication
- **`src/app.css`**: Global styles and design system
- **`vite.config.js`**: Development proxy to daemon

### API Integration

The web interface uses a comprehensive API client to interact with the TAKL daemon:

```javascript
import { issuesApi, projectApi } from '$lib/api.js';

// List issues
const issues = await issuesApi.list(projectId, { status: 'open' });

// Create issue
const newIssue = await issuesApi.create(projectId, {
  type: 'bug',
  title: 'Fix login issue',
  content: 'Users cannot log in...',
  priority: 'high'
});
```

## Development

### Prerequisites

- Node.js 18+
- TAKL daemon running on `localhost:8080`

### Setup

```bash
cd web
npm install
npm run dev
```

The development server runs on `http://localhost:3000` with API proxy to the TAKL daemon.

### Building

```bash
npm run build
```

Builds the app for production to the `build/` directory using the Node.js adapter.

## Deployment

The web interface can be deployed alongside the TAKL daemon or as a standalone service:

### With Daemon
The daemon can serve the web interface directly (future feature).

### Standalone
Deploy the built app to any Node.js hosting service:

```bash
npm run build
node build/index.js
```

### Docker
```bash
# Build image
docker build -t takl-web .

# Run container
docker run -p 3000:3000 \
  -e TAKL_DAEMON_URL=http://daemon:8080 \
  takl-web
```

## Configuration

Environment variables:

- `TAKL_DAEMON_URL`: TAKL daemon URL (default: http://localhost:8080)
- `PORT`: Web server port (default: 3000)
- `NODE_ENV`: Environment (development/production)

## Real-time Features

The web interface supports real-time updates via WebSocket:

```javascript
import { TAKLRealtimeClient } from '$lib/api.js';

const client = new TAKLRealtimeClient(projectId);
client.connect();

client.on('issue-updated', (issue) => {
  // Handle issue update
});

client.on('issue-created', (issue) => {
  // Handle new issue
});
```

## UI Components

The interface uses a consistent design system:

- **Colors**: Primary blue, semantic colors for status/priority
- **Typography**: System fonts with monospace for IDs
- **Cards**: Elevated surfaces for content
- **Buttons**: Primary, secondary, and danger variants
- **Badges**: Status and priority indicators
- **Forms**: Consistent input styling

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Development Roadmap

- [x] Basic SvelteKit setup and routing
- [x] Dashboard with project overview
- [x] Issue list with filtering and search
- [x] API client with full TAKL daemon integration
- [x] Responsive design and dark mode
- [ ] Real-time WebSocket updates
- [ ] Issue detail pages with editing
- [ ] Create issue form with templates
- [ ] Kanban board view
- [ ] Settings and configuration
- [ ] Import/export interface
- [ ] User authentication (when daemon supports it)
- [ ] Mobile app (PWA)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes following the established patterns
4. Test with the TAKL daemon
5. Submit a pull request

## License

MIT License - see the main TAKL project for details.