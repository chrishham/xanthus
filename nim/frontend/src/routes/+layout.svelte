<script>
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth.js';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	onMount(() => {
		// Check authentication status on app load
		auth.checkAuth();
		
		// Subscribe to auth changes
		const unsubscribe = auth.subscribe((authState) => {
			// Redirect to login if not authenticated and not on auth page
			if (!authState.isAuthenticated && !$page.url.pathname.startsWith('/auth')) {
				goto('/auth/login');
			}
		});

		return unsubscribe;
	});
</script>

<main class="min-h-screen bg-gray-50">
	<slot />
</main>

<style>
	:global(html) {
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
	}
</style>