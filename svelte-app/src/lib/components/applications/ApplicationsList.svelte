<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { applicationStore, applications, predefinedApps, setAutoRefreshEnabled, showDeploymentModal, showPasswordModal } from '$lib/stores/applications';
	import { api } from '$lib/services/api';
	import { autoRefreshService } from '$lib/services/autoRefresh';
	import { formatDate } from '$lib/utils/formatting';
	import type { Application, PredefinedApp } from '../../../app';
	import Button from '$lib/components/common/Button.svelte';
	import Card from '$lib/components/common/Card.svelte';
	import DeploymentModal from './DeploymentModal.svelte';
	import PasswordModal from './PasswordModal.svelte';
	
	// Component props for initial data
	export let initialApplications: Application[] = [];
	export let initialPredefinedApps: PredefinedApp[] = [];
	
	// Reactive state from stores
	$: appList = $applications;
	$: predefinedAppList = $predefinedApps;
	$: ({ loading, autoRefresh } = $applicationStore);
	
	// Initialize component
	onMount(() => {
		// Set initial data if provided
		if (initialApplications.length > 0) {
			applicationStore.update(state => ({ ...state, applications: initialApplications }));
		}
		if (initialPredefinedApps.length > 0) {
			applicationStore.update(state => ({ ...state, predefinedApps: initialPredefinedApps }));
		}
		
		// Start initial data fetch if no initial data
		if (initialApplications.length === 0) {
			refreshApplications();
		}
		
		// Start auto-refresh service
		startAutoRefresh();
	});
	
	onDestroy(() => {
		// Clean up auto-refresh when component is destroyed
		autoRefreshService.stop();
	});
	
	// Application management functions
	async function refreshApplications() {
		applicationStore.update(state => ({ 
			...state, 
			loading: true,
			autoRefresh: { ...state.autoRefresh, isRefreshing: true }
		}));
		
		try {
			const data = await api.get('/applications/list');
			const validApps = (data.applications || []).filter(isValidApplication);
			const uniqueApps = deduplicateById(validApps);
			
			applicationStore.update(state => ({ 
				...state, 
				applications: uniqueApps,
				loading: false,
				autoRefresh: { ...state.autoRefresh, isRefreshing: false }
			}));
		} catch (error) {
			console.error('Error refreshing applications:', error);
			applicationStore.update(state => ({ 
				...state, 
				loading: false,
				autoRefresh: { ...state.autoRefresh, isRefreshing: false }
			}));
		}
	}
	
	function isValidApplication(app: any): app is Application {
		return app && app.id && app.name && app.status && app.url && app.created_at &&
			   app.name.trim() !== '' && app.url !== '';
	}
	
	function deduplicateById(apps: Application[]): Application[] {
		const uniqueApps: Application[] = [];
		const seenIds = new Set<string>();
		
		for (const app of apps) {
			if (!seenIds.has(app.id)) {
				seenIds.add(app.id);
				uniqueApps.push(app);
			}
		}
		
		return uniqueApps;
	}
	
	function getPredefinedAppIcon(appType: string): string {
		const app = predefinedAppList.find(a => a.id === appType);
		return app?.icon || '📦';
	}
	
	function getPredefinedAppName(appType: string): string {
		const app = predefinedAppList.find(a => a.id === appType);
		return app?.name || appType;
	}
	
	function getStatusBadgeClass(status: string): string {
		const normalizedStatus = status.toLowerCase();
		switch (normalizedStatus) {
			case 'running':
			case 'deployed':
				return 'bg-green-100 text-green-800';
			case 'deploying':
			case 'creating':
				return 'bg-blue-100 text-blue-800';
			case 'pending':
				return 'bg-yellow-100 text-yellow-800';
			case 'failed':
				return 'bg-red-100 text-red-800';
			case 'not deployed':
				return 'bg-gray-100 text-gray-800';
			default:
				return 'bg-gray-100 text-gray-800';
		}
	}
	
	function startAutoRefresh() {
		const refreshFn = async () => {
			try {
				const data = await api.get('/applications/list');
				const validApps = (data.applications || []).filter(isValidApplication);
				const uniqueApps = deduplicateById(validApps);
				
				applicationStore.update(state => ({
					...state,
					applications: uniqueApps,
					autoRefresh: {
						...state.autoRefresh,
						isRefreshing: false
					}
				}));
			} catch (error) {
				console.error('Auto-refresh failed:', error);
				applicationStore.update(state => ({
					...state,
					autoRefresh: {
						...state.autoRefresh,
						isRefreshing: false
					}
				}));
			}
		};

		if (autoRefresh.enabled) {
			autoRefreshService.start(refreshFn, 30000);
		}
	}

	function toggleAutoRefresh() {
		const newEnabled = !autoRefresh.enabled;
		setAutoRefreshEnabled(newEnabled);
		
		if (newEnabled) {
			startAutoRefresh();
		} else {
			autoRefreshService.stop();
		}
	}
	
	function visitApplication(app: Application) {
		window.open(app.url, '_blank');
	}

	async function deployApplication(predefinedApp: PredefinedApp) {
		try {
			// Check prerequisites first
			applicationStore.update(state => ({ ...state, loading: true }));
			
			const data = await api.get('/applications/prerequisites');
			const { domains, servers } = data;
			
			// Check if we have domains and servers
			if (!domains || domains.length === 0) {
				alert('No managed domains found. Please configure SSL for at least one domain first.');
				return;
			}
			
			if (!servers || servers.length === 0) {
				alert('No VPS instances found. Please create at least one VPS with K3s first.');
				return;
			}
			
			// Show deployment modal
			showDeploymentModal(predefinedApp, domains, servers);
			
		} catch (error) {
			console.error('Error checking prerequisites:', error);
			alert('Failed to load prerequisites. Please check your configuration.');
		} finally {
			applicationStore.update(state => ({ ...state, loading: false }));
		}
	}

	function handleDeploymentSuccess(event: CustomEvent) {
		// Refresh applications list after successful deployment
		refreshApplications();
	}

	function handleDeploymentError(event: CustomEvent) {
		console.error('Deployment failed:', event.detail);
		alert('Deployment failed. Please try again.');
	}

	function showCurrentPassword(app: Application) {
		showPasswordModal(app, 'view');
	}

	function showChangePassword(app: Application) {
		showPasswordModal(app, 'change');
	}

	function handlePasswordChanged(event: CustomEvent) {
		console.log('Password changed successfully for:', event.detail.app.name);
		// Could show a success toast here
	}

	function handlePasswordError(event: CustomEvent) {
		console.error('Password operation failed:', event.detail);
		alert('Password operation failed. Please try again.');
	}
