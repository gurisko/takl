<script>
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { marked } from 'marked';
	import { onMount } from 'svelte';
	
	export let data;
	$: ({ issue, project } = data);
	
	// Configure marked for better security and rendering
	let renderedContent = '';
	
	onMount(() => {
		if (issue.content) {
			marked.setOptions({
				breaks: true,
				gfm: true,
				sanitize: false, // We trust TAKL content
			});
			renderedContent = marked(issue.content);
		}
	});
	
	// Reactive update if content changes
	$: if (issue.content) {
		marked.setOptions({
			breaks: true,
			gfm: true,
			sanitize: false,
		});
		renderedContent = marked(issue.content);
	}
	
	function goBack() {
		goto('/issues');
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
	
	function formatDate(dateStr) {
		return new Date(dateStr).toLocaleString();
	}
	
	function getPriorityColor(priority) {
		const colors = {
			low: '#22c55e',
			medium: '#f59e0b', 
			high: '#ef4444',
			critical: '#dc2626'
		};
		return colors[priority] || '#6b7280';
	}
	
	function getStatusColor(status) {
		const colors = {
			backlog: '#6b7280',
			doing: '#3b82f6',
			review: '#f59e0b',
			done: '#22c55e'
		};
		return colors[status] || '#6b7280';
	}
</script>

<svelte:head>
	<title>{issue.title} - Issues - TAKL</title>
</svelte:head>

<div class="issue-detail">
	<!-- Header -->
	<div class="issue-header">
		<button class="back-button button button-secondary" on:click={goBack}>
			← Back to Issues
		</button>
		
		<div class="issue-meta">
			<span class="issue-project">{project.Name}</span>
		</div>
	</div>
	
	<!-- Issue card -->
	<div class="issue-card card">
		<div class="issue-title-row">
			<div class="issue-id-type">
				<span class="issue-icon">{getTypeIcon(issue.type)}</span>
				<span class="issue-id">{issue.id}</span>
				<span class="issue-type">{issue.type}</span>
			</div>
			
			<div class="issue-badges">
				<span 
					class="status-badge" 
					style="background-color: {getStatusColor(issue.status)}20; color: {getStatusColor(issue.status)}; border: 1px solid {getStatusColor(issue.status)}40"
				>
					{issue.status}
				</span>
				<span 
					class="priority-badge"
					style="background-color: {getPriorityColor(issue.priority)}20; color: {getPriorityColor(issue.priority)}; border: 1px solid {getPriorityColor(issue.priority)}40"
				>
					{issue.priority}
				</span>
			</div>
		</div>
		
		<h1 class="issue-title">{issue.title}</h1>
		
		<!-- Issue metadata -->
		<div class="issue-metadata">
			<div class="metadata-row">
				<span class="metadata-label">Assignee:</span>
				<span class="metadata-value">👤 {issue.assignee || 'Unassigned'}</span>
			</div>
			<div class="metadata-row">
				<span class="metadata-label">Created:</span>
				<span class="metadata-value">{formatDate(issue.created)}</span>
			</div>
			{#if issue.updated && issue.updated !== issue.created}
				<div class="metadata-row">
					<span class="metadata-label">Updated:</span>
					<span class="metadata-value">{formatDate(issue.updated)}</span>
				</div>
			{/if}
			{#if issue.labels && issue.labels.length > 0}
				<div class="metadata-row">
					<span class="metadata-label">Labels:</span>
					<div class="issue-labels">
						{#each issue.labels as label}
							<span class="label-tag">{label}</span>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	</div>
	
	<!-- Issue content -->
	{#if issue.content}
		<div class="issue-content card">
			<h2>Description</h2>
			<div class="markdown-content">
				{@html renderedContent}
			</div>
		</div>
	{:else}
		<div class="issue-content card">
			<p class="no-content">No description provided for this issue.</p>
		</div>
	{/if}
	
	<!-- Actions -->
	<div class="issue-actions card">
		<h3>Actions</h3>
		<p class="text-muted">Use the CLI to modify this issue:</p>
		<div class="cli-commands">
			<div class="command-example">
				<code>takl update {issue.id} --status done</code>
				<span class="command-desc">Mark as done</span>
			</div>
			<div class="command-example">
				<code>takl update {issue.id} --assignee user@example.com</code>
				<span class="command-desc">Assign to user</span>
			</div>
			<div class="command-example">
				<code>takl show {issue.id}</code>
				<span class="command-desc">View in CLI</span>
			</div>
		</div>
	</div>
</div>

<style>
	.issue-detail {
		max-width: 1000px;
		margin: 0 auto;
		padding: 0 1rem;
	}
	
	.issue-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 2rem;
		padding: 1rem 0;
	}
	
	.back-button {
		text-decoration: none;
	}
	
	.issue-project {
		font-size: 0.875rem;
		color: var(--color-text-secondary);
		font-weight: 500;
	}
	
	.issue-card {
		margin-bottom: 2rem;
	}
	
	.issue-title-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.5rem;
	}
	
	.issue-id-type {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	
	.issue-icon {
		font-size: 1.5rem;
	}
	
	.issue-id {
		font-family: var(--font-family-mono);
		font-size: 1rem;
		font-weight: 700;
		color: var(--color-text-secondary);
	}
	
	.issue-type {
		font-size: 0.875rem;
		color: var(--color-text-muted);
		text-transform: uppercase;
		font-weight: 600;
		letter-spacing: 0.05em;
	}
	
	.issue-badges {
		display: flex;
		gap: 0.5rem;
	}
	
	.status-badge,
	.priority-badge {
		padding: 0.375rem 0.75rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	
	.issue-title {
		font-size: 2rem;
		font-weight: 700;
		line-height: 1.3;
		margin-bottom: 1.5rem;
		color: var(--color-text);
	}
	
	.issue-metadata {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		border-top: 1px solid var(--color-border);
		padding-top: 1.5rem;
	}
	
	.metadata-row {
		display: flex;
		align-items: center;
		gap: 1rem;
	}
	
	.metadata-label {
		font-weight: 600;
		color: var(--color-text-secondary);
		min-width: 80px;
		font-size: 0.875rem;
	}
	
	.metadata-value {
		font-size: 0.875rem;
		color: var(--color-text);
	}
	
	.issue-labels {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	
	.label-tag {
		display: inline-block;
		padding: 0.25rem 0.5rem;
		background: var(--color-bg-secondary);
		color: var(--color-text-secondary);
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 500;
		border: 1px solid var(--color-border);
	}
	
	.issue-content {
		margin-bottom: 2rem;
	}
	
	.issue-content h2 {
		margin-bottom: 1rem;
		font-size: 1.25rem;
		font-weight: 600;
	}
	
	.markdown-content {
		line-height: 1.6;
		color: var(--color-text);
	}
	
	/* Markdown styling */
	.markdown-content :global(h1),
	.markdown-content :global(h2),
	.markdown-content :global(h3),
	.markdown-content :global(h4),
	.markdown-content :global(h5),
	.markdown-content :global(h6) {
		margin-top: 1.5rem;
		margin-bottom: 0.5rem;
		font-weight: 600;
		line-height: 1.3;
	}
	
	.markdown-content :global(h1) { font-size: 1.875rem; }
	.markdown-content :global(h2) { font-size: 1.5rem; }
	.markdown-content :global(h3) { font-size: 1.25rem; }
	.markdown-content :global(h4) { font-size: 1.125rem; }
	
	.markdown-content :global(p) {
		margin-bottom: 1rem;
	}
	
	.markdown-content :global(ul),
	.markdown-content :global(ol) {
		margin-bottom: 1rem;
		padding-left: 1.5rem;
	}
	
	.markdown-content :global(li) {
		margin-bottom: 0.25rem;
	}
	
	.markdown-content :global(blockquote) {
		border-left: 4px solid var(--color-primary);
		padding-left: 1rem;
		margin: 1rem 0;
		font-style: italic;
		color: var(--color-text-secondary);
	}
	
	.markdown-content :global(code) {
		background: var(--color-bg-secondary);
		padding: 0.125rem 0.25rem;
		border-radius: 3px;
		font-family: var(--font-family-mono);
		font-size: 0.875rem;
		color: var(--color-primary);
	}
	
	.markdown-content :global(pre) {
		background: var(--color-bg-secondary);
		padding: 1rem;
		border-radius: 6px;
		overflow-x: auto;
		margin: 1rem 0;
	}
	
	.markdown-content :global(pre code) {
		background: none;
		padding: 0;
		color: var(--color-text);
	}
	
	.markdown-content :global(table) {
		width: 100%;
		border-collapse: collapse;
		margin: 1rem 0;
	}
	
	.markdown-content :global(th),
	.markdown-content :global(td) {
		border: 1px solid var(--color-border);
		padding: 0.5rem;
		text-align: left;
	}
	
	.markdown-content :global(th) {
		background: var(--color-bg-secondary);
		font-weight: 600;
	}
	
	.no-content {
		color: var(--color-text-muted);
		font-style: italic;
		text-align: center;
		padding: 2rem;
	}
	
	.issue-actions h3 {
		margin-bottom: 1rem;
		font-size: 1.125rem;
		font-weight: 600;
	}
	
	.cli-commands {
		margin-top: 1rem;
	}
	
	.command-example {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem;
		background: var(--color-bg-secondary);
		border-radius: 6px;
		border-left: 3px solid var(--color-primary);
		margin-bottom: 0.5rem;
	}
	
	.command-example code {
		font-family: var(--font-family-mono);
		color: var(--color-primary);
		font-weight: 600;
	}
	
	.command-desc {
		font-size: 0.875rem;
		color: var(--color-text-secondary);
	}
	
	@media (max-width: 768px) {
		.issue-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}
		
		.issue-title-row {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}
		
		.issue-title {
			font-size: 1.5rem;
		}
		
		.metadata-row {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.25rem;
		}
		
		.command-example {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.5rem;
		}
	}
</style>