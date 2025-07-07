import { writable, derived } from 'svelte/store';

export interface Domain {
	id: string;
	name: string;
	status: string;
	paused: boolean;
	type: string;
	managed: boolean;
	created_on: string;
	modified_on: string;
}

export interface DNSState {
	domains: Domain[];
	loading: boolean;
	error: string | null;
	filters: {
		managed: string; // 'all' | 'managed' | 'unmanaged'
		status: string; // 'all' | 'active' | 'pending' | 'error'
		search: string;
	};
	sort: {
		field: string;
		direction: 'asc' | 'desc';
	};
}

const initialState: DNSState = {
	domains: [],
	loading: false,
	error: null,
	filters: {
		managed: 'all',
		status: 'all',
		search: ''
	},
	sort: {
		field: 'name',
		direction: 'asc'
	}
};

export const dnsStore = writable<DNSState>(initialState);

// Derived store for filtered and sorted domains
export const filteredDomains = derived(dnsStore, ($dnsState) => {
	let filtered = [...$dnsState.domains];

	// Apply filters
	if ($dnsState.filters.managed !== 'all') {
		filtered = filtered.filter(domain => {
			if ($dnsState.filters.managed === 'managed') {
				return domain.managed;
			} else if ($dnsState.filters.managed === 'unmanaged') {
				return !domain.managed;
			}
			return true;
		});
	}

	if ($dnsState.filters.status !== 'all') {
		filtered = filtered.filter(domain => domain.status === $dnsState.filters.status);
	}

	if ($dnsState.filters.search) {
		const searchTerm = $dnsState.filters.search.toLowerCase();
		filtered = filtered.filter(domain => 
			domain.name.toLowerCase().includes(searchTerm) ||
			domain.status.toLowerCase().includes(searchTerm) ||
			domain.type.toLowerCase().includes(searchTerm)
		);
	}

	// Apply sorting
	filtered.sort((a, b) => {
		let aValue: any = a[$dnsState.sort.field as keyof Domain];
		let bValue: any = b[$dnsState.sort.field as keyof Domain];

		// Handle date fields
		if ($dnsState.sort.field === 'created_on' || $dnsState.sort.field === 'modified_on') {
			aValue = new Date(aValue);
			bValue = new Date(bValue);
		}

		// Handle string comparison
		if (typeof aValue === 'string' && typeof bValue === 'string') {
			aValue = aValue.toLowerCase();
			bValue = bValue.toLowerCase();
		}

		let comparison = 0;
		if (aValue > bValue) {
			comparison = 1;
		} else if (aValue < bValue) {
			comparison = -1;
		}

		return $dnsState.sort.direction === 'desc' ? -comparison : comparison;
	});

	return filtered;
});

// Action creators
export const setDomains = (domains: Domain[]) => {
	dnsStore.update(state => ({
		...state,
		domains,
		loading: false,
		error: null
	}));
};

export const setLoading = (loading: boolean) => {
	dnsStore.update(state => ({
		...state,
		loading
	}));
};

export const setError = (error: string | null) => {
	dnsStore.update(state => ({
		...state,
		error,
		loading: false
	}));
};

export const setManagedFilter = (managed: string) => {
	dnsStore.update(state => ({
		...state,
		filters: {
			...state.filters,
			managed
		}
	}));
};

export const setStatusFilter = (status: string) => {
	dnsStore.update(state => ({
		...state,
		filters: {
			...state.filters,
			status
		}
	}));
};

export const setSearchFilter = (search: string) => {
	dnsStore.update(state => ({
		...state,
		filters: {
			...state.filters,
			search
		}
	}));
};

export const setSortField = (field: string) => {
	dnsStore.update(state => {
		const newDirection = state.sort.field === field && state.sort.direction === 'asc' ? 'desc' : 'asc';
		return {
			...state,
			sort: {
				field,
				direction: newDirection
			}
		};
	});
};

export const clearFilters = () => {
	dnsStore.update(state => ({
		...state,
		filters: {
			managed: 'all',
			status: 'all',
			search: ''
		}
	}));
};

export const updateDomain = (updatedDomain: Domain) => {
	dnsStore.update(state => ({
		...state,
		domains: state.domains.map(domain => 
			domain.id === updatedDomain.id ? updatedDomain : domain
		)
	}));
};

export const removeDomain = (domainId: string) => {
	dnsStore.update(state => ({
		...state,
		domains: state.domains.filter(domain => domain.id !== domainId)
	}));
};