import { writable, derived } from 'svelte/store';
import type { VPS } from '../../app';

export interface VPSCreationModal {
	show: boolean;
	step: number;
	formData: {
		name: string;
		provider: 'hetzner' | 'oracle' | '';
		server_type: string;
		region: string;
		setup_k3s: boolean;
		ssh_key: string;
	};
}

export interface TerminalModal {
	show: boolean;
	vps: VPS | null;
	connected: boolean;
}

export interface VPSState {
	servers: VPS[];
	loading: boolean;
	autoRefresh: {
		enabled: boolean;
		interval: number;
		countdown: number;
		isRefreshing: boolean;
	};
	modals: {
		creation: VPSCreationModal;
		terminal: TerminalModal;
	};
}

const initialState: VPSState = {
	servers: [],
	loading: false,
	autoRefresh: {
		enabled: true,
		interval: 30000, // 30 seconds
		countdown: 0,
		isRefreshing: false
	},
	modals: {
		creation: {
			show: false,
			step: 1,
			formData: {
				name: '',
				provider: '',
				server_type: '',
				region: '',
				setup_k3s: true,
				ssh_key: ''
			}
		},
		terminal: {
			show: false,
			vps: null,
			connected: false
		}
	}
};

export const vpsStore = writable<VPSState>(initialState);

// Derived stores
export const servers = derived(
	vpsStore,
	$store => $store.servers
);

export const autoRefreshState = derived(
	vpsStore,
	$store => $store.autoRefresh
);

export const creationModal = derived(
	vpsStore,
	$store => $store.modals.creation
);

export const terminalModal = derived(
	vpsStore,
	$store => $store.modals.terminal
);

// Actions
export const setServers = (serverList: VPS[]) => {
	vpsStore.update(state => ({
		...state,
		servers: serverList
	}));
};

export const addServer = (server: VPS) => {
	vpsStore.update(state => ({
		...state,
		servers: [...state.servers, server]
	}));
};

export const updateServer = (serverId: string, updates: Partial<VPS>) => {
	vpsStore.update(state => ({
		...state,
		servers: state.servers.map(server =>
			server.id === serverId ? { ...server, ...updates } : server
		)
	}));
};

export const removeServer = (serverId: string) => {
	vpsStore.update(state => ({
		...state,
		servers: state.servers.filter(server => server.id !== serverId)
	}));
};

export const setVPSLoading = (loading: boolean) => {
	vpsStore.update(state => ({
		...state,
		loading
	}));
};

export const setVPSAutoRefreshEnabled = (enabled: boolean) => {
	vpsStore.update(state => ({
		...state,
		autoRefresh: {
			...state.autoRefresh,
			enabled
		}
	}));
};

export const setVPSAutoRefreshCountdown = (countdown: number) => {
	vpsStore.update(state => ({
		...state,
		autoRefresh: {
			...state.autoRefresh,
			countdown
		}
	}));
};

export const setVPSRefreshing = (isRefreshing: boolean) => {
	vpsStore.update(state => ({
		...state,
		autoRefresh: {
			...state.autoRefresh,
			isRefreshing
		}
	}));
};

// Modal actions
export const showCreationModal = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				show: true,
				step: 1
			}
		}
	}));
};

export const hideCreationModal = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				show: false
			}
		}
	}));
};

export const setCreationStep = (step: number) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				step
			}
		}
	}));
};

export const updateCreationFormData = (updates: Partial<VPSCreationModal['formData']>) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				formData: {
					...state.modals.creation.formData,
					...updates
				}
			}
		}
	}));
};

export const showTerminalModal = (vps: VPS) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			terminal: {
				show: true,
				vps,
				connected: false
			}
		}
	}));
};

export const hideTerminalModal = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			terminal: {
				...state.modals.terminal,
				show: false,
				connected: false
			}
		}
	}));
};

export const setTerminalConnected = (connected: boolean) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			terminal: {
				...state.modals.terminal,
				connected
			}
		}
	}));
};