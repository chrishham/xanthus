<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Button from '../common/Button.svelte';
	import Card from '../common/Card.svelte';
	import type { VPS } from '../../../../app';

	export let vps: VPS;

	const dispatch = createEventDispatcher<{
		power: 'poweron' | 'poweroff' | 'reboot';
		delete: void;
		terminal: void;
		health: void;
		applications: void;
		ssh: void;
	}>();

	function getStatusColor(status: string): string {
		switch (status) {
			case 'running':
				return 'text-green-600';
			case 'stopped':
				return 'text-red-600';
			case 'starting':
			case 'stopping':
			case 'rebooting':
				return 'text-yellow-600';
			default:
				return 'text-gray-600';
		}
	}

	function getStatusBadgeColor(status: string): string {
		switch (status) {
			case 'running':
				return 'bg-green-100 text-green-800';
			case 'stopped':
				return 'bg-red-100 text-red-800';
			case 'starting':
			case 'stopping':
			case 'rebooting':
				return 'bg-yellow-100 text-yellow-800';
			default:
				return 'bg-gray-100 text-gray-800';
		}
	}

	function getProviderIcon(provider: string): string {
		switch (provider) {
			case 'hetzner':
				return '🔥'; // Hetzner icon
			case 'oracle':
				return '🔴'; // Oracle icon
			default:
				return '☁️';
		}
	}

	function formatCost(cost: number): string {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD',
			minimumFractionDigits: 2,
			maximumFractionDigits: 4
		}).format(cost);
	}

	function formatDate(dateString: string): string {
		try {
			return new Date(dateString).toLocaleDateString('en-US', {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		} catch {
			return dateString;
		}
	}

	function canPowerOn(): boolean {
		return vps.status === 'stopped';
	}

	function canPowerOff(): boolean {
		return vps.status === 'running';
	}

	function canReboot(): boolean {
		return vps.status === 'running';
	}

	function canTerminal(): boolean {
		return vps.status === 'running';
	}

	function isTransitioning(): boolean {
		return ['starting', 'stopping', 'rebooting'].includes(vps.status);
	}
</script>

<Card>
	<div class="p-6">
		<!-- Header -->
		<div class="flex items-start justify-between mb-4">
			<div class="flex items-center space-x-3">
				<span class="text-2xl">{getProviderIcon(vps.provider)}</span>
				<div>
					<h3 class="text-lg font-semibold text-gray-900">{vps.name}</h3>
					<p class="text-sm text-gray-500 capitalize">{vps.provider}</p>
				</div>
			</div>
			
			<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {getStatusBadgeColor(vps.status)}">
				{#if isTransitioning()}
					<svg class="animate-spin -ml-1 mr-2 h-3 w-3" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
				{/if}
				{vps.status}
			</span>
		</div>

		<!-- Server Details -->
		<div class="space-y-3 mb-6">
			<!-- Server Type -->
			<div class="flex justify-between items-center">
				<span class="text-sm text-gray-500">Server Type</span>
				<span class="text-sm font-medium text-gray-900">{vps.server_type?.name || 'N/A'}</span>
			</div>

			<!-- Region -->
			<div class="flex justify-between items-center">
				<span class="text-sm text-gray-500">Region</span>
				<span class="text-sm font-medium text-gray-900">{vps.labels?.region || vps.datacenter?.location?.description || 'N/A'}</span>
			</div>

			<!-- Public IP -->
			{#if vps.public_net?.ipv4?.ip || vps.labels?.ip_address}
				<div class="flex justify-between items-center">
					<span class="text-sm text-gray-500">Public IP</span>
					<span class="text-sm font-mono font-medium text-gray-900">{vps.public_net?.ipv4?.ip || vps.labels?.ip_address}</span>
				</div>
			{/if}

			<!-- Server Specs -->
			{#if vps.server_type?.cores || vps.server_type?.memory}
				<div class="flex justify-between items-center">
					<span class="text-sm text-gray-500">Specs</span>
					<span class="text-sm font-medium text-gray-900">
						{vps.server_type?.cores || 0} CPU, {vps.server_type?.memory || 0}GB RAM
					</span>
				</div>
			{/if}

			<!-- Cost Information -->
			{#if vps.labels?.monthly_cost}
				<div class="flex justify-between items-center">
					<span class="text-sm text-gray-500">Monthly Cost</span>
					<span class="text-sm font-medium text-gray-900">€{vps.labels.monthly_cost}</span>
				</div>
			{/if}

			{#if vps.labels?.hourly_cost}
				<div class="flex justify-between items-center">
					<span class="text-sm text-gray-500">Hourly Cost</span>
					<span class="text-sm font-medium text-gray-900">€{vps.labels.hourly_cost}</span>
				</div>
			{/if}

			<!-- Created Date -->
			{#if vps.created}
				<div class="flex justify-between items-center">
					<span class="text-sm text-gray-500">Created</span>
					<span class="text-sm font-medium text-gray-900">{formatDate(vps.created)}</span>
				</div>
			{/if}
		</div>

		<!-- Action Buttons -->
		<div class="space-y-3">
			<!-- Power Management -->
			<div class="flex space-x-2">
				{#if canPowerOn()}
					<Button
						variant="success"
						size="sm"
						class="flex-1"
						disabled={isTransitioning()}
						on:click={() => dispatch('power', 'poweron')}
					>
						<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.828 14.828a4 4 0 01-5.656 0M9 10h6m-3-7v2.5M12 19v2.5" />
						</svg>
						Start
					</Button>
				{/if}

				{#if canPowerOff()}
					<Button
						variant="danger"
						size="sm"
						class="flex-1"
						disabled={isTransitioning()}
						on:click={() => dispatch('power', 'poweroff')}
					>
						<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5.636 5.636l12.728 12.728m-12.728 0L18.364 5.636" />
						</svg>
						Stop
					</Button>
				{/if}

				{#if canReboot()}
					<Button
						variant="warning"
						size="sm"
						class="flex-1"
						disabled={isTransitioning()}
						on:click={() => dispatch('power', 'reboot')}
					>
						<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
						</svg>
						Reboot
					</Button>
				{/if}
			</div>

			<!-- Management Actions -->
			<div class="grid grid-cols-2 gap-2">
				<Button
					variant="secondary"
					size="sm"
					disabled={!canTerminal()}
					on:click={() => dispatch('terminal')}
				>
					<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
					</svg>
					Terminal
				</Button>

				<Button
					variant="secondary"
					size="sm"
					on:click={() => dispatch('health')}
				>
					<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
					</svg>
					Health
				</Button>

				<Button
					variant="secondary"
					size="sm"
					on:click={() => dispatch('applications')}
				>
					<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
					</svg>
					Apps
				</Button>

				<Button
					variant="secondary"
					size="sm"
					on:click={() => dispatch('ssh')}
				>
					<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 12H9v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2h2.764l4.893-4.893A6 6 0 0115 7z" />
					</svg>
					SSH
				</Button>
			</div>

			<!-- Delete Button -->
			<Button
				variant="danger"
				size="sm"
				class="w-full"
				disabled={isTransitioning()}
				on:click={() => dispatch('delete')}
			>
				<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
				</svg>
				Delete VPS
			</Button>
		</div>
	</div>
</Card>