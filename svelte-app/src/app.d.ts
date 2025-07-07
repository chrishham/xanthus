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

export interface VPS {
	id: string;
	name: string;
	provider: 'hetzner' | 'oracle';
	status: 'running' | 'stopped' | 'pending' | 'error';
	ip_address: string;
	created_at: string;
	updated_at: string;
}

export interface PredefinedApp {
	name: string;
	description: string;
	icon: string;
	category: string;
	default_port: number;
	template: string;
}