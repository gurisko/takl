// Dashboard page loader - fetches live data from TAKL daemon
import { projectApi, issuesApi } from '$lib/api.js';

export async function load() {
	try {
		// Get all registered projects
		const projectsData = await projectApi.list();
		const projects = projectsData.projects || [];
		
		// If we have projects, get issues from the first one
		let issues = [];
		let currentProject = null;
		
		if (projects.length > 0) {
			currentProject = projects[0];
			try {
				const issuesData = await issuesApi.list(currentProject.ID);
				issues = issuesData.issues || [];
			} catch (error) {
				console.warn('Failed to load issues:', error);
				issues = [];
			}
		}
		
		// Calculate stats
		const stats = {
			totalIssues: issues.length,
			openIssues: issues.filter(i => i.status !== 'done').length,
			inProgress: issues.filter(i => i.status === 'in_progress' || i.status === 'doing').length,
			completed: issues.filter(i => i.status === 'done').length,
		};
		
		// Get recent issues (last 5)
		const recentIssues = issues
			.sort((a, b) => new Date(b.updated || b.created) - new Date(a.updated || a.created))
			.slice(0, 5);
		
		return {
			projects,
			currentProject,
			issues,
			stats,
			recentIssues,
			hasData: projects.length > 0
		};
	} catch (error) {
		console.error('Failed to load dashboard data:', error);
		return {
			projects: [],
			currentProject: null,
			issues: [],
			stats: { totalIssues: 0, openIssues: 0, inProgress: 0, completed: 0 },
			recentIssues: [],
			hasData: false,
			error: error.message
		};
	}
}