import { writable, derived } from 'svelte/store';
import type { Application, PredefinedApp } from '../../app';

export interface AutoRefreshState {
	enabled: boolean;
	interval: number;
	countdown: number;
	isRefreshing: boolean;
}

export interface DeploymentModal {
	show: boolean;
	predefinedApp: PredefinedApp | null;
	domains: Array<{ name: string }>;
	servers: Array<{ id: string; name: string; public_net: { ipv4: { ip: string } } }>;
}

export interface PasswordModal {
	show: boolean;
	app: Application | null;
	mode: 'view' | 'change';
}

export interface PortForwardingModal {
	show: boolean;
	app: Application | null;
	domain: string;
	ports: Array<{ id: string; port: number; subdomain: string; url: string }>;
	newPort: { port: string; subdomain: string };
}

export interface ApplicationState {
	applications: Application[];
	predefinedApps: PredefinedApp[];
	loading: boolean;
	autoRefresh: AutoRefreshState;
	modals: {
		deployment: DeploymentModal;
		password: PasswordModal;
		portForwarding: PortForwardingModal;
	};
}

const initialState: ApplicationState = {
	applications: [],
	predefinedApps: [],
	loading: false,
	autoRefresh: {
		enabled: true,
		interval: 30000, // 30 seconds
		countdown: 0,
		isRefreshing: false
	},
	modals: {
		deployment: {
			show: false,
			predefinedApp: null,
			domains: [],
			servers: []
		},
		password: {
			show: false,
			app: null,
			mode: 'view'
		},
		portForwarding: {
			show: false,
			app: null,
			domain: '',
			ports: [],
			newPort: { port: '', subdomain: '' }
		}
	}
};

export const applicationStore = writable<ApplicationState>(initialState);

// Derived stores for easy access
export const applications = derived(
	applicationStore,
	$store => $store.applications
);

export const predefinedApps = derived(
	applicationStore,
	$store => $store.predefinedApps
);

export const autoRefreshState = derived(
	applicationStore,
	$store => $store.autoRefresh
);

// Actions
export const setApplications = (apps: Application[]) => {
	applicationStore.update(state => ({
		...state,
		applications: apps
	}));
};

export const setPredefinedApps = (apps: PredefinedApp[]) => {
	applicationStore.update(state => ({
		...state,
		predefinedApps: apps
	}));
};

export const setAutoRefreshEnabled = (enabled: boolean) => {
	applicationStore.update(state => ({
		...state,
		autoRefresh: {
			...state.autoRefresh,
			enabled
		}
	}));
};

export const setAutoRefreshCountdown = (countdown: number) => {
	applicationStore.update(state => ({
		...state,
		autoRefresh: {
			...state.autoRefresh,
			countdown
		}
	}));
};

export const setRefreshing = (isRefreshing: boolean) => {
	applicationStore.update(state => ({
		...state,
		autoRefresh: {
			...state.autoRefresh,
			isRefreshing
		}
	}));
};

export const showDeploymentModal = (predefinedApp: PredefinedApp, domains: Array<{ name: string }>, servers: Array<{ id: string; name: string; public_net: { ipv4: { ip: string } } }>) => {
	applicationStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			deployment: {
				show: true,
				predefinedApp,
				domains,
				servers
			}
		}
	}));
};

export const hideDeploymentModal = () => {
	applicationStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			deployment: {
				...state.modals.deployment,
				show: false
			}
		}
	}));
};

export const showPasswordModal = (app: Application, mode: 'view' | 'change' = 'view') => {
	applicationStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			password: {
				show: true,
				app,
				mode
			}
		}
	}));
};

export const hidePasswordModal = () => {
	applicationStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			password: {
				...state.modals.password,
				show: false
			}
		}
	}));
};

export const showPortForwardingModal = (app: Application) => {
	applicationStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			portForwarding: {
				show: true,
				app,
				domain: extractDomain(app.url || ''),
				ports: [],
				newPort: { port: '', subdomain: '' }
			}
		}
	}));
};

export const hidePortForwardingModal = () => {
	applicationStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			portForwarding: {
				...state.modals.portForwarding,
				show: false
			}
		}
	}));
};

// Helper functions
export const isValidApplication = (app: any): app is Application => {
	return app &&
		app.id &&
		app.name &&
		app.status &&
		app.url &&
		app.created_at &&
		app.name.trim() !== '' &&
		app.url !== '';
};

export const extractDomain = (url: string): string => {
	try {
		const urlObj = new URL(url);
		const parts = urlObj.hostname.split('.');
		return parts.slice(1).join('.');
	} catch (error) {
		console.error('Error extracting domain from URL:', url, error);
		return '';
	}
};

export const formatDate = (dateString: string): string => {
	return new Date(dateString).toLocaleDateString('en-US', {
		year: 'numeric',
		month: 'short',
		day: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
};