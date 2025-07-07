import { writable, derived } from 'svelte/store';
import type { VPS } from '../../app';

export interface VPSCreationModal {
	show: boolean;
	step: number;
	maxSteps: number;
	formData: {
		name: string;
		provider: 'hetzner' | 'oracle' | '';
		server_type: string;
		region: string;
		setup_k3s: boolean;
		ssh_key: string;
		// Hetzner specific
		hetzner_api_key?: string;
		location?: string;
		// OCI specific
		oci_config?: {
			user_id: string;
			tenancy_id: string;
			fingerprint: string;
			private_key: string;
			region: string;
		};
		instance_shape?: string;
		ocpu_count?: number;
		memory_gb?: number;
	};
	validationErrors: { [key: string]: string };
	loading: boolean;
}

export interface TerminalModal {
	show: boolean;
	vps: VPS | null;
	connected: boolean;
	sessionId?: string;
}

export interface HealthModal {
	show: boolean;
	vps: VPS | null;
	data: {
		k3s_status: string;
		uptime: string;
		memory_usage: string;
		disk_usage: string;
		load_average: string;
	} | null;
	loading: boolean;
}

export interface ApplicationsModal {
	show: boolean;
	vps: VPS | null;
	applications: any[];
	loading: boolean;
}

export interface SSHModal {
	show: boolean;
	vps: VPS | null;
}

export interface VPSState {
	servers: VPS[];
	loading: boolean;
	error: string | null;
	autoRefresh: {
		enabled: boolean;
		interval: number;
		countdown: number;
		isRefreshing: boolean;
		adaptivePolling: boolean;
	};
	modals: {
		creation: VPSCreationModal;
		terminal: TerminalModal;
		health: HealthModal;
		applications: ApplicationsModal;
		ssh: SSHModal;
	};
	filters: {
		provider: string;
		status: string;
		search: string;
	};
	sort: {
		field: string;
		direction: 'asc' | 'desc';
	};
}

const initialState: VPSState = {
	servers: [],
	loading: false,
	error: null,
	autoRefresh: {
		enabled: true,
		interval: 30000, // 30 seconds
		countdown: 0,
		isRefreshing: false,
		adaptivePolling: true
	},
	modals: {
		creation: {
			show: false,
			step: 1,
			maxSteps: 5, // Default to Hetzner flow
			formData: {
				name: '',
				provider: '',
				server_type: '',
				region: '',
				setup_k3s: true,
				ssh_key: ''
			},
			validationErrors: {},
			loading: false
		},
		terminal: {
			show: false,
			vps: null,
			connected: false
		},
		health: {
			show: false,
			vps: null,
			data: null,
			loading: false
		},
		applications: {
			show: false,
			vps: null,
			applications: [],
			loading: false
		},
		ssh: {
			show: false,
			vps: null
		}
	},
	filters: {
		provider: '',
		status: '',
		search: ''
	},
	sort: {
		field: 'name',
		direction: 'asc'
	}
};

export const vpsStore = writable<VPSState>(initialState);

