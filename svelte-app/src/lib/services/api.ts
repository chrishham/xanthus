export class ApiError extends Error {
	constructor(
		message: string,
		public status: number,
		public response?: Response
	) {
		super(message);
		this.name = 'ApiError';
	}
}

export class ApiClient {
	private baseUrl = '/api';

	async get<T>(endpoint: string): Promise<T> {
		const response = await fetch(`${this.baseUrl}${endpoint}`, {
			method: 'GET',
			headers: {
				'Content-Type': 'application/json'
			}
		});

		return this.handleResponse<T>(response);
	}

	async post<T>(endpoint: string, data?: unknown): Promise<T> {
		const response = await fetch(`${this.baseUrl}${endpoint}`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: data ? JSON.stringify(data) : undefined
		});

		return this.handleResponse<T>(response);
	}

	async put<T>(endpoint: string, data?: unknown): Promise<T> {
		const response = await fetch(`${this.baseUrl}${endpoint}`, {
			method: 'PUT',
			headers: {
				'Content-Type': 'application/json'
			},
			body: data ? JSON.stringify(data) : undefined
		});

		return this.handleResponse<T>(response);
	}

	async delete<T>(endpoint: string): Promise<T> {
		const response = await fetch(`${this.baseUrl}${endpoint}`, {
			method: 'DELETE',
			headers: {
				'Content-Type': 'application/json'
			}
		});

		return this.handleResponse<T>(response);
	}

	private async handleResponse<T>(response: Response): Promise<T> {
		if (!response.ok) {
			let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
			
			try {
				const errorData = await response.json();
				errorMessage = errorData.error || errorData.message || errorMessage;
			} catch {
				// If JSON parsing fails, use the default error message
			}

			throw new ApiError(errorMessage, response.status, response);
		}

		// Handle empty responses
		const contentType = response.headers.get('Content-Type');
		if (!contentType || !contentType.includes('application/json')) {
			return {} as T;
		}

		try {
			return await response.json();
		} catch {
			return {} as T;
		}
	}

	// Compatibility methods for existing API endpoints that don't use /api prefix
	async legacyGet<T>(endpoint: string): Promise<T> {
		const response = await fetch(endpoint, {
			method: 'GET',
			headers: {
				'Content-Type': 'application/json'
			}
		});

		return this.handleResponse<T>(response);
	}

	async legacyPost<T>(endpoint: string, data?: unknown): Promise<T> {
		const response = await fetch(endpoint, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: data ? JSON.stringify(data) : undefined
		});

		return this.handleResponse<T>(response);
	}

	async legacyDelete<T>(endpoint: string): Promise<T> {
		const response = await fetch(endpoint, {
			method: 'DELETE',
			headers: {
				'Content-Type': 'application/json'
			}
		});

		return this.handleResponse<T>(response);
	}
}

export const api = new ApiClient();