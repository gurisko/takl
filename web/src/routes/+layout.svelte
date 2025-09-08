<script>
	import '../app.css';
	
	// Navigation state
	let currentPath = '';
	
	// Handle route changes
	import { page } from '$app/stores';
	$: currentPath = $page.url.pathname;
</script>

<div class="app">
	<header class="header">
		<nav class="container flex items-center justify-between">
			<div class="flex items-center gap-4">
				<h1 class="logo">
					<a href="/" class="text-primary font-bold text-xl">TAKL</a>
				</h1>
				<span class="text-sm text-muted">Git-native Issue Tracker</span>
			</div>
			
			<div class="nav-links flex items-center gap-4">
				<a 
					href="/" 
					class="nav-link"
					class:active={currentPath === '/'}
				>
					Dashboard
				</a>
				<a 
					href="/issues" 
					class="nav-link"
					class:active={currentPath.startsWith('/issues')}
				>
					Issues
				</a>
				<a 
					href="/create" 
					class="nav-link"
					class:active={currentPath === '/create'}
				>
					Create
				</a>
				<a 
					href="/settings" 
					class="nav-link"
					class:active={currentPath === '/settings'}
				>
					Settings
				</a>
			</div>
		</nav>
	</header>
	
	<main class="main">
		<div class="container">
			<slot />
		</div>
	</main>
	
	<footer class="footer">
		<div class="container">
			<p class="text-sm text-muted text-center">
				TAKL Web Interface - Connected to daemon
			</p>
		</div>
	</footer>
</div>

<style>
	.app {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}
	
	.header {
		background: var(--color-bg-elevated);
		border-bottom: 1px solid var(--color-border);
		padding: 1rem 0;
		box-shadow: var(--shadow-sm);
	}
	
	.logo a {
		color: var(--color-primary);
		text-decoration: none;
	}
	
	.nav-link {
		color: var(--color-text-secondary);
		text-decoration: none;
		padding: 0.5rem 1rem;
		border-radius: var(--border-radius);
		transition: all 0.2s ease;
		font-weight: 500;
	}
	
	.nav-link:hover,
	.nav-link.active {
		color: var(--color-primary);
		background: rgba(37, 99, 235, 0.1);
	}
	
	.main {
		flex: 1;
		padding: 2rem 0;
	}
	
	.footer {
		background: var(--color-bg-secondary);
		border-top: 1px solid var(--color-border);
		padding: 1rem 0;
		margin-top: auto;
	}
	
	.text-primary { color: var(--color-primary); }
	.text-muted { color: var(--color-text-muted); }
	.text-sm { font-size: 0.875rem; }
	.text-xl { font-size: 1.25rem; }
	.font-bold { font-weight: 700; }
	.text-center { text-align: center; }
</style>