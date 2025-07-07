import { writable, derived } from 'svelte/store';

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

export const authStore = writable<AuthState>(initialState);

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

// Actions
export const setUser = (userData: User | null) => {
	authStore.update(state => ({
		...state,
		user: userData,
		isAuthenticated: !!userData,
		error: null
	}));
};

export const setTokens = (tokens: AuthTokens | null) => {
	authStore.update(state => ({
		...state,
		tokens
	}));
	
	if (tokens) {
		saveTokens(tokens);
	} else {
		clearTokens();
	}
};

export const setAuthLoading = (loading: boolean) => {
	authStore.update(state => ({
		...state,
		loading
	}));
};

export const setAuthError = (error: string | null) => {
	authStore.update(state => ({
		...state,
		error
	}));
};

export const login = async (cf_token: string) => {
	setAuthLoading(true);
	setAuthError(null);

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
			setTokens(tokens);
			
			// Get user profile after successful login
			await checkAuthStatus();
			return true;
		} else {
			const errorData = await response.json();
			setAuthError(errorData.error || 'Login failed');
			return false;
		}
	} catch (error) {
		setAuthError('Network error during login');
		return false;
	} finally {
		setAuthLoading(false);
	}
};

export const logout = async () => {
	setAuthLoading(true);
	
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
		setUser(null);
		setTokens(null);
		setAuthLoading(false);
		// Redirect to login page
		window.location.href = '/login';
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
			setTokens(newTokens);
			return true;
		} else {
			// Refresh failed, clear tokens and redirect to login
			setTokens(null);
			setUser(null);
			window.location.href = '/login';
			return false;
		}
	} catch (error) {
		console.error('Token refresh error:', error);
		return false;
	}
};

export const checkAuthStatus = async () => {
	setAuthLoading(true);
	
	try {
		// Check if we need to refresh the token
		if (shouldRefreshToken()) {
			const refreshed = await refreshTokens();
			if (!refreshed) {
				setUser(null);
				return;
			}
		}
		
		const tokens = getTokens();
		if (!tokens || isTokenExpired()) {
			setUser(null);
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
				setUser({
					id: data.user.id,
					account_id: data.user.account_id,
					namespace_id: data.user.namespace_id,
					isAuthenticated: true
				});
			} else {
				setUser(null);
			}
		} else {
			setUser(null);
		}
	} catch (error) {
		console.error('Auth check error:', error);
		setUser(null);
	} finally {
		setAuthLoading(false);
	}
};

// Auto-initialize auth store from stored tokens
export const initializeAuth = async () => {
	const tokens = getTokens();
	if (tokens) {
		setTokens(tokens);
		await checkAuthStatus();
	}
};

// Helper to make authenticated API requests
export const authenticatedFetch = async (url: string, options: RequestInit = {}): Promise<Response> => {
	// Check if we need to refresh the token
	if (shouldRefreshToken()) {
		await refreshTokens();
	}
	
	const tokens = getTokens();
	if (!tokens) {
		throw new Error('No authentication tokens available');
	}
	
	return fetch(url, {
		...options,
		headers: {
			...options.headers,
			'Authorization': `${tokens.token_type} ${tokens.access_token}`
		}
	});
};