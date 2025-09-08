// TAKL API Client for Web Interface
// This module handles communication with the TAKL daemon via /api proxy

const API_BASE = '/api';

class TAKLApiError extends Error {
	constructor(message, status, response) {
		super(message);
		this.name = 'TAKLApiError';
		this.status = status;
		this.response = response;
	}
}

async function apiRequest(endpoint, options = {}) {
	const url = `${API_BASE}${endpoint}`;
	
	const defaultOptions = {
		headers: {
			'Content-Type': 'application/json',
		},
	};
	
	const config = { ...defaultOptions, ...options };
	
	if (config.body && typeof config.body === 'object') {
		config.body = JSON.stringify(config.body);
	}
	
	try {
		const response = await fetch(url, config);
		
		if (!response.ok) {
			const errorData = await response.json().catch(() => ({}));
			throw new TAKLApiError(
				errorData.error || `HTTP ${response.status}: ${response.statusText}`,
				response.status,
				errorData
			);
		}
		
		// Extract ETag and index status for response metadata
		const etag = response.headers.get('ETag');
		const indexStatus = response.headers.get('X-Index');
		
		const contentType = response.headers.get('content-type');
		let data;
		if (contentType && contentType.includes('application/json')) {
			data = await response.json();
		} else {
			data = await response.text();
		}
		
		// Attach response metadata if present
		if (etag || indexStatus) {
			if (typeof data === 'object' && data !== null) {
				data._meta = { etag, indexStatus };
			}
		}
		
		return data;
	} catch (error) {
		if (error instanceof TAKLApiError) {
			throw error;
		}
		
		// Network or other errors
		throw new TAKLApiError(
			`Network error: ${error.message}`,
			0,
			null
		);
	}
}

// Project API - Updated to match daemon routes
export const projectApi = {
	async list() {
		return await apiRequest('/registry/projects');
	},
	
	async get(projectId) {
		return await apiRequest(`/registry/projects/${projectId}`);
	},
	
	async register(payload) {
		return await apiRequest('/registry/projects', {
			method: 'POST',
			body: payload
		});
	},
	
	async health() {
		return await apiRequest('/registry/health');
	}
};

// Issues API
export const issuesApi = {
	async list(projectId, filters = {}) {
		const params = new URLSearchParams(filters);
		return await apiRequest(`/projects/${projectId}/issues?${params}`);
	},
	
	async get(projectId, issueId) {
		return await apiRequest(`/projects/${projectId}/issues/${issueId}`);
	},
	
	async create(projectId, issueData) {
		return await apiRequest(`/projects/${projectId}/issues`, {
			method: 'POST',
			body: issueData
		});
	},
	
	async update(projectId, issueId, updateData) {
		return await apiRequest(`/projects/${projectId}/issues/${issueId}`, {
			method: 'PUT',
			body: updateData
		});
	},
	
	async patch(projectId, issueId, patchData, etag) {
		const headers = {};
		if (etag) {
			headers['If-Match'] = etag;
		}
		
		return await apiRequest(`/projects/${projectId}/issues/${issueId}`, {
			method: 'PATCH',
			headers,
			body: patchData
		});
	},
	
	async delete(projectId, issueId) {
		return await apiRequest(`/projects/${projectId}/issues/${issueId}`, {
			method: 'DELETE'
		});
	},
	
	async transition(projectId, issueId, fromStatus, toStatus) {
		return await apiRequest(`/projects/${projectId}/issues/${issueId}/transition`, {
			method: 'POST',
			body: { from: fromStatus, to: toStatus }
		});
	}
};

// Templates API
export const templatesApi = {
	async list(projectId) {
		return await apiRequest(`/projects/${projectId}/templates`);
	},
	
	async get(projectId, templateName) {
		return await apiRequest(`/projects/${projectId}/templates/${templateName}`);
	},
	
	async render(projectId, templateName, values) {
		return await apiRequest(`/projects/${projectId}/templates/${templateName}/render`, {
			method: 'POST',
			body: { values }
		});
	}
};

// Paradigm API  
export const paradigmApi = {
	async list() {
		return await apiRequest('/paradigms');
	},
	
	async get(paradigmName) {
		return await apiRequest(`/paradigms/${paradigmName}`);
	},
	
	async operations(projectId, paradigmName, operationType) {
		return await apiRequest(`/projects/${projectId}/paradigm/${paradigmName}/${operationType}`);
	},
	
	async execute(projectId, operationName, args = {}) {
		return await apiRequest(`/projects/${projectId}/operations/${operationName}`, {
			method: 'POST',
			body: args
		});
	},
	
	async metrics(projectId) {
		return await apiRequest(`/projects/${projectId}/metrics`);
	}
};

