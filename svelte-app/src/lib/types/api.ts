export interface ApiResponse<T = any> {
	success: boolean;
	data?: T;
	error?: string;
	message?: string;
}

export interface PaginatedResponse<T> extends ApiResponse<T[]> {
	pagination: {
		page: number;
		limit: number;
		total: number;
		pages: number;
	};
}

export interface ApplicationCreationRequest {
	name: string;
	app_type: string;
	vps: string;
	subdomain: string;
	domain: string;
	description?: string;
	version: string;
}

export interface ApplicationUpgradeRequest {
	version: string;
}

export interface PasswordChangeRequest {
	new_password: string;
}

export interface PortForwardRequest {
	port: number;
	subdomain: string;
}

export interface PortForward {
	id: string;
	port: number;
	subdomain: string;
	url: string;
}

export interface VPSCreationRequest {
	name: string;
	provider: 'hetzner' | 'oracle';
	server_type: string;
	region: string;
	setup_k3s: boolean;
	ssh_key: string;
}

export interface PrerequisitesResponse {
	domains: Array<{
		name: string;
	}>;
	servers: Array<{
		id: string;
		name: string;
		public_net: {
			ipv4: {
				ip: string;
			};
		};
	}>;
}

export interface VersionInfo {
	version: string;
	is_latest: boolean;
	is_stable: boolean;
	published_at?: string;
}

export interface VersionsResponse extends ApiResponse {
	versions: VersionInfo[];
}

export interface AuthResponse extends ApiResponse {
	user?: {
		id: string;
		username: string;
		email?: string;
	};
	authenticated: boolean;
}

export interface LoginRequest {
	username: string;
	password: string;
}

export interface TokenResponse extends ApiResponse {
	token: string;
}

export interface PasswordResponse extends ApiResponse {
	password: string;
}

export interface ServerType {
	id: string;
	name: string;
	description: string;
	cores: number;
	memory: number;
	disk: number;
	prices: {
		monthly: number;
		hourly: number;
	};
}

export interface Region {
	id: string;
	name: string;
	description: string;
	location: string;
}

export interface SSHKey {
	id: string;
	name: string;
	fingerprint: string;
	public_key: string;
}

export interface ProvidersResponse extends ApiResponse {
	server_types: ServerType[];
	regions: Region[];
	ssh_keys: SSHKey[];
}