<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { api } from '$lib/services/api';
	import { setLoading, clearLoading } from '$lib/stores/ui';
	import Card from '$lib/components/common/Card.svelte';
	import Button from '$lib/components/common/Button.svelte';

	interface GitHubRelease {
		tag_name: string;
		name: string;
		body: string;
		draft: boolean;
		prerelease: boolean;
		published_at: string;
		html_url: string;
		assets: Array<{
			name: string;
			browser_download_url: string;
		}>;
	}

	interface VersionInfo {
		current: string;
		available: GitHubRelease[];
		status: string;
	}

	interface UpdateStatus {
		in_progress: boolean;
		version: string;
		status: string;
		progress: number;
		message: string;
		error?: string;
	}

	let versionInfo: VersionInfo | null = null;
	let updateStatus: UpdateStatus | null = null;
	let loading = false;
	let error = '';
	let autoRefreshInterval: number | null = null;

	onMount(async () => {
		await loadVersionInfo();
		await loadUpdateStatus();
		
		// Auto-refresh update status if an update is in progress
		if (updateStatus?.in_progress) {
			startAutoRefresh();
		}

		return () => {
			if (autoRefreshInterval) {
				clearInterval(autoRefreshInterval);
			}
		};
	});

	async function loadVersionInfo() {
		loading = true;
		error = '';
		try {
			versionInfo = await api.get('/version/available');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load version information';
			console.error('Error loading version info:', err);
		} finally {
			loading = false;
		}
	}

	async function loadUpdateStatus() {
		try {
			updateStatus = await api.get('/version/status');
		} catch (err) {
			console.error('Error loading update status:', err);
		}
	}

	function startAutoRefresh() {
		if (autoRefreshInterval) {
			clearInterval(autoRefreshInterval);
		}
		
		autoRefreshInterval = setInterval(async () => {
			await loadUpdateStatus();
			if (!updateStatus?.in_progress) {
				clearInterval(autoRefreshInterval!);
				autoRefreshInterval = null;
				await loadVersionInfo(); // Reload version info after update completes
			}
		}, 2000); // Refresh every 2 seconds
	}

	async function triggerUpdate(version: string) {
		if (!browser) return;

		try {
			const { default: Swal } = await import('sweetalert2');
			
			const result = await Swal.fire({
				title: `Update to ${version}?`,
				html: `
					<div class="text-left">
						<p class="mb-4">This will update Xanthus to version <strong>${version}</strong>.</p>
						<div class="bg-yellow-50 border border-yellow-200 rounded-md p-3">
							<p class="text-sm text-yellow-800">
								<strong>Warning:</strong> The platform will restart during the update process. This may take a few minutes.
							</p>
						</div>
						<p class="mt-3 text-sm text-gray-600">
							The previous version will be available for rollback if needed.
						</p>
					</div>
				`,
				icon: 'question',
				showCancelButton: true,
				confirmButtonText: 'Start Update',
				cancelButtonText: 'Cancel',
				confirmButtonColor: '#3b82f6',
				showLoaderOnConfirm: true,
				preConfirm: async () => {
					try {
						setLoading('Starting update...', 'Please wait while we prepare the update');
						const result = await api.post('/version/update', { version });
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
					title: 'Update Started!',
					text: `Xanthus is updating to ${version}. The platform will restart when complete.`,
					icon: 'success',
					confirmButtonColor: '#10b981'
				});
				
				// Start monitoring update status
				await loadUpdateStatus();
				if (updateStatus?.in_progress) {
					startAutoRefresh();
				}
			}
		} catch (err) {
			console.error('Error triggering update:', err);
		}
	}

	async function rollbackVersion() {
		if (!browser) return;

		try {
			const { default: Swal } = await import('sweetalert2');
			
			const result = await Swal.fire({
				title: 'Rollback to Previous Version?',
				html: `
					<div class="text-left">
						<p class="mb-4">This will rollback Xanthus to the previous version.</p>
						<div class="bg-red-50 border border-red-200 rounded-md p-3">
							<p class="text-sm text-red-800">
								<strong>Warning:</strong> The platform will restart during the rollback process.
							</p>
						</div>
						<p class="mt-3 text-sm text-gray-600">
							Only use this if you're experiencing issues with the current version.
						</p>
					</div>
				`,
				icon: 'warning',
				showCancelButton: true,
				confirmButtonText: 'Rollback',
				cancelButtonText: 'Cancel',
				confirmButtonColor: '#ef4444',
				showLoaderOnConfirm: true,
				preConfirm: async () => {
					try {
						setLoading('Starting rollback...', 'Please wait while we rollback to the previous version');
						const result = await api.post('/version/rollback', {});
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
					title: 'Rollback Started!',
					text: 'Xanthus is rolling back to the previous version. The platform will restart when complete.',
					icon: 'success',
					confirmButtonColor: '#10b981'
				});
				
				// Start monitoring update status
				await loadUpdateStatus();
				if (updateStatus?.in_progress) {
					startAutoRefresh();
				}
			}
		} catch (err) {
			console.error('Error triggering rollback:', err);
		}
	}

	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	function isNewerVersion(version: string, currentVersion: string): boolean {
		// Simple version comparison - assumes semantic versioning
		const parseVersion = (v: string) => v.replace(/^v/, '').split('.').map(Number);
		const [major1, minor1, patch1] = parseVersion(version);
		const [major2, minor2, patch2] = parseVersion(currentVersion);
		
		if (major1 > major2) return true;
		if (major1 < major2) return false;
		if (minor1 > minor2) return true;
		if (minor1 < minor2) return false;
		return patch1 > patch2;
	}

	function getVersionBadgeClass(release: GitHubRelease): string {
		if (release.prerelease) {
			return 'bg-yellow-100 text-yellow-800';
		}
		if (versionInfo && isNewerVersion(release.tag_name, versionInfo.current)) {
			return 'bg-green-100 text-green-800';
		}
		return 'bg-blue-100 text-blue-800';
	}

	function getVersionBadgeText(release: GitHubRelease): string {
		if (release.prerelease) {
			return 'Pre-release';
		}
		if (versionInfo && isNewerVersion(release.tag_name, versionInfo.current)) {
			return 'Newer';
		}
		if (versionInfo && release.tag_name === versionInfo.current) {
			return 'Current';
		}
		return 'Available';
	}

	function getProgressBarClass(progress: number): string {
		if (progress < 30) return 'bg-red-500';
		if (progress < 70) return 'bg-yellow-500';
		return 'bg-green-500';
	}
</script>

<svelte:head>
	<title>Version Management - Xanthus</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<!-- Header -->
	<div class="mb-8">
		<h2 class="text-3xl font-bold text-gray-900 mb-2">Version Management</h2>
		<p class="text-gray-600">Manage platform updates and monitor version status</p>
	</div>

	<!-- Refresh Button -->
	<div class="mb-6">
		<Button 
			variant="primary" 
			size="sm" 
			on:click={loadVersionInfo}
			disabled={loading}
		>
			{loading ? 'Refreshing...' : 'Refresh Versions'}
		</Button>
	</div>

	<!-- Update Status -->
	{#if updateStatus?.in_progress}
		<div class="mb-6">
			<Card>
				<div class="p-6">
					<div class="flex items-center justify-between mb-4">
						<h3 class="text-lg font-medium text-gray-900">Update in Progress</h3>
						<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
							{updateStatus.status}
						</span>
					</div>
					
					<div class="mb-4">
						<div class="flex justify-between text-sm text-gray-600 mb-1">
							<span>Updating to {updateStatus.version}</span>
							<span>{updateStatus.progress}%</span>
						</div>
						<div class="w-full bg-gray-200 rounded-full h-2">
							<div 
								class="h-2 rounded-full transition-all duration-300 {getProgressBarClass(updateStatus.progress)}"
								style="width: {updateStatus.progress}%"
							></div>
						</div>
					</div>
					
					<p class="text-sm text-gray-600">{updateStatus.message}</p>
					
					{#if updateStatus.error}
						<div class="mt-3 p-3 bg-red-50 border border-red-200 rounded-md">
							<p class="text-sm text-red-800">{updateStatus.error}</p>
						</div>
					{/if}
				</div>
			</Card>
		</div>
	{/if}

	<!-- Current Version -->
	{#if versionInfo}
		<div class="mb-6">
			<Card>
				<div class="p-6">
					<h3 class="text-lg font-medium text-gray-900 mb-4">Current Version</h3>
					<div class="flex items-center justify-between">
						<div>
							<p class="text-2xl font-bold text-blue-600">{versionInfo.current}</p>
							<p class="text-sm text-gray-600">Currently running</p>
						</div>
						<div class="flex space-x-2">
							<Button 
								variant="secondary" 
								size="sm" 
								on:click={rollbackVersion}
								disabled={updateStatus?.in_progress}
							>
								Rollback
							</Button>
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
			<p class="mt-2 text-sm text-gray-600">Loading version information...</p>
		</div>
	{/if}

	<!-- Available Versions -->
	{#if !loading && versionInfo && versionInfo.available.length > 0}
		<div>
			<h3 class="text-lg font-medium text-gray-900 mb-4">Available Versions</h3>
			<div class="grid grid-cols-1 gap-4">
				{#each versionInfo.available as release}
					<Card>
						<div class="p-6">
							<div class="flex items-start justify-between">
								<div class="flex-1">
									<div class="flex items-center space-x-3 mb-2">
										<h4 class="text-lg font-semibold text-gray-900">{release.tag_name}</h4>
										<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {getVersionBadgeClass(release)}">
											{getVersionBadgeText(release)}
										</span>
									</div>
									
									<h5 class="text-md font-medium text-gray-700 mb-2">{release.name}</h5>
									
									<div class="text-sm text-gray-600 mb-3">
										<p>Published: {formatDate(release.published_at)}</p>
									</div>
									
									{#if release.body}
										<div class="text-sm text-gray-700 bg-gray-50 rounded-md p-3 mb-3">
											<pre class="whitespace-pre-wrap font-mono text-xs">{release.body}</pre>
										</div>
									{/if}
								</div>
								
								<div class="flex items-center space-x-2 ml-4">
									{#if release.tag_name !== versionInfo.current && !release.draft}
										<Button 
											variant={isNewerVersion(release.tag_name, versionInfo.current) ? 'primary' : 'secondary'}
											size="sm" 
											on:click={() => triggerUpdate(release.tag_name)}
											disabled={updateStatus?.in_progress}
										>
											{isNewerVersion(release.tag_name, versionInfo.current) ? 'Update' : 'Install'}
										</Button>
									{/if}
									
									<a 
										href={release.html_url} 
										target="_blank" 
										class="inline-flex items-center px-3 py-1.5 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
									>
										View on GitHub
										<svg class="ml-1 w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
											<path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" />
										</svg>
									</a>
								</div>
							</div>
						</div>
					</Card>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Empty State -->
	{#if !loading && versionInfo && versionInfo.available.length === 0 && !error}
		<div class="text-center py-12">
			<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
			</svg>
			<h3 class="mt-2 text-sm font-medium text-gray-900">No updates available</h3>
			<p class="mt-1 text-sm text-gray-500">You're running the latest version of Xanthus.</p>
		</div>
	{/if}
</div>