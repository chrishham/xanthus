<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/services/api';
	import { autoRefreshService } from '$lib/services/autoRefresh';
	import { 
		servers, 
		autoRefreshState,
		setServers, 
		setVPSLoading, 
		setVPSError,
		setVPSAutoRefreshEnabled,
		setVPSRefreshing,
		showCreationModal,
		showTerminalModal,
		showHealthModal,
		showApplicationsModal,
		showSSHModal,
		updateServer,
		removeServer
	} from '$lib/stores/vps';
	import VPSCreationModal from '$lib/components/vps/VPSCreationModal.svelte';
	import VPSTerminalModal from '$lib/components/vps/VPSTerminalModal.svelte';
	import VPSHealthModal from '$lib/components/vps/VPSHealthModal.svelte';
	import VPSApplicationsModal from '$lib/components/vps/VPSApplicationsModal.svelte';
	import VPSSSHModal from '$lib/components/vps/VPSSSHModal.svelte';
	import VPSServerCard from '$lib/components/vps/VPSServerCard.svelte';
	import VPSFilters from '$lib/components/vps/VPSFilters.svelte';
	import LoadingModal from '$lib/components/common/LoadingModal.svelte';
	import Button from '$lib/components/common/Button.svelte';
	import type { VPS } from '../../../app';

	let autoRefreshInterval: number | null = null;

	// Reactive statements
	$: currentServers = $servers;
	$: autoRefresh = $autoRefreshState;
	$: hasServers = currentServers.length > 0;

	onMount(async () => {
		await loadVPSList();
		startAutoRefresh();
	});

	onDestroy(() => {
		stopAutoRefresh();
	});

	async function loadVPSList() {
		try {
			setVPSLoading(true);
			setVPSError(null);
			const response = await api.get<{ servers: VPS[] }>('/vps');
			setServers(response.servers);
		} catch (error) {
			console.error('Failed to load VPS list:', error);
			setVPSError(error instanceof Error ? error.message : 'Failed to load VPS list');
		} finally {
			setVPSLoading(false);
		}
	}

	function startAutoRefresh() {
		if (autoRefresh.enabled) {
			autoRefreshService.start(async () => {
				if (!document.hidden) {
					setVPSRefreshing(true);
					try {
						await loadVPSList();
					} finally {
						setVPSRefreshing(false);
					}
				}
			}, autoRefresh.interval);
		}
	}

	function stopAutoRefresh() {
		autoRefreshService.stop();
	}

	function toggleAutoRefresh() {
		const newEnabled = !autoRefresh.enabled;
		setVPSAutoRefreshEnabled(newEnabled);
		
		if (newEnabled) {
			startAutoRefresh();
		} else {
			stopAutoRefresh();
		}
	}

	async function handlePowerAction(vps: VPS, action: 'poweron' | 'poweroff' | 'reboot') {
		try {
			setVPSLoading(true);
			
			// Update server status optimistically
			updateServer(vps.id, { status: action === 'poweron' ? 'starting' : action === 'poweroff' ? 'stopping' : 'rebooting' });
			
			// Call the API endpoints that expect form data
			await api.post(`/vps/${action}`, { server_id: vps.id });
			
			// Refresh the server list after a short delay
			setTimeout(() => {
				loadVPSList();
			}, 2000);
			
		} catch (error) {
			console.error(`Failed to ${action} VPS:`, error);
			setVPSError(`Failed to ${action} VPS: ${error instanceof Error ? error.message : 'Unknown error'}`);
			// Revert optimistic update
			loadVPSList();
		} finally {
			setVPSLoading(false);
		}
	}

	async function handleDeleteVPS(vps: VPS) {
		const confirmed = confirm(`Are you sure you want to delete VPS "${vps.name}"? This action cannot be undone.`);
		if (!confirmed) return;

		try {
			setVPSLoading(true);
			
			if (vps.provider === 'oracle') {
				await api.post('/vps/oci/delete', { server_id: vps.id });
			} else {
				await api.post('/vps/delete', { server_id: vps.id });
			}
			
			// Remove from local state
			removeServer(vps.id);
			
		} catch (error) {
			console.error('Failed to delete VPS:', error);
			setVPSError(`Failed to delete VPS: ${error instanceof Error ? error.message : 'Unknown error'}`);
		} finally {
			setVPSLoading(false);
		}
	}

	function handleTerminal(vps: VPS) {
		showTerminalModal(vps);
	}

	function handleHealth(vps: VPS) {
		showHealthModal(vps);
	}

	function handleApplications(vps: VPS) {
		showApplicationsModal(vps);
	}

	function handleSSH(vps: VPS) {
		showSSHModal(vps);
	}

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
</script>

