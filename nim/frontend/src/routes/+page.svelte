<script>
	import { auth } from '$lib/stores/auth.js';
	
	$: user = $auth.user;
</script>

<svelte:head>
	<title>Xanthus Dashboard</title>
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<header class="mb-8">
		<h1 class="text-3xl font-bold text-gray-900">Xanthus Dashboard</h1>
		{#if user}
			<p class="text-gray-600 mt-2">Welcome back, {user.name || user.email}</p>
		{/if}
	</header>

	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
		<div class="bg-white rounded-lg shadow-md border hover:shadow-lg transition-shadow p-6">
			<h2 class="text-xl font-semibold mb-4">Applications</h2>
			<p class="text-gray-600 mb-4">Deploy and manage your applications</p>
			<a href="/applications" class="inline-block px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors">Manage Applications</a>
		</div>

		<div class="bg-white rounded-lg shadow-md border hover:shadow-lg transition-shadow p-6">
			<h2 class="text-xl font-semibold mb-4">VPS Instances</h2>
			<p class="text-gray-600 mb-4">Provision and monitor your servers</p>
			<a href="/vps" class="inline-block px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors">Manage VPS</a>
		</div>

		<div class="bg-white rounded-lg shadow-md border hover:shadow-lg transition-shadow p-6">
			<h2 class="text-xl font-semibold mb-4">DNS Management</h2>
			<p class="text-gray-600 mb-4">Configure domains and DNS records</p>
			<a href="/dns" class="inline-block px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors">Manage DNS</a>
		</div>
	</div>
</div>

