<script lang="ts">
	import { healthModal, hideHealthModal } from '$lib/stores/vps';
	import Button from '../common/Button.svelte';

	$: modal = $healthModal;
</script>

{#if modal.show}
	<!-- Modal Overlay -->
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
		<div class="bg-white rounded-lg shadow-xl w-full max-w-2xl">
			<!-- Header -->
			<div class="flex items-center justify-between p-6 border-b border-gray-200">
				<div class="flex items-center space-x-3">
					<svg class="h-6 w-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
					</svg>
					<h3 class="text-lg font-semibold text-gray-900">
						Server Health - {modal.vps?.name || 'Unknown Server'}
					</h3>
				</div>
				<Button
					variant="secondary"
					size="sm"
					on:click={hideHealthModal}
				>
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</Button>
			</div>

			<!-- Content -->
			<div class="p-6">
				<div class="text-center py-12">
					<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
					</svg>
					<h4 class="mt-2 text-lg font-medium text-gray-900">Health Monitoring</h4>
					<p class="mt-1 text-sm text-gray-500">
						Server health and diagnostics - Coming soon in Phase 4C
					</p>
					<div class="mt-6">
						<Button variant="secondary" on:click={hideHealthModal}>
							Close
						</Button>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}