<svelte:head>
	<title>VPS Management - Xanthus</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
	<!-- Header -->
	<div class="bg-white shadow-sm border-b border-gray-200">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex justify-between items-center py-6">
				<div>
					<h1 class="text-3xl font-bold text-gray-900">VPS Management</h1>
					<p class="mt-1 text-sm text-gray-500">
						Manage your cloud servers and infrastructure
					</p>
				</div>
				<div class="flex items-center space-x-4">
					<!-- Auto-refresh controls -->
					<div class="flex items-center space-x-2">
						<button
							on:click={toggleAutoRefresh}
							class="inline-flex items-center px-3 py-2 text-sm rounded-md transition-colors
								{autoRefresh.enabled 
									? 'bg-green-100 text-green-800 hover:bg-green-200' 
									: 'bg-gray-100 text-gray-600 hover:bg-gray-200'}"
						>
							<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
									d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
							</svg>
							{autoRefresh.enabled ? `Auto-refresh (${autoRefresh.countdown}s)` : 'Auto-refresh off'}
						</button>
						
						<button
							on:click={loadVPSList}
							class="inline-flex items-center px-3 py-2 text-sm bg-gray-100 text-gray-600 rounded-md hover:bg-gray-200 transition-colors"
							disabled={autoRefresh.isRefreshing}
						>
							<svg class="h-4 w-4 mr-1 {autoRefresh.isRefreshing ? 'animate-spin' : ''}" 
								fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
									d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
							</svg>
							Refresh
						</button>
					</div>

					<!-- Create VPS button -->
					<Button variant="primary" on:click={showCreationModal}>
						<svg class="h-4 w-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
						</svg>
						Create VPS
					</Button>
				</div>
			</div>
		</div>
	</div>

	<!-- Main content -->
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
		<!-- Filters -->
		<VPSFilters />

		<!-- Servers grid -->
		{#if hasServers}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-8">
				{#each currentServers as vps (vps.id)}
					<VPSServerCard 
						{vps}
						on:power={(event) => handlePowerAction(vps, event.detail)}
						on:delete={() => handleDeleteVPS(vps)}
						on:terminal={() => handleTerminal(vps)}
						on:health={() => handleHealth(vps)}
						on:applications={() => handleApplications(vps)}
						on:ssh={() => handleSSH(vps)}
					/>
				{/each}
			</div>
		{:else}
			<!-- Empty state -->
			<div class="text-center py-12">
				<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
						d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
				</svg>
				<h3 class="mt-2 text-sm font-medium text-gray-900">No VPS servers</h3>
				<p class="mt-1 text-sm text-gray-500">Get started by creating your first VPS server.</p>
				<div class="mt-6">
					<Button variant="primary" on:click={showCreationModal}>
						<svg class="h-4 w-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
						</svg>
						Create your first VPS
					</Button>
				</div>
			</div>
		{/if}
	</div>
</div>

<!-- Modals -->
<VPSCreationModal />
<VPSTerminalModal />
<VPSHealthModal />
<VPSApplicationsModal />
<VPSSSHModal />
<LoadingModal />