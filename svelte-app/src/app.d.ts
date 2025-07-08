// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};

// Type definitions for xanthus domain objects
export interface Application {
	id: string;
	name: string;
	status: 'running' | 'stopped' | 'pending' | 'error';
	url?: string;
	subdomain: string;
	port: number;
	created_at: string;
	updated_at: string;
}

export interface VPSServerType {
	name: string;
	description: string;
	cores: number;
	memory: number;
	disk: number;
	cpu_type: string;
}

export interface VPSDatacenter {
	location: {
		description: string;
	};
}

export interface VPSPublicNet {
	ipv4: {
		ip: string;
		blocked: boolean;
	};
}

export interface VPS {
	id: number;
	name: string;
	status: 'running' | 'stopped' | 'starting' | 'stopping' | 'rebooting' | 'unknown';
	provider: 'hetzner' | 'oracle';
	created: string;
	server_type: VPSServerType;
	datacenter: VPSDatacenter;
	public_net: VPSPublicNet;
	private_net?: any[];
	labels: {
		managed_by: string;
		accumulated_cost: string;
		monthly_cost: string;
		hourly_cost: string;
		provider: string;
		application_count: string;
		configured_timezone: string;
		region: string;
		ip_address: string;
	};
	// Computed properties for UI compatibility
	public_ip?: string;
	private_ip?: string;
	region?: string;
	monthly_cost?: number;
	hourly_cost?: number;
	created_at?: string;
}

export interface PredefinedApp {
	name: string;
	description: string;
	icon: string;
	category: string;
	default_port: number;
	template: string;
}