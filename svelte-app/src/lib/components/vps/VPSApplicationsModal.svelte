<script lang="ts">
	import { applicationsModal, hideApplicationsModal } from '$lib/stores/vps';
	import Button from '../common/Button.svelte';

	$: modal = $applicationsModal;
</script>

{#if modal.show}
	<!-- Modal Overlay -->
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
		<div class="bg-white rounded-lg shadow-xl w-full max-w-3xl">
			<!-- Header -->
			<div class="flex items-center justify-between p-6 border-b border-gray-200">
				<div class="flex items-center space-x-3">
					<svg class="h-6 w-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
					</svg>
					<h3 class="text-lg font-semibold text-gray-900">
						Applications - {modal.vps?.name || 'Unknown Server'}
					</h3>
				</div>
				<Button
					variant="secondary"
					size="sm"
					on:click={hideApplicationsModal}
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
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
					</svg>
					<h4 class="mt-2 text-lg font-medium text-gray-900">Deployed Applications</h4>
					<p class="mt-1 text-sm text-gray-500">
						View and manage applications on this server - Coming soon in Phase 4C
					</p>
					<div class="mt-6">
						<Button variant="secondary" on:click={hideApplicationsModal}>
							Close
						</Button>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}