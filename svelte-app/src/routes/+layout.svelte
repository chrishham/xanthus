<script lang="ts">
	import '../app.css';
	import Navigation from '$lib/components/common/Navigation.svelte';
	import LoadingModal from '$lib/components/common/LoadingModal.svelte';
	import NotificationSystem from '$lib/components/common/NotificationSystem.svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { setCurrentPage } from '$lib/stores/ui';
	import { checkAuthStatus } from '$lib/stores/auth';

	onMount(() => {
		checkAuthStatus();
	});

	// Update current page based on route
	$: {
		if ($page.url.pathname === '/app') {
			setCurrentPage('main');
		} else if ($page.url.pathname.startsWith('/app/applications')) {
			setCurrentPage('applications');
		} else if ($page.url.pathname.startsWith('/app/vps')) {
			setCurrentPage('vps');
		} else if ($page.url.pathname.startsWith('/app/dns')) {
			setCurrentPage('dns');
		} else if ($page.url.pathname.startsWith('/app/version')) {
			setCurrentPage('version');
		}
	}
</script>

<div class="min-h-screen bg-gray-100">
	<Navigation currentPage={$page.url.pathname.includes('/app/') ? $page.url.pathname.split('/')[2] || 'main' : 'main'} />
	<slot />
	<LoadingModal />
	<NotificationSystem />
</div>