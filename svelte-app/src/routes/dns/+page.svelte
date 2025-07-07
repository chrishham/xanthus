<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { dnsStore, filteredDomains, setDomains, setLoading, setError, type Domain } from '$lib/stores/dns';
	import { dnsService } from '$lib/services/dns';
	import { setLoading as setGlobalLoading, clearLoading } from '$lib/stores/ui';
	import Card from '$lib/components/common/Card.svelte';
	import Button from '$lib/components/common/Button.svelte';

	// Reactive subscriptions
	$: ({ loading, error } = $dnsStore);
	$: domains = $filteredDomains;

	onMount(async () => {
		await loadDomains();
	});

	async function loadDomains() {
		setLoading(true);
		setError(null);
		try {
			const domainsList = await dnsService.fetchDomains();
			setDomains(domainsList);
		} catch (err) {
			// Error handling is now done by the errorHandler service
			setError('Failed to load domains');
		}
	}

	async function configureDomain(domain: Domain) {
		if (!browser) return;

		try {
			const { default: Swal } = await import('sweetalert2');
			
			const result = await Swal.fire({
				title: `Configure SSL for ${domain.name}`,
				html: `
					<div class="text-left">
						<p class="mb-4">This will automatically configure:</p>
						<ul class="list-disc list-inside space-y-2 text-sm">
							<li>SSL/TLS mode to Full (strict)</li>
							<li>Create Origin Server Certificate</li>
							<li>Append Cloudflare Root CA</li>
							<li>Enable Always Use HTTPS</li>
							<li>Create www redirect page rule</li>
						</ul>
						<p class="mt-4 text-sm text-gray-600">The certificates will be stored in Cloudflare KV for K8s deployment.</p>
					</div>
				`,
				icon: 'question',
				showCancelButton: true,
				confirmButtonText: 'Configure SSL',
				cancelButtonText: 'Cancel',
				confirmButtonColor: '#3b82f6',
				showLoaderOnConfirm: true,
				preConfirm: async () => {
					try {
						setGlobalLoading('Configuring SSL...', 'Please wait while we configure SSL for your domain');
						const result = await dnsService.configureDomain(domain.name);
						return result;
					} catch (error) {
						Swal.showValidationMessage('Error: ' + error.message);
					} finally {
						clearLoading();
					}
				}
			});

			if (result.isConfirmed) {
				await Swal.fire({
					title: 'Success!',
					text: `SSL configuration completed for ${domain.name}`,
					icon: 'success',
					confirmButtonColor: '#10b981'
				});
				await loadDomains();
			}
		} catch (err) {
			console.error('Error configuring domain:', err);
		}
	}

	async function removeDomain(domain: Domain) {
		if (!browser) return;

		try {
			const { default: Swal } = await import('sweetalert2');
			
			const result = await Swal.fire({
				title: `Remove ${domain.name} from Xanthus?`,
				html: `
					<div class="text-left">
						<p class="mb-4">This will completely revert all Cloudflare changes made by Xanthus:</p>
						<ul class="list-disc list-inside space-y-2 text-sm text-red-600">
							<li>Delete origin server certificate</li>
							<li>Remove www redirect page rules</li>
							<li>Reset SSL mode to Flexible</li>
							<li>Disable Always Use HTTPS</li>
							<li>Remove configuration from Xanthus storage</li>
						</ul>
						<div class="mt-4 p-3 bg-amber-50 border border-amber-200 rounded-md">
							<p class="text-sm text-amber-800">
								<strong>Warning:</strong> Your domain will return to its original Cloudflare state before Xanthus management.
							</p>
						</div>
					</div>
				`,
				icon: 'warning',
				showCancelButton: true,
				confirmButtonText: 'Remove & Revert All',
				cancelButtonText: 'Cancel',
				confirmButtonColor: '#ef4444',
				showLoaderOnConfirm: true,
				preConfirm: async () => {
					try {
						setGlobalLoading('Removing domain...', 'Please wait while we revert all changes');
						const result = await dnsService.removeDomain(domain.name);
						return result;
					} catch (error) {
						Swal.showValidationMessage('Error: ' + error.message);
					} finally {
						clearLoading();
					}
				}
			});

			if (result.isConfirmed) {
				await Swal.fire({
					title: 'Successfully Removed!',
					html: `
						<div class="text-center">
							<p class="mb-3">All Cloudflare changes have been reverted for <strong>${domain.name}</strong></p>
							<p class="text-sm text-gray-600">Your domain has been restored to its original Cloudflare configuration.</p>
						</div>
					`,
					icon: 'success',
					confirmButtonColor: '#10b981'
				});
				await loadDomains();
			}
		} catch (err) {
			console.error('Error removing domain:', err);
		}
	}

	async function viewConfiguration(domain: Domain) {
		if (!browser) return;

		try {
			const { default: Swal } = await import('sweetalert2');
			
			await Swal.fire({
				title: `Configuration for ${domain.name}`,
				html: `
					<div class="text-left">
						<div class="space-y-3">
							<div class="flex justify-between">
								<span class="font-medium">SSL Mode:</span>
								<span class="text-green-600">Full (strict)</span>
							</div>
							<div class="flex justify-between">
								<span class="font-medium">Origin Certificate:</span>
								<span class="text-green-600">✓ Created</span>
							</div>
							<div class="flex justify-between">
								<span class="font-medium">Always Use HTTPS:</span>
								<span class="text-green-600">✓ Enabled</span>
							</div>
							<div class="flex justify-between">
								<span class="font-medium">Page Rule:</span>
								<span class="text-green-600">✓ www redirect</span>
							</div>
							<div class="flex justify-between">
								<span class="font-medium">Certificates:</span>
								<span class="text-green-600">✓ Stored in KV</span>
							</div>
						</div>
						<div class="mt-4 p-3 bg-blue-50 rounded-md">
							<p class="text-sm text-blue-800">
								SSL certificates are ready for K8s deployment. They will be automatically applied when you deploy your application.
							</p>
						</div>
					</div>
				`,
				icon: 'info',
				confirmButtonText: 'Close',
				confirmButtonColor: '#3b82f6'
			});
		} catch (err) {
			console.error('Error viewing configuration:', err);
		}
	}
