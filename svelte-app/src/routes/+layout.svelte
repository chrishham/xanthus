<script lang="ts">
	import '../app.css';
	import Navigation from '$lib/components/common/Navigation.svelte';
	import LoadingModal from '$lib/components/common/LoadingModal.svelte';
	import NotificationSystem from '$lib/components/common/NotificationSystem.svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { setCurrentPage } from '$lib/stores/ui';
	import { authStore } from '$lib/stores/auth';

	onMount(() => {
		authStore.initialize();
	});

	// Authentication guard for app routes (exclude login and setup)
	$: {
		const isAppRoute = $page.url.pathname.startsWith('/app');
		const isLoginRoute = $page.url.pathname === '/app/login' || $page.url.pathname.startsWith('/app/login/');
		const isSetupRoute = $page.url.pathname === '/app/setup' || $page.url.pathname.startsWith('/app/setup/');
		
		if (isAppRoute && !isLoginRoute && !isSetupRoute && !$authStore.isAuthenticated) {
			goto('/app/login');
		}
	}

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