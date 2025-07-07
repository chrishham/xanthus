<script lang="ts">
	import { creationModal, hideCreationModal } from '$lib/stores/vps';
	import Button from '../common/Button.svelte';

	$: modal = $creationModal;
</script>

{#if modal.show}
	<!-- Modal Overlay -->
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
		<div class="bg-white rounded-lg shadow-xl w-full max-w-2xl">
			<!-- Header -->
			<div class="flex items-center justify-between p-6 border-b border-gray-200">
				<h3 class="text-lg font-semibold text-gray-900">Create VPS Server</h3>
				<Button
					variant="secondary"
					size="sm"
					on:click={hideCreationModal}
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
					<h4 class="mt-2 text-lg font-medium text-gray-900">VPS Creation Wizard</h4>
					<p class="mt-1 text-sm text-gray-500">
						Step {modal.step} of {modal.maxSteps} - Coming soon in Phase 4B
					</p>
					<div class="mt-6">
						<Button variant="secondary" on:click={hideCreationModal}>
							Close
						</Button>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}