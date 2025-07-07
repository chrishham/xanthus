import { writable, derived } from 'svelte/store';

export interface User {
	id: string;
	username: string;
	email?: string;
	isAuthenticated: boolean;
}

export interface AuthState {
	user: User | null;
	loading: boolean;
	error: string | null;
	isAuthenticated: boolean;
}

const initialState: AuthState = {
	user: null,
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

// Actions
export const setUser = (userData: User | null) => {
	authStore.update(state => ({
		...state,
		user: userData,
		isAuthenticated: !!userData,
		error: null
	}));
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

export const login = async (credentials: { username: string; password: string }) => {
	setAuthLoading(true);
	setAuthError(null);

	try {
		const response = await fetch('/login', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify(credentials)
		});

		if (response.ok) {
			// Assuming login success redirects or returns user data
			const userData = await response.json();
			setUser(userData.user || {
				id: '1',
				username: credentials.username,
				isAuthenticated: true
			});
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
		await fetch('/logout', {
			method: 'POST'
		});
		
		setUser(null);
		// Redirect to login page or handle logout
		window.location.href = '/login';
	} catch (error) {
		console.error('Logout error:', error);
		// Still clear user data on error
		setUser(null);
	} finally {
		setAuthLoading(false);
	}
};

export const checkAuthStatus = async () => {
	setAuthLoading(true);
	
	try {
		const response = await fetch('/auth/status');
		
		if (response.ok) {
			const data = await response.json();
			if (data.authenticated) {
				setUser(data.user || {
					id: '1',
					username: 'user',
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