</script>

<svelte:head>
	<title>DNS Configuration - Xanthus</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<!-- Header -->
	<div class="mb-8">
		<h2 class="text-3xl font-bold text-gray-900 mb-2">DNS Configuration</h2>
		<p class="text-gray-600">Manage your Cloudflare domains and SSL certificates</p>
	</div>

	<!-- Refresh Button -->
	<div class="mb-6">
		<Button 
			variant="primary" 
			size="sm" 
			on:click={loadDomains}
			disabled={loading}
		>
			{loading ? 'Refreshing...' : 'Refresh Domains'}
		</Button>
	</div>

	<!-- Info Box -->
	{#if domains.length > 0}
		<div class="mb-6">
			<Card>
				<div class="bg-blue-50 border border-blue-200 rounded-md p-4">
					<div class="flex">
						<div class="flex-shrink-0">
							<svg class="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
							</svg>
						</div>
						<div class="ml-3">
							<h3 class="text-sm font-medium text-blue-800">Domain Management</h3>
							<div class="mt-2 text-sm text-blue-700">
								<p>Below are all domains registered in your Cloudflare account. Domains marked as "Managed by Xanthus" are configured for K3s deployment.</p>
							</div>
						</div>
					</div>
				</div>
			</Card>
		</div>
	{/if}

	<!-- Error Message -->
	{#if error}
		<div class="mb-6">
			<Card>
				<div class="bg-red-50 border border-red-200 rounded-md p-4">
					<div class="flex">
						<div class="flex-shrink-0">
							<svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
							</svg>
						</div>
						<div class="ml-3">
							<h3 class="text-sm font-medium text-red-800">Error</h3>
							<p class="mt-1 text-sm text-red-700">{error}</p>
						</div>
					</div>
				</div>
			</Card>
		</div>
	{/if}

	<!-- Loading State -->
	{#if loading}
		<div class="text-center py-12">
			<div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
			<p class="mt-2 text-sm text-gray-600">Loading domains...</p>
		</div>
	{/if}

	<!-- Domains List -->
	{#if !loading && domains.length > 0}
		<div class="grid grid-cols-1 gap-4">
			{#each domains as domain}
				<Card>
					<div class="flex items-center justify-between p-6">
						<div class="flex-1">
							<div class="flex items-center space-x-3">
								<h3 class="text-lg font-semibold text-gray-900">{domain.name}</h3>
								
								<!-- Status Badge -->
								<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {getStatusBadgeClass(domain.status)}">
									<svg class="w-2 h-2 mr-1 fill-current" viewBox="0 0 8 8">
										<circle cx="4" cy="4" r="3"/>
									</svg>
									{domain.status.charAt(0).toUpperCase() + domain.status.slice(1)}
								</span>
								
								<!-- Xanthus Management Badge -->
								{#if domain.managed}
									<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
										<svg class="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
											<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
										</svg>
										Managed by Xanthus
									</span>
								{:else}
									<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600">
										Not Managed
									</span>
								{/if}
								
								<!-- Paused Badge -->
								{#if domain.paused}
									<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800">
										<svg class="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
											<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd"></path>
										</svg>
										Paused
									</span>
								{/if}
							</div>
							
							<div class="mt-2 text-sm text-gray-600">
								<p>Type: {domain.type} | ID: {domain.id}</p>
								<p>Created: {dnsService.formatDate(domain.created_on)} | Modified: {dnsService.formatDate(domain.modified_on)}</p>
							</div>
						</div>
						
						<div class="flex items-center space-x-2">
							{#if domain.managed}
								<Button 
									variant="secondary" 
									size="sm" 
									on:click={() => viewConfiguration(domain)}
								>
									View Config
								</Button>
								<Button 
									variant="danger" 
									size="sm" 
									on:click={() => removeDomain(domain)}
								>
									Remove
								</Button>
							{:else}
								<Button 
									variant="primary" 
									size="sm" 
									on:click={() => configureDomain(domain)}
								>
									Add to Xanthus
								</Button>
							{/if}
						</div>
					</div>
				</Card>
			{/each}
		</div>
	{/if}

	<!-- Empty State -->
	{#if !loading && domains.length === 0 && !error}
		<div class="text-center py-12">
			<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
			</svg>
			<h3 class="mt-2 text-sm font-medium text-gray-900">No domains found</h3>
			<p class="mt-1 text-sm text-gray-500">No domains are registered in your Cloudflare account.</p>
			<div class="mt-6">
				<a href="https://dash.cloudflare.com/" target="_blank" class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700">
					Add Domain to Cloudflare
				</a>
			</div>
		</div>
	{/if}
</div>