// Dashboard API
export const dashboardApi = {
	async overview(projectId) {
		return await apiRequest(`/projects/${projectId}/dashboard`);
	},
	
	async stats(projectId, timeRange = '30d') {
		return await apiRequest(`/projects/${projectId}/stats?range=${timeRange}`);
	}
};

// Import/Export API
export const importApi = {
	async validateFile(projectId, fileData) {
		return await apiRequest(`/projects/${projectId}/import/validate`, {
			method: 'POST',
			body: fileData
		});
	},
	
	async import(projectId, fileData, options = {}) {
		return await apiRequest(`/projects/${projectId}/import`, {
			method: 'POST',
			body: { ...fileData, ...options }
		});
	},
	
	async export(projectId, format = 'json', filters = {}) {
		const params = new URLSearchParams({ format, ...filters });
		return await apiRequest(`/projects/${projectId}/export?${params}`);
	}
};

// Daemon API - Updated to match daemon routes  
export const daemonApi = {
	async status() {
		return await apiRequest('/../stats'); // Direct daemon stats
	},
	
	async health() {
		return await apiRequest('/../health'); // Direct daemon health
	},
	
	async proxyHealth() {
		return await fetch('/healthz').then(r => r.json()); // Proxy health check
	}
};

// Utility functions
export function isApiError(error) {
	return error instanceof TAKLApiError;
}

export function getErrorMessage(error) {
	if (error instanceof TAKLApiError) {
		return error.message;
	}
	return error.message || 'An unknown error occurred';
}

// ETag and concurrency control helpers
export function getETag(response) {
	return response?._meta?.etag;
}

export function isIndexStale(response) {
	return response?._meta?.indexStatus === 'stale';
}

export function isConflictError(error) {
	return error instanceof TAKLApiError && error.status === 409;
}

export function isVersionMismatch(error) {
	return isConflictError(error) && 
		   (error.message.includes('version mismatch') || error.message.includes('conflict'));
}

// Real-time connection for WebSocket updates (placeholder)
export class TAKLRealtimeClient {
	constructor(projectId) {
		this.projectId = projectId;
		this.ws = null;
		this.listeners = new Map();
		this.reconnectAttempts = 0;
		this.maxReconnectAttempts = 5;
	}
	
	connect() {
		const wsUrl = `ws://${window.location.host}/ws/projects/${this.projectId}`;
		
		this.ws = new WebSocket(wsUrl);
		
		this.ws.onopen = () => {
			console.log('TAKL WebSocket connected');
			this.reconnectAttempts = 0;
			this.emit('connected');
		};
		
		this.ws.onmessage = (event) => {
			try {
				const data = JSON.parse(event.data);
				this.emit(data.type, data.payload);
			} catch (error) {
				console.error('Failed to parse WebSocket message:', error);
			}
		};
		
		this.ws.onclose = () => {
			console.log('TAKL WebSocket disconnected');
			this.emit('disconnected');
			this.attemptReconnect();
		};
		
		this.ws.onerror = (error) => {
			console.error('TAKL WebSocket error:', error);
			this.emit('error', error);
		};
	}
	
	disconnect() {
		if (this.ws) {
			this.ws.close();
			this.ws = null;
		}
	}
	
	on(event, listener) {
		if (!this.listeners.has(event)) {
			this.listeners.set(event, []);
		}
		this.listeners.get(event).push(listener);
	}
	
	off(event, listener) {
		const eventListeners = this.listeners.get(event);
		if (eventListeners) {
			const index = eventListeners.indexOf(listener);
			if (index > -1) {
				eventListeners.splice(index, 1);
			}
		}
	}
	
	emit(event, data) {
		const eventListeners = this.listeners.get(event);
		if (eventListeners) {
			eventListeners.forEach(listener => {
				try {
					listener(data);
				} catch (error) {
					console.error('Error in WebSocket event listener:', error);
				}
			});
		}
	}
	
	attemptReconnect() {
		if (this.reconnectAttempts < this.maxReconnectAttempts) {
			const delay = Math.pow(2, this.reconnectAttempts) * 1000; // Exponential backoff
			
			setTimeout(() => {
				console.log(`Attempting to reconnect WebSocket (attempt ${this.reconnectAttempts + 1}/${this.maxReconnectAttempts})`);
				this.reconnectAttempts++;
				this.connect();
			}, delay);
		} else {
			console.error('Max WebSocket reconnection attempts reached');
			this.emit('reconnect-failed');
		}
	}
}

export { TAKLApiError };