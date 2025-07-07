import { errorHandler } from './errorHandler';

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

	async get<T>(endpoint: string, context = 'API', showErrors = true): Promise<T> {
		try {
			const response = await fetch(`${this.baseUrl}${endpoint}`, {
				method: 'GET',
				headers: {
					'Content-Type': 'application/json'
				}
			});

			return await this.handleResponse<T>(response);
		} catch (error) {
			if (showErrors) {
				await this.handleError(error, context);
			}
			throw error;
		}
	}

	async post<T>(endpoint: string, data?: unknown, context = 'API', showErrors = true): Promise<T> {
		try {
			const response = await fetch(`${this.baseUrl}${endpoint}`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: data ? JSON.stringify(data) : undefined
			});

			return await this.handleResponse<T>(response);
		} catch (error) {
			if (showErrors) {
				await this.handleError(error, context);
			}
			throw error;
		}
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

	private async handleError(error: any, context: string): Promise<void> {
		if (this.isNetworkError(error)) {
			await errorHandler.handleNetworkError(context);
		} else {
			await errorHandler.handleAPIError(error, context);
		}
	}

	private isNetworkError(error: any): boolean {
		// Network errors typically don't have a status code
		return !error.status && (
			error.message?.includes('NetworkError') ||
			error.message?.includes('Failed to fetch') ||
			error.message?.includes('fetch') ||
			error.name === 'TypeError'
		);
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