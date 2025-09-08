<script>
	import { goto } from '$app/navigation';
	
	// Get live data from the loader
	export let data;
	
	$: ({ issues, project, hasData, error } = data);
	$: filteredIssues = issues; // Start with all issues, could add filtering later
	
	// Filters (for future enhancement)
	let filterType = 'all';
	let filterStatus = 'all';
	let filterPriority = 'all';
	let searchQuery = '';
	
	// Simple client-side filtering (can be enhanced later)
	$: {
		if (issues && issues.length > 0) {
			filteredIssues = issues.filter(issue => {
				if (searchQuery) {
					const query = searchQuery.toLowerCase();
					const searchable = `${issue.title} ${issue.id}`.toLowerCase();
					return searchable.includes(query);
				}
				return true;
			});
		} else {
			filteredIssues = [];
		}
	}
	
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
	
	function clearFilters() {
		filterType = 'all';
		filterStatus = 'all';
		filterPriority = 'all';
		searchQuery = '';
	}
</script>

<svelte:head>
	<title>Issues - TAKL</title>
</svelte:head>

<div class="issues-page">
	<div class="page-header">
		<div class="header-content">
			<h1>Issues</h1>
			{#if project}
				<p class="text-muted">Issues from {project.Name}</p>
			{:else}
				<p class="text-muted">Manage and track all project issues</p>
			{/if}
		</div>
		<div class="header-actions">
			<p class="text-muted">Use CLI to create issues: <code>takl create bug -m "Title"</code></p>
		</div>
	</div>
	
	<!-- Filters -->
	<div class="filters card">
		<div class="filters-row">
			<div class="search-input">
				<input
					type="text"
					placeholder="Search issues..."
					class="input"
					bind:value={searchQuery}
				/>
			</div>
			
			<div class="filter-selects">
				<select class="input" bind:value={filterType}>
					<option value="all">All Types</option>
					<option value="bug">Bug</option>
					<option value="feature">Feature</option>
					<option value="task">Task</option>
					<option value="epic">Epic</option>
				</select>
				
				<select class="input" bind:value={filterStatus}>
					<option value="all">All Status</option>
					<option value="backlog">Backlog</option>
					<option value="doing">Doing</option>
					<option value="review">Review</option>
					<option value="done">Done</option>
				</select>
				
				<select class="input" bind:value={filterPriority}>
					<option value="all">All Priority</option>
					<option value="low">Low</option>
					<option value="medium">Medium</option>
					<option value="high">High</option>
					<option value="critical">Critical</option>
				</select>
				
				<button class="button button-secondary" on:click={clearFilters}>
					Clear
				</button>
			</div>
		</div>
		
		{#if filteredIssues.length !== issues.length}
			<div class="filter-summary">
				Showing {filteredIssues.length} of {issues.length} issues
			</div>
		{/if}
	</div>
	
	{#if error}
		<div class="error">
			<p>Failed to load issues: {error}</p>
			<a href="/issues" class="button button-secondary">Retry</a>
		</div>
	{:else if !hasData}
		<div class="empty-state card">
			<h3>No project found</h3>
			<p>Register a project first to start tracking issues</p>
			<div class="empty-actions">
				<code>takl register . "My Project"</code>
			</div>
		</div>
	{:else if filteredIssues.length === 0}
		<div class="empty-state card">
			<h3>No issues yet</h3>
			<p>Create your first issue with the CLI</p>
			<div class="empty-actions">
				<code>takl create bug -m "Issue title"</code>
			</div>
		</div>
	{:else}
		<!-- Issues list -->
		<div class="issues-list">
			{#each filteredIssues as issue}
				<div 
					class="issue-item card" 
					role="button" 
					tabindex="0" 
					on:click={() => navigateToIssue(issue.id)} 
					on:keydown={(e) => e.key === 'Enter' && navigateToIssue(issue.id)}
				>
					<div class="issue-header">
						<div class="issue-id-type">
							<span class="issue-icon">{getTypeIcon(issue.type)}</span>
							<span class="issue-id">{issue.id}</span>
							<span class="issue-type">{issue.type}</span>
						</div>
						<div class="issue-meta">
							<span class="status-badge status-{issue.status}">{issue.status}</span>
							<span class="priority-indicator priority-{issue.priority}">●</span>
						</div>
					</div>
					
					<div class="issue-title">{issue.title}</div>
					
					{#if issue.content}
						<div class="issue-content">
							{issue.content}
						</div>
					{/if}
					
					{#if issue.labels && issue.labels.length > 0}
						<div class="issue-tags">
							{#each issue.labels as label}
								<span class="tag">{label}</span>
							{/each}
						</div>
					{/if}
					
					<div class="issue-footer">
						<div class="assignee">
							👤 {issue.assignee || 'Unassigned'}
						</div>
						<div class="dates">
							<span>Created {formatDate(issue.created)}</span>
							{#if issue.updated && issue.updated !== issue.created}
								<span>• Updated {formatDate(issue.updated)}</span>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.issues-page {
		max-width: 1200px;
		margin: 0 auto;
	}
	
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		margin-bottom: 2rem;
	}
	
	.page-header h1 {
		margin-bottom: 0.5rem;
		font-size: 2rem;
		font-weight: 700;
	}
	
	.filters {
		margin-bottom: 2rem;
	}
	
	.filters-row {
		display: flex;
		gap: 1rem;
		align-items: stretch;
	}
	
	.search-input {
		flex: 1;
		min-width: 300px;
	}
	
	.filter-selects {
		display: flex;
		gap: 0.5rem;
		flex-shrink: 0;
	}
	
	.filter-selects select,
	.filter-selects button {
		min-width: 120px;
	}
	
	.filter-summary {
		margin-top: 1rem;
		font-size: 0.875rem;
		color: var(--color-text-secondary);
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
		box-shadow: var(--shadow-md);
	}
	
	.issue-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.75rem;
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
	
	.issue-type {
		font-size: 0.75rem;
		color: var(--color-text-muted);
		text-transform: uppercase;
		font-weight: 500;
		letter-spacing: 0.05em;
	}
	
	.issue-meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	
	.priority-indicator {
		font-size: 1.25rem;
		line-height: 1;
	}
	
	.issue-title {
		font-weight: 600;
		font-size: 1.125rem;
		margin-bottom: 0.5rem;
		line-height: 1.4;
	}
	
	.issue-content {
		color: var(--color-text-secondary);
		line-height: 1.5;
		margin-bottom: 1rem;
		overflow: hidden;
		text-overflow: ellipsis;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
	}
	
	.issue-tags {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1rem;
		flex-wrap: wrap;
	}
	
	.tag {
		display: inline-block;
		padding: 0.25rem 0.5rem;
		background: var(--color-bg-secondary);
		color: var(--color-text-secondary);
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 500;
	}
	
	.issue-footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.875rem;
		color: var(--color-text-secondary);
	}
	
	.dates {
		display: flex;
		gap: 0.5rem;
	}
	
	.empty-state {
		text-align: center;
		padding: 3rem 2rem;
	}
	
	.empty-state h3 {
		margin-bottom: 0.5rem;
		color: var(--color-text);
	}
	
	.empty-state p {
		margin-bottom: 1.5rem;
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
		.page-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}
		
		.filters-row {
			flex-direction: column;
		}
		
		.filter-selects {
			flex-direction: column;
		}
		
		.filter-selects select,
		.filter-selects button {
			min-width: unset;
		}
		
		.issue-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.5rem;
		}
		
		.issue-footer {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.25rem;
		}
		
		.dates {
			flex-direction: column;
			gap: 0.25rem;
		}
	}
</style>