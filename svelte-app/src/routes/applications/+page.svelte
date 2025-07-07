<script lang="ts">
	import { onMount } from 'svelte';
	import ApplicationsList from '$lib/components/applications/ApplicationsList.svelte';
	import LoadingModal from '$lib/components/common/LoadingModal.svelte';
	import { api } from '$lib/services/api';
	import type { Application, PredefinedApp } from '../../app';

	// Page data - will be populated by server-side rendering or API calls
	let initialApplications: Application[] = [];
	let initialPredefinedApps: PredefinedApp[] = [];
	
	// Check if we have initial data from server-side rendering
	onMount(async () => {
		try {
			// Check if we have window globals (from Go template)
			if (typeof window !== 'undefined' && (window as any).initialApplications) {
				initialApplications = (window as any).initialApplications || [];
				initialPredefinedApps = (window as any).initialPredefinedApps || [];
			} else {
				// Fetch data from API if no initial data
				const [appsResponse, predefinedResponse] = await Promise.all([
					api.get('/applications/list'),
					api.get('/applications/predefined')
				]);
				
				initialApplications = appsResponse.applications || [];
				initialPredefinedApps = predefinedResponse.predefined_apps || [];
			}
		} catch (error) {
			console.error('Error loading initial data:', error);
		}
	});
</script>

<svelte:head>
	<title>Applications - Xanthus</title>
	<meta name="description" content="Deploy and manage applications on your VPS servers" />
</svelte:head>

<ApplicationsList {initialApplications} {initialPredefinedApps} />

<LoadingModal />