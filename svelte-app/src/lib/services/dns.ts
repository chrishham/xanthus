import { api } from './api';
import { authenticatedFetch } from '../stores/auth';
import type { Domain } from '../stores/dns';

export interface DNSConfigurationOptions {
	domain: string;
	enableHTTPS?: boolean;
	enableRedirect?: boolean;
	sslMode?: 'flexible' | 'full' | 'strict';
}

export interface DNSConfigurationResult {
	success: boolean;
	message: string;
	config?: {
		domain: string;
		ssl_mode: string;
		https_redirect: boolean;
		origin_certificate: string;
		private_key: string;
		created_at: string;
	};
}

export interface DNSRemovalResult {
	success: boolean;
	message: string;
}

export class DNSService {
	/**
	 * Fetch all domains from the API
	 */
	async fetchDomains(): Promise<Domain[]> {
		const response = await api.get('/dns', 'DNS Management');
		return response.domains || [];
	}

	/**
	 * Configure SSL for a domain
	 */
	async configureDomain(domain: string): Promise<DNSConfigurationResult> {
		try {
			const formData = new FormData();
			formData.append('domain', domain);
			
			const response = await authenticatedFetch('/api/dns/configure', {
				method: 'POST',
				body: formData
			});
			
			const data = await response.json();
			
			if (!response.ok) {
				throw new Error(data.error || 'Configuration failed');
			}
			
			return {
				success: true,
				message: data.message || 'SSL configuration completed successfully',
				config: data.config
			};
		} catch (error) {
			console.error('Error configuring domain:', error);
			throw new Error(error instanceof Error ? error.message : 'Failed to configure domain');
		}
	}

	/**
	 * Remove a domain from Xanthus management
	 */
	async removeDomain(domain: string): Promise<DNSRemovalResult> {
		try {
			const formData = new FormData();
			formData.append('domain', domain);
			
			const response = await authenticatedFetch('/api/dns/remove', {
				method: 'POST',
				body: formData
			});
			
			const data = await response.json();
			
			if (!response.ok) {
				throw new Error(data.error || 'Removal failed');
			}
			
			return {
				success: true,
				message: data.message || 'Domain configuration removed successfully'
			};
		} catch (error) {
			console.error('Error removing domain:', error);
			throw new Error(error instanceof Error ? error.message : 'Failed to remove domain');
		}
	}

	/**
	 * Get configuration details for a domain
	 */
	async getDomainConfiguration(domain: string): Promise<any> {
		try {
			const response = await api.get(`/dns/config/${domain}`);
			return response;
		} catch (error) {
			console.error('Error fetching domain configuration:', error);
			throw new Error('Failed to fetch domain configuration');
		}
	}

	/**
	 * Validate domain name format
	 */
	validateDomainName(domain: string): boolean {
		const domainRegex = /^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9](?:\.[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9])*$/;
		return domainRegex.test(domain);
	}

	/**
	 * Get status badge class for a domain status
	 */
	getStatusBadgeClass(status: string): string {
		switch (status.toLowerCase()) {
			case 'active':
				return 'bg-green-100 text-green-800';
			case 'pending':
				return 'bg-yellow-100 text-yellow-800';
			case 'error':
			case 'failed':
				return 'bg-red-100 text-red-800';
			case 'paused':
				return 'bg-orange-100 text-orange-800';
			default:
				return 'bg-gray-100 text-gray-800';
		}
	}

	/**
	 * Format date for display
	 */
	formatDate(dateString: string): string {
		try {
			const date = new Date(dateString);
			return date.toLocaleDateString('en-US', {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		} catch (error) {
			return dateString;
		}
	}

	/**
	 * Format date with time for display
	 */
	formatDateTime(dateString: string): string {
		try {
			const date = new Date(dateString);
			return date.toLocaleString('en-US', {
				year: 'numeric',
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch (error) {
			return dateString;
		}
	}

	/**
	 * Get domain type display name
	 */
	getDomainTypeDisplay(type: string): string {
		switch (type.toLowerCase()) {
			case 'full':
				return 'Full Zone';
			case 'partial':
				return 'Partial Zone';
			default:
				return type;
		}
	}

	/**
	 * Check if domain is ready for applications
	 */
	isDomainReady(domain: Domain): boolean {
		return domain.managed && domain.status === 'active' && !domain.paused;
	}

	/**
	 * Get domain management status text
	 */
	getManagementStatusText(domain: Domain): string {
		if (domain.managed) {
			return 'Managed by Xanthus';
		} else {
			return 'Not Managed';
		}
	}

	/**
	 * Get available actions for a domain
	 */
	getAvailableActions(domain: Domain): string[] {
		const actions = [];

		if (domain.managed) {
			actions.push('view-config');
			actions.push('remove');
		} else {
			actions.push('configure');
		}

		return actions;
	}
}

export const dnsService = new DNSService();