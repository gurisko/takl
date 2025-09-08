// Issue detail page loader - fetches specific issue data
import { projectApi, issuesApi } from '$lib/api.js';
import { error } from '@sveltejs/kit';

// Don't prerender dynamic routes
export const prerender = false;

export async function load({ params }) {
	try {
		// Get first project (simple implementation for now)
		const projectsData = await projectApi.list();
		const projects = projectsData.projects || [];
		
		if (projects.length === 0) {
			throw error(404, 'No projects found');
		}
		
		const project = projects[0];
		
		// Get the specific issue
		const issue = await issuesApi.get(project.ID, params.id);
		
		if (!issue) {
			throw error(404, 'Issue not found');
		}
		
		return {
			issue,
			project
		};
	} catch (err) {
		console.error('Failed to load issue:', err);
		throw error(500, err.message || 'Failed to load issue');
	}
}