// Issues page loader - shows all issues from the current project
import { projectApi, issuesApi } from '$lib/api.js';

export async function load() {
	try {
		// Get first project (simple implementation)
		const projectsData = await projectApi.list();
		const projects = projectsData.projects || [];
		
		if (projects.length === 0) {
			return {
				issues: [],
				project: null,
				hasData: false
			};
		}
		
		const project = projects[0];
		const issuesData = await issuesApi.list(project.ID);
		const issues = issuesData.issues || [];
		
		return {
			issues,
			project,
			hasData: true
		};
	} catch (error) {
		console.error('Failed to load issues:', error);
		return {
			issues: [],
			project: null,
			hasData: false,
			error: error.message
		};
	}
}