<script lang="ts">
	import { uiStore } from '$lib/stores/ui';
	import { fade } from 'svelte/transition';

	export let spinnerColor = 'border-blue-600';

	$: ({ loading, loadingTitle, loadingMessage } = $uiStore);
</script>

{#if loading}
	<div 
		class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
		transition:fade={{ duration: 200 }}
	>
		<div class="bg-white rounded-lg shadow-xl p-8 max-w-md mx-4">
			<div class="text-center">
				<div class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 {spinnerColor} mb-4"></div>
				<h3 class="text-lg font-medium text-gray-900 mb-2">{loadingTitle}</h3>
				<p class="text-gray-600">{loadingMessage}</p>
			</div>
		</div>
	</div>
{/if}