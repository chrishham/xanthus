<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { applicationStore, hideDeploymentModal } from '$lib/stores/applications';
	import { api } from '$lib/services/api';
	import { setLoading, setLoadingTitle, setLoadingMessage } from '$lib/stores/ui';
	import Button from '$lib/components/common/Button.svelte';
	import Input from '$lib/components/forms/Input.svelte';
	import Select from '$lib/components/forms/Select.svelte';
	import type { PredefinedApp } from '../../../app';

	const dispatch = createEventDispatcher();

	// Reactive state from store
	$: ({ modals } = $applicationStore);
	$: modal = modals.deployment;
	$: show = modal.show;
	$: predefinedApp = modal.predefinedApp;
	$: domains = modal.domains;
	$: servers = modal.servers;

	// Form state
	let appName = '';
	let selectedVPS = '';
	let selectedDomain = '';
	let subdomain = '';
	let description = '';
	let selectedVersion = '';
	let versions: Array<{ version: string; is_latest: boolean; is_stable: boolean }> = [];
	let versionsLoading = false;

	// Form validation
	$: formValid = appName.trim() !== '' && selectedVPS !== '' && selectedDomain !== '' && subdomain.trim() !== '';

	// Reactive updates when modal opens
	$: if (show && predefinedApp) {
		resetForm();
		loadVersions();
	}

	function resetForm() {
		if (!predefinedApp) return;
		
		appName = `my-${predefinedApp.id}`;
		selectedVPS = '';
		selectedDomain = '';
		subdomain = predefinedApp.id;
		description = `My ${predefinedApp.name} instance`;
		selectedVersion = predefinedApp.version;
		versions = [];
	}

	async function loadVersions() {
		if (!predefinedApp) return;
		
		versionsLoading = true;
		try {
			const data = await api.get(`/applications/versions/${predefinedApp.id}`);
			if (data.success && data.versions) {
				versions = data.versions;
				// Select default version if not already set
				if (!selectedVersion && versions.length > 0) {
					const defaultVersion = versions.find(v => v.version === predefinedApp.version) || versions[0];
					selectedVersion = defaultVersion.version;
				}
			}
		} catch (error) {
			console.warn('Failed to fetch versions:', error);
			// Use predefined app version as fallback
			versions = [{ version: predefinedApp.version, is_latest: false, is_stable: true }];
			selectedVersion = predefinedApp.version;
		} finally {
			versionsLoading = false;
		}
	}

	function closeModal() {
		hideDeploymentModal();
	}

	async function handleDeploy() {
		if (!formValid || !predefinedApp) return;

		setLoading(true);
		setLoadingTitle('Deploying Application');
		setLoadingMessage(`Deploying ${predefinedApp.name} to your VPS...`);

		try {
			const deploymentData = {
				app_type: predefinedApp.id,
				name: appName.trim(),
				vps_id: selectedVPS,
				domain: selectedDomain,
				subdomain: subdomain.trim(),
				description: description.trim(),
				version: selectedVersion
			};

			const response = await api.post('/applications/deploy', deploymentData);
			
			if (response.success) {
				dispatch('deployment-success', response);
				closeModal();
				// Show success message
				// Note: In a real implementation, you might want to use a toast system here
				console.log('Application deployed successfully');
			} else {
				throw new Error(response.error || 'Deployment failed');
			}
		} catch (error) {
			console.error('Deployment error:', error);
			dispatch('deployment-error', error);
		} finally {
			setLoading(false);
		}
	}

	// Generate domain options for select
	$: domainOptions = domains.map(d => ({ value: d.name, label: d.name }));
	
	// Generate server options for select
	$: serverOptions = servers.map(s => ({ 
		value: s.id, 
		label: `${s.name} (${s.public_net.ipv4.ip})` 
	}));

	// Generate version options for select
	$: versionOptions = versions.map(v => ({
		value: v.version,
		label: `${v.version}${v.is_latest ? ' (Latest)' : ''}${!v.is_stable ? ' (Pre-release)' : ''}`
	}));

	// Keyboard handling
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			closeModal();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if show}
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" on:click={closeModal}>
		<div class="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto" on:click|stopPropagation>
			<!-- Header -->
			<div class="p-6 border-b border-gray-200">
				<div class="flex items-center justify-between">
					<div class="flex items-center">
						{#if predefinedApp}
							<div class="text-2xl mr-3">{predefinedApp.icon}</div>
							<div>
								<h3 class="text-lg font-medium text-gray-900">Deploy {predefinedApp.name}</h3>
								<p class="text-sm text-gray-600">{predefinedApp.description}</p>
							</div>
						{/if}
					</div>
					<button on:click={closeModal} class="text-gray-400 hover:text-gray-600">
						<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
						</svg>
					</button>
				</div>
			</div>

			<!-- Form -->
			<div class="p-6 space-y-6">
				<!-- Application Name -->
				<div>
					<Input
						label="Application Name"
						bind:value={appName}
						placeholder="my-{predefinedApp?.id || 'app'}"
						help="A friendly name for your application instance"
						required
					/>
				</div>

				<!-- VPS Server Selection -->
				<div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
					<div class="flex items-center mb-2">
						<svg class="w-4 h-4 mr-1 text-blue-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z"></path>
						</svg>
						<span class="text-sm font-medium text-blue-900">VPS Server *</span>
					</div>
					<Select
						bind:value={selectedVPS}
						options={serverOptions}
						placeholder="👆 Click to choose a VPS server"
						help="Select the server where your application will be deployed"
						class="border-blue-300 focus:border-blue-500 focus:ring-blue-500"
						required
					/>
				</div>

				<!-- Domain Selection -->
				<div class="bg-green-50 border border-green-200 rounded-lg p-4">
					<div class="flex items-center mb-2">
						<svg class="w-4 h-4 mr-1 text-green-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9v-9m0-9v9"></path>
						</svg>
						<span class="text-sm font-medium text-green-900">Domain *</span>
					</div>
					<Select
						bind:value={selectedDomain}
						options={domainOptions}
						placeholder="👆 Click to select a domain"
						help="Choose the domain for your application"
						class="border-green-300 focus:border-green-500 focus:ring-green-500"
						required
					/>
				</div>

				<!-- Subdomain -->
				<div>
					<Input
						label="Subdomain"
						bind:value={subdomain}
						placeholder={predefinedApp?.id || 'app'}
						help="Your app will be available at subdomain.domain.com"
						required
					/>
				</div>

				<!-- Description -->
				<div>
					<Input
						label="Description (optional)"
						bind:value={description}
						placeholder="My {predefinedApp?.name || 'application'} instance"
						help="Optional description for this application instance"
					/>
				</div>

				<!-- Version Selection -->
				<div class="bg-purple-50 border border-purple-200 rounded-lg p-4">
					<div class="flex items-center mb-2">
						<svg class="w-4 h-4 mr-1 text-purple-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a2 2 0 012-2z"></path>
						</svg>
						<span class="text-sm font-medium text-purple-900">Version</span>
					</div>
					<Select
						bind:value={selectedVersion}
						options={versionOptions}
						placeholder="Select version..."
						help="Choose the application version to deploy"
						loading={versionsLoading}
						class="border-purple-300 focus:border-purple-500 focus:ring-purple-500"
					/>
				</div>

				<!-- URL Preview -->
				{#if selectedDomain && subdomain}
					<div class="bg-gray-50 border border-gray-200 rounded-lg p-4">
						<div class="flex items-center mb-2">
							<svg class="w-4 h-4 mr-1 text-gray-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.102m0 0l4-4a4 4 0 105.656-5.656l-1.102 1.102m-2.828 2.828l4 4"></path>
							</svg>
							<span class="text-sm font-medium text-gray-700">Application URL</span>
						</div>
						<p class="text-lg font-mono bg-white px-3 py-2 rounded border">
							https://{subdomain}.{selectedDomain}
						</p>
					</div>
				{/if}
			</div>

			<!-- Actions -->
			<div class="px-6 py-4 bg-gray-50 border-t border-gray-200 rounded-b-lg">
				<div class="flex justify-end space-x-3">
					<Button variant="outline" on:click={closeModal}>
						Cancel
					</Button>
					<Button 
						variant="primary"
						on:click={handleDeploy}
						disabled={!formValid}
						class="bg-purple-600 hover:bg-purple-700">
						Deploy Application
					</Button>
				</div>
			</div>
		</div>
	</div>
{/if}