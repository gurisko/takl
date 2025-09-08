<script>
	import { goto } from '$app/navigation';
	
	// Get live data from the loader
	export let data;
	
	// Extract data from the loader
	$: ({ projects, currentProject, issues, stats, recentIssues, hasData, error } = data);
	
	function navigateToIssue(issueId) {
		goto(`/issues/${issueId}`);
	}
	
	function formatDate(dateStr) {
		return new Date(dateStr).toLocaleDateString();
	}
	
	function getTypeIcon(type) {
		const icons = {
			bug: '🐛',
			feature: '✨',
			task: '✅',
			epic: '🎯'
		};
		return icons[type] || '📄';
	}
</script>

<svelte:head>
	<title>Dashboard - TAKL</title>
</svelte:head>

<div class="dashboard">
	<div class="dashboard-header">
		<h1>Dashboard</h1>
		<p class="text-muted">Overview of your issues and project activity</p>
	</div>
	
	{#if error}
		<div class="error">
			<p>Failed to load dashboard: {error}</p>
			<a href="/" class="button button-secondary">Retry</a>
		</div>
	{:else if !hasData}
		<div class="empty-state card">
			<h2>🎯 Welcome to TAKL</h2>
			<p>No projects registered yet. Let's get started!</p>
			<p class="text-muted">Register a project to start tracking issues.</p>
			<div class="empty-actions">
				<code>takl register . "My Project"</code>
			</div>
		</div>
	{:else}
		<!-- Project info -->
		{#if currentProject}
			<div class="project-info card">
				<div class="flex items-center justify-between">
					<div>
						<h2>📁 {currentProject.Name}</h2>
						<p class="text-sm text-muted">{currentProject.Path}</p>
						<p class="text-xs text-muted">Project ID: {currentProject.ID}</p>
					</div>
					<div class="project-actions">
						{#if projects.length > 1}
							<select class="button button-secondary">
								{#each projects as project}
									<option value={project.ID} selected={project.ID === currentProject.ID}>
										{project.Name}
									</option>
								{/each}
							</select>
						{/if}
						<a href="/settings" class="button">Settings</a>
					</div>
				</div>
			</div>
			
			<!-- Stats overview -->
			<div class="stats-grid">
				<div class="stat-card card">
					<div class="stat-number">{stats.totalIssues}</div>
					<div class="stat-label">Total Issues</div>
				</div>
				<div class="stat-card card">
					<div class="stat-number">{stats.openIssues}</div>
					<div class="stat-label">Open Issues</div>
				</div>
				<div class="stat-card card">
					<div class="stat-number">{stats.inProgress}</div>
					<div class="stat-label">In Progress</div>
				</div>
				<div class="stat-card card">
					<div class="stat-number">{stats.completed}</div>
					<div class="stat-label">Completed</div>
				</div>
			</div>
		{/if}
		
		<!-- Recent issues -->
		<div class="recent-section">
			<div class="section-header">
				<h2>Recent Issues</h2>
				<div class="section-actions">
					<a href="/create" class="button">Create Issue</a>
					<a href="/issues" class="button button-secondary">View All</a>
				</div>
			</div>
			
			{#if recentIssues.length > 0}
				<div class="issues-list">
					{#each recentIssues as issue}
						<div class="issue-item card" role="button" tabindex="0" on:click={() => navigateToIssue(issue.id)} on:keydown={(e) => e.key === 'Enter' && navigateToIssue(issue.id)}>
							<div class="issue-header">
								<div class="issue-id-type">
									<span class="issue-icon">{getTypeIcon(issue.type)}</span>
									<span class="issue-id">{issue.id}</span>
								</div>
								<div class="issue-meta">
									<span class="status-badge status-{issue.status}">{issue.status}</span>
									<span class="priority-{issue.priority}">●</span>
								</div>
							</div>
							<div class="issue-title">{issue.title}</div>
							<div class="issue-footer">
								<span class="assignee">👤 {issue.assignee || 'Unassigned'}</span>
								<span class="updated">Updated {formatDate(issue.updated || issue.created)}</span>
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<div class="empty-state card">
					<p>No issues found in this project</p>
					<p class="text-muted">Create your first issue with the CLI:</p>
					<div class="empty-actions">
						<code>takl create bug -m "Issue title"</code>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.dashboard {
		max-width: 1200px;
		margin: 0 auto;
	}
	
	.dashboard-header {
		margin-bottom: 2rem;
	}
	
	.dashboard-header h1 {
		margin-bottom: 0.5rem;
		font-size: 2rem;
		font-weight: 700;
	}
	
	.project-info {
		margin-bottom: 2rem;
	}
	
	.project-actions {
		display: flex;
		gap: 0.5rem;
	}
	
	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
		margin-bottom: 2rem;
	}
	
	.stat-card {
		text-align: center;
		padding: 2rem 1rem;
	}
	
	.stat-number {
		font-size: 2.5rem;
		font-weight: 700;
		color: var(--color-primary);
		margin-bottom: 0.5rem;
	}
	
	.stat-label {
		color: var(--color-text-secondary);
		font-size: 0.875rem;
		font-weight: 500;
	}
	
	.recent-section {
		margin-bottom: 2rem;
	}
	
	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}
	
	.section-header h2 {
		margin: 0;
		font-size: 1.5rem;
	}
	
	.section-actions {
		display: flex;
		gap: 0.5rem;
	}
	
	.issues-list {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	
	.issue-item {
		cursor: pointer;
		transition: all 0.2s ease;
		border-left: 3px solid transparent;
	}
	
	.issue-item:hover {
		border-left-color: var(--color-primary);
		transform: translateX(2px);
	}
	
	.issue-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.5rem;
	}
	
	.issue-id-type {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	
	.issue-icon {
		font-size: 1rem;
	}
	
	.issue-id {
		font-family: var(--font-family-mono);
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--color-text-secondary);
	}
	
	.issue-meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	
	.issue-title {
		font-weight: 600;
		margin-bottom: 0.75rem;
		line-height: 1.4;
	}
	
	.issue-footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.875rem;
		color: var(--color-text-secondary);
	}
	
	.assignee {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}
	
	.empty-state {
		text-align: center;
		padding: 3rem 2rem;
	}
	
	.empty-state p {
		margin-bottom: 1rem;
		color: var(--color-text-muted);
	}
	
	.empty-actions {
		margin-top: 1rem;
		padding: 1rem;
		background: var(--color-gray-50);
		border-radius: 6px;
		border-left: 3px solid var(--color-primary);
	}
	
	.empty-actions code {
		font-family: var(--font-family-mono);
		background: transparent;
		padding: 0;
		border: none;
		color: var(--color-primary);
		font-weight: 600;
	}
	
	.loading {
		padding: 3rem;
		text-align: center;
		color: var(--color-text-muted);
	}
	
	@media (max-width: 768px) {
		.dashboard-header h1 {
			font-size: 1.5rem;
		}
		
		.stats-grid {
			grid-template-columns: repeat(2, 1fr);
		}
		
		.section-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}
		
		.project-actions,
		.section-actions {
			flex-direction: column;
			width: 100%;
		}
		
		.issue-footer {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.25rem;
		}
	}
</style>