// Derived stores
export const servers = derived(
	vpsStore,
	$store => {
		let filtered = $store.servers;
		
		// Apply filters
		if ($store.filters.provider) {
			filtered = filtered.filter(server => server.provider === $store.filters.provider);
		}
		if ($store.filters.status) {
			filtered = filtered.filter(server => server.status === $store.filters.status);
		}
		if ($store.filters.search) {
			const search = $store.filters.search.toLowerCase();
			filtered = filtered.filter(server => 
				server.name.toLowerCase().includes(search) ||
				server.public_ip?.toLowerCase().includes(search) ||
				server.private_ip?.toLowerCase().includes(search)
			);
		}
		
		// Apply sorting
		return filtered.sort((a, b) => {
			const aValue = a[$store.sort.field as keyof VPS] || '';
			const bValue = b[$store.sort.field as keyof VPS] || '';
			const direction = $store.sort.direction === 'asc' ? 1 : -1;
			
			if (aValue < bValue) return -1 * direction;
			if (aValue > bValue) return 1 * direction;
			return 0;
		});
	}
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

export const healthModal = derived(
	vpsStore,
	$store => $store.modals.health
);

export const applicationsModal = derived(
	vpsStore,
	$store => $store.modals.applications
);

export const sshModal = derived(
	vpsStore,
	$store => $store.modals.ssh
);

// Basic VPS actions
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

export const setVPSError = (error: string | null) => {
	vpsStore.update(state => ({
		...state,
		error
	}));
};

// Auto-refresh actions
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

export const setVPSRefreshInterval = (interval: number) => {
	vpsStore.update(state => ({
		...state,
		autoRefresh: {
			...state.autoRefresh,
			interval
		}
	}));
};

// Filter and sort actions
export const setVPSFilters = (filters: Partial<VPSState['filters']>) => {
	vpsStore.update(state => ({
		...state,
		filters: {
			...state.filters,
			...filters
		}
	}));
};

export const setVPSSort = (field: string, direction: 'asc' | 'desc') => {
	vpsStore.update(state => ({
		...state,
		sort: {
			field,
			direction
		}
	}));
};

// Creation modal actions
export const showCreationModal = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				show: true,
				step: 1,
				validationErrors: {}
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
				show: false,
				loading: false,
				validationErrors: {}
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

export const nextCreationStep = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				step: Math.min(state.modals.creation.step + 1, state.modals.creation.maxSteps)
			}
		}
	}));
};

export const prevCreationStep = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				step: Math.max(state.modals.creation.step - 1, 1)
			}
		}
	}));
};

export const updateCreationFormData = (updates: Partial<VPSCreationModal['formData']>) => {
	vpsStore.update(state => {
		const newFormData = {
			...state.modals.creation.formData,
			...updates
		};
		
		// Update maxSteps based on provider
		let maxSteps = 5; // Default Hetzner
		if (newFormData.provider === 'oracle') {
			maxSteps = 3;
		}
		
		return {
			...state,
			modals: {
				...state.modals,
				creation: {
					...state.modals.creation,
					formData: newFormData,
					maxSteps
				}
			}
		};
	});
};

export const setCreationValidationErrors = (errors: { [key: string]: string }) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				validationErrors: errors
			}
		}
	}));
};

export const setCreationLoading = (loading: boolean) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			creation: {
				...state.modals.creation,
				loading
			}
		}
	}));
};

// Terminal modal actions
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
				connected: false,
				sessionId: undefined
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

export const setTerminalSessionId = (sessionId: string) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			terminal: {
				...state.modals.terminal,
				sessionId
			}
		}
	}));
};

// Health modal actions
export const showHealthModal = (vps: VPS) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			health: {
				show: true,
				vps,
				data: null,
				loading: true
			}
		}
	}));
};

export const hideHealthModal = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			health: {
				...state.modals.health,
				show: false,
				loading: false
			}
		}
	}));
};

export const setHealthData = (data: HealthModal['data']) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			health: {
				...state.modals.health,
				data,
				loading: false
			}
		}
	}));
};

export const setHealthLoading = (loading: boolean) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			health: {
				...state.modals.health,
				loading
			}
		}
	}));
};

// Applications modal actions
export const showApplicationsModal = (vps: VPS) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			applications: {
				show: true,
				vps,
				applications: [],
				loading: true
			}
		}
	}));
};

export const hideApplicationsModal = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			applications: {
				...state.modals.applications,
				show: false,
				loading: false
			}
		}
	}));
};

export const setApplicationsData = (applications: any[]) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			applications: {
				...state.modals.applications,
				applications,
				loading: false
			}
		}
	}));
};

export const setApplicationsLoading = (loading: boolean) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			applications: {
				...state.modals.applications,
				loading
			}
		}
	}));
};

// SSH modal actions
export const showSSHModal = (vps: VPS) => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			ssh: {
				show: true,
				vps
			}
		}
	}));
};

export const hideSSHModal = () => {
	vpsStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			ssh: {
				...state.modals.ssh,
				show: false
			}
		}
	}));
};