import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';

export interface User {
	id: string;
	account_id: string;
	namespace_id?: string;
	isAuthenticated: boolean;
}

export interface AuthTokens {
	access_token: string;
	refresh_token: string;
	token_type: string;
	expires_in: number;
}

export interface AuthState {
	user: User | null;
	tokens: AuthTokens | null;
	loading: boolean;
	error: string | null;
	isAuthenticated: boolean;
}

const initialState: AuthState = {
	user: null,
	tokens: null,
	loading: false,
	error: null,
	isAuthenticated: false
};

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>(initialState);

	return {
		subscribe,
		set,
		update,
		// Actions
		login: async (cf_token: string) => {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				const response = await fetch('/api/auth/login', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify({ cf_token })
				});

				if (response.ok) {
					const tokens: AuthTokens = await response.json();
					saveTokens(tokens);
					
					// Get user profile after successful login
					await checkAuthStatus();
					return true;
				} else {
					const errorData = await response.json();
					update(state => ({ ...state, error: errorData.error || 'Login failed' }));
					return false;
				}
			} catch (error) {
				update(state => ({ ...state, error: 'Network error during login' }));
				return false;
			} finally {
				update(state => ({ ...state, loading: false }));
			}
		},
		logout: async () => {
			update(state => ({ ...state, loading: true }));
			
			try {
				const tokens = getTokens();
				if (tokens) {
					await fetch('/api/auth/logout', {
						method: 'POST',
						headers: {
							'Authorization': `${tokens.token_type} ${tokens.access_token}`,
							'Content-Type': 'application/json'
						}
					});
				}
			} catch (error) {
				console.error('Logout error:', error);
			} finally {
				set(initialState);
				clearTokens();
				// Redirect to login page
				window.location.href = '/login';
			}
		},
		initialize: async () => {
			const tokens = getTokens();
			if (tokens) {
				await checkAuthStatus();
			}
		}
	};
}

export const authStore = createAuthStore();

// Derived stores
export const user = derived(
	authStore,
	$store => $store.user
);

export const isAuthenticated = derived(
	authStore,
	$store => $store.isAuthenticated
);

export const authLoading = derived(
	authStore,
	$store => $store.loading
);

export const authError = derived(
	authStore,
	$store => $store.error
);

// Token management
const TOKEN_STORAGE_KEY = 'xanthus_tokens';
const REFRESH_THRESHOLD = 5 * 60 * 1000; // 5 minutes before expiry

export const saveTokens = (tokens: AuthTokens) => {
	localStorage.setItem(TOKEN_STORAGE_KEY, JSON.stringify({
		...tokens,
		expires_at: Date.now() + (tokens.expires_in * 1000)
	}));
};

export const getTokens = (): AuthTokens | null => {
	try {
		const stored = localStorage.getItem(TOKEN_STORAGE_KEY);
		if (!stored) return null;
		
		const tokens = JSON.parse(stored);
		return {
			access_token: tokens.access_token,
			refresh_token: tokens.refresh_token,
			token_type: tokens.token_type,
			expires_in: tokens.expires_in
		};
	} catch {
		return null;
	}
};

export const clearTokens = () => {
	localStorage.removeItem(TOKEN_STORAGE_KEY);
};

export const isTokenExpired = (): boolean => {
	try {
		const stored = localStorage.getItem(TOKEN_STORAGE_KEY);
		if (!stored) return true;
		
		const tokens = JSON.parse(stored);
		return Date.now() >= tokens.expires_at;
	} catch {
		return true;
	}
};

export const shouldRefreshToken = (): boolean => {
	try {
		const stored = localStorage.getItem(TOKEN_STORAGE_KEY);
		if (!stored) return false;
		
		const tokens = JSON.parse(stored);
		return Date.now() >= (tokens.expires_at - REFRESH_THRESHOLD);
	} catch {
		return false;
	}
};

// Helper function for auth status check
export const checkAuthStatus = async () => {
	authStore.update(state => ({ ...state, loading: true }));
	
	try {
		// Check if we need to refresh the token
		if (shouldRefreshToken()) {
			const refreshed = await refreshTokens();
			if (!refreshed) {
				authStore.update(state => ({ ...state, user: null, isAuthenticated: false, loading: false }));
				return;
			}
		}
		
		const tokens = getTokens();
		if (!tokens || isTokenExpired()) {
			authStore.update(state => ({ ...state, user: null, isAuthenticated: false, loading: false }));
			return;
		}

		const response = await fetch('/api/user/profile', {
			headers: {
				'Authorization': `${tokens.token_type} ${tokens.access_token}`
			}
		});
		
		if (response.ok) {
			const data = await response.json();
			if (data.authenticated && data.user) {
				authStore.update(state => ({ 
					...state, 
					user: {
						id: data.user.id,
						account_id: data.user.account_id,
						namespace_id: data.user.namespace_id,
						isAuthenticated: true
					},
					isAuthenticated: true,
					loading: false
				}));
			} else {
				authStore.update(state => ({ ...state, user: null, isAuthenticated: false, loading: false }));
			}
		} else {
			authStore.update(state => ({ ...state, user: null, isAuthenticated: false, loading: false }));
		}
	} catch (error) {
		console.error('Auth check error:', error);
		authStore.update(state => ({ ...state, user: null, isAuthenticated: false, loading: false }));
	}
};

export const refreshTokens = async (): Promise<boolean> => {
	const currentTokens = getTokens();
	if (!currentTokens?.refresh_token) {
		return false;
	}

	try {
		const response = await fetch('/api/auth/refresh', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({ refresh_token: currentTokens.refresh_token })
		});

		if (response.ok) {
			const newTokens: AuthTokens = await response.json();
			saveTokens(newTokens);
			return true;
		} else {
			// Refresh failed, clear tokens and redirect to login
			clearTokens();
			authStore.set(initialState);
			window.location.href = '/login';
			return false;
		}
	} catch (error) {
		console.error('Token refresh error:', error);
		return false;
	}
};

// Helper to make authenticated API requests
export const authenticatedFetch = async (url: string, options: RequestInit = {}): Promise<Response> => {
	// Check if we need to refresh the token
	if (shouldRefreshToken()) {
		const refreshed = await refreshTokens();
		if (!refreshed) {
			// Redirect to login if refresh failed
			if (browser) {
				window.location.href = '/login';
			}
			throw new Error('Authentication required');
		}
	}
	
	const tokens = getTokens();
	if (!tokens) {
		// Redirect to login if no tokens
		if (browser) {
			window.location.href = '/login';
		}
		throw new Error('Authentication required');
	}
	
	return fetch(url, {
		...options,
		headers: {
			...options.headers,
			'Authorization': `${tokens.token_type} ${tokens.access_token}`
		}
	});
};