</script>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<!-- Header -->
	<div class="mb-8">
		<h2 class="text-3xl font-bold text-gray-900 mb-2">Applications</h2>
		<p class="text-gray-600">Deploy and manage curated applications on your VPS servers</p>
	</div>

	<!-- Action Buttons -->
	<div class="flex justify-between items-center mb-6">
		<div class="flex space-x-3">
			<Button 
				variant="outline" 
				on:click={refreshApplications}
				disabled={loading}
				class="inline-flex items-center">
				<svg class="w-4 h-4 mr-2" class:animate-spin={loading} fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
				</svg>
				Refresh
			</Button>
			
			<Button 
				variant={autoRefresh.enabled ? "success" : "outline"}
				on:click={toggleAutoRefresh}
				disabled={loading}
				class="inline-flex items-center">
				<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path>
				</svg>
				{autoRefresh.enabled ? 'Auto-refresh ON' : 'Auto-refresh OFF'}
			</Button>
		</div>
		
		{#if autoRefresh.enabled}
			<div class="text-sm text-gray-500 flex items-center space-x-2">
				<svg class="w-4 h-4 text-green-500 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
					<circle cx="10" cy="10" r="3"></circle>
				</svg>
				<span>Auto-refresh: every 30 seconds</span>
				{#if autoRefresh.countdown > 0}
					<span class="text-xs bg-gray-200 px-2 py-1 rounded">Next: {autoRefresh.countdown}s</span>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Available Applications Catalog -->
	<div class="mb-12">
		<h3 class="text-xl font-semibold text-gray-900 mb-4">Available Applications</h3>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each predefinedAppList as app (app.id)}
				<Card class="hover:shadow-lg transition-shadow">
					<!-- Application Header -->
					<div class="p-6 border-b border-gray-200">
						<div class="flex items-center">
							<div class="text-3xl mr-4">{app.icon}</div>
							<div class="flex-1">
								<h4 class="text-lg font-medium text-gray-900">{app.name}</h4>
								<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
									{app.category}
								</span>
							</div>
						</div>
						<p class="text-sm text-gray-500 mt-3">{app.description}</p>
					</div>

					<!-- Application Details -->
					<div class="p-6">
						<div class="space-y-3">
							<!-- Version -->
							<div class="flex items-center justify-between">
								<span class="text-sm text-gray-500">Version:</span>
								<span class="text-sm font-medium text-gray-900">{app.version}</span>
							</div>
							
							<!-- Features -->
							<div>
								<span class="text-sm text-gray-500 block mb-2">Features:</span>
								<div class="space-y-1">
									{#each app.features.slice(0, 3) as feature}
										<div class="text-xs text-gray-600 flex items-center">
											<svg class="w-3 h-3 text-green-500 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
											</svg>
											<span>{feature}</span>
										</div>
									{/each}
									{#if app.features.length > 3}
										<div class="text-xs text-gray-400">
											<span>+{app.features.length - 3} more features</span>
										</div>
									{/if}
								</div>
							</div>
						</div>
					</div>

					<!-- Actions -->
					<div class="px-6 py-4 bg-gray-50 border-t border-gray-200 rounded-b-lg">
						<Button 
							variant="primary" 
							size="sm"
							class="w-full"
							on:click={() => deployApplication(app)}>
							Deploy Application
						</Button>
					</div>
				</Card>
			{/each}
		</div>
	</div>

	<!-- Deployed Applications -->
	<div>
		<h3 class="text-xl font-semibold text-gray-900 mb-4">Deployed Applications</h3>
		
		<!-- No Applications State -->
		{#if appList.length === 0 && !loading}
			<div class="text-center py-12 bg-white rounded-lg shadow-md">
				<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
				</svg>
				<h4 class="mt-2 text-lg font-medium text-gray-900">No applications deployed</h4>
				<p class="mt-1 text-gray-500">Deploy your first application from the catalog above.</p>
			</div>
		{/if}

		<!-- Applications Grid -->
		{#if appList.length > 0}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
				{#each appList as app (app.id)}
					<Card class="hover:shadow-lg transition-shadow">
						<!-- Application Header -->
						<div class="p-6 border-b border-gray-200">
							<div class="flex items-center justify-between">
								<div class="flex items-center">
									<div class="text-2xl mr-3">{getPredefinedAppIcon(app.app_type)}</div>
									<div>
										<h4 class="text-lg font-medium text-gray-900">{app.name}</h4>
										<span class="text-sm text-gray-500">{getPredefinedAppName(app.app_type)}</span>
									</div>
								</div>
								<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {getStatusBadgeClass(app.status)}">
									{app.status}
								</span>
							</div>
							<p class="text-sm text-gray-500 mt-2">{app.description || 'No description'}</p>
						</div>

						<!-- Application Details -->
						<div class="p-6">
							<div class="space-y-3">
								<!-- URL -->
								<div class="flex items-center justify-between">
									<span class="text-sm text-gray-500">URL:</span>
									<a href={app.url} 
									   target="_blank"
									   class="text-sm font-medium text-purple-600 hover:text-purple-800 truncate max-w-48">
										{app.url.replace('https://', '')}
									</a>
								</div>
								
								<!-- VPS -->
								<div class="flex items-center justify-between">
									<span class="text-sm text-gray-500">VPS:</span>
									<span class="text-sm font-medium text-gray-900">{app.vps_name}</span>
								</div>
								
								<!-- Version -->
								<div class="flex items-center justify-between">
									<span class="text-sm text-gray-500">Version:</span>
									<span class="text-sm font-medium text-gray-900">{app.app_version}</span>
								</div>
								
								<!-- Created -->
								<div class="flex items-center justify-between">
									<span class="text-sm text-gray-500">Created:</span>
									<span class="text-sm font-medium text-gray-900">{formatDate(app.created_at)}</span>
								</div>
							</div>
						</div>

						<!-- Actions -->
						<div class="px-6 py-4 bg-gray-50 border-t border-gray-200 rounded-b-lg">
							<div class="flex flex-wrap gap-2">
								<!-- Visit Application -->
								<Button 
									variant="primary" 
									size="xs"
									class="flex-1"
									on:click={() => visitApplication(app)}>
									Visit
								</Button>
								
								<!-- App-specific actions -->
								{#if app.app_type === 'code-server' || app.app_type === 'argocd'}
									<Button 
										variant="secondary" 
										size="xs" 
										class="flex-1"
										on:click={() => showCurrentPassword(app)}>
										👁️ Get Password
									</Button>
									<Button 
										variant="secondary" 
										size="xs" 
										class="flex-1"
										on:click={() => showChangePassword(app)}>
										🔑 Change Password
									</Button>
								{/if}
								
								{#if app.app_type === 'code-server'}
									<Button variant="secondary" size="xs" class="flex-1">
										🔗 Port Forwarding
									</Button>
								{/if}
								
								{#if app.app_type === 'headlamp'}
									<Button variant="secondary" size="xs" class="flex-1">
										🔐 Get Auth Token
									</Button>
								{/if}
								
								<Button variant="secondary" size="xs" class="flex-1">
									Change Version
								</Button>
								
								<Button variant="danger" size="xs" class="flex-1">
									Delete
								</Button>
							</div>
						</div>
					</Card>
				{/each}
			</div>
		{/if}
	</div>
</div>

<!-- Deployment Modal -->
<DeploymentModal 
	on:deployment-success={handleDeploymentSuccess}
	on:deployment-error={handleDeploymentError}
/>

<!-- Password Modal -->
<PasswordModal 
	on:password-changed={handlePasswordChanged}
	on:password-error={handlePasswordError}
/>