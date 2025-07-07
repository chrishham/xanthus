<script lang="ts">
	import { onMount } from 'svelte';

	let mounted = false;

	onMount(() => {
		mounted = true;
	});
</script>

<svelte:head>
	<title>Xanthus - Infrastructure Management</title>
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<div class="text-center">
		<h1 class="text-4xl font-bold text-gray-900 mb-4">
			Welcome to Xanthus Svelte
		</h1>
		<p class="text-xl text-gray-600 mb-8">
			Modern infrastructure management with Svelte frontend
		</p>
		{#if mounted}
			<div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded">
				✅ SvelteKit is running successfully!
			</div>
		{/if}
	</div>
</div>