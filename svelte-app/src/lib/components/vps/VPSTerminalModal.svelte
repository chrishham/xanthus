<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { terminalModal, hideTerminalModal } from '$lib/stores/vps';
	import { terminalService, type TerminalSession } from '$lib/services/terminal';
	import Button from '../common/Button.svelte';

	let terminalContainer: HTMLDivElement;
	let currentSession: TerminalSession | null = null;

	$: modal = $terminalModal;
	$: vps = modal.vps;

	onMount(() => {
		// Initialize terminal when modal opens and VPS is available
		if (modal.show && vps && terminalContainer) {
			initializeTerminal();
		}
	});

	onDestroy(() => {
		cleanup();
	});

	// Watch for modal state changes
	$: if (modal.show && vps && terminalContainer) {
		initializeTerminal();
	} else if (!modal.show) {
		cleanup();
	}

	async function initializeTerminal() {
		if (!vps || !terminalContainer) return;

		try {
			// Clean up any existing session
			cleanup();

			// Create new terminal session
			currentSession = await terminalService.createTerminalSession(vps, terminalContainer);
			
			console.log('Terminal session created:', currentSession.id);
		} catch (error) {
			console.error('Failed to initialize terminal:', error);
			// Show error in the terminal container
			if (terminalContainer) {
				terminalContainer.innerHTML = `
					<div class="flex items-center justify-center h-full text-red-600">
						<div class="text-center">
							<svg class="h-12 w-12 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.966-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
							</svg>
							<h3 class="text-lg font-medium mb-2">Failed to connect</h3>
							<p class="text-sm text-gray-500">${error instanceof Error ? error.message : 'Unknown error'}</p>
						</div>
					</div>
				`;
			}
		}
	}

	function cleanup() {
		if (currentSession) {
			// Use the session's cleanup method
			currentSession.cleanup();
			currentSession = null;
		}
	}

	function handleClose() {
		cleanup();
		hideTerminalModal();
	}

	function handleOpenInNewTab() {
		if (vps) {
			const url = `/terminal?vps_id=${vps.id}`;
			window.open(url, '_blank', 'noopener,noreferrer');
		}
	}

	function handleResize() {
		if (currentSession) {
			// Trigger resize after a short delay to ensure container dimensions are updated
			setTimeout(() => {
				terminalService.resizeSession(currentSession!.id);
			}, 100);
		}
	}

	// Handle window resize
	let resizeTimeout: number;
	function handleWindowResize() {
		clearTimeout(resizeTimeout);
		resizeTimeout = setTimeout(handleResize, 250);
	}

	// SSH user resolution based on provider
	function getSSHUser(provider: string): string {
		switch (provider) {
			case 'oracle':
				return 'ubuntu';
			case 'hetzner':
			default:
				return 'root';
		}
	}
</script>

<svelte:window on:resize={handleWindowResize} />

{#if modal.show}
	<!-- Modal Overlay -->
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
		<div class="bg-white rounded-lg shadow-xl w-full max-w-6xl h-full max-h-[90vh] flex flex-col">
			<!-- Header -->
			<div class="flex items-center justify-between p-4 border-b border-gray-200">
				<div class="flex items-center space-x-3">
					<svg class="h-6 w-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
					</svg>
					<div>
						<h3 class="text-lg font-semibold text-gray-900">
							Terminal - {vps?.name || 'Unknown Server'}
						</h3>
						<p class="text-sm text-gray-500">
							{#if vps}
								{getSSHUser(vps.provider)}@{vps.public_ip || vps.private_ip || vps.name}
							{/if}
						</p>
					</div>
				</div>

				<div class="flex items-center space-x-2">
					<!-- Connection Status -->
					<div class="flex items-center space-x-2">
						<div class="flex items-center">
							<div class="w-2 h-2 rounded-full {modal.connected ? 'bg-green-400' : 'bg-red-400'} mr-2"></div>
							<span class="text-sm text-gray-600">
								{modal.connected ? 'Connected' : 'Disconnected'}
							</span>
						</div>
					</div>

					<!-- Actions -->
					<Button
						variant="secondary"
						size="sm"
						on:click={handleOpenInNewTab}
						title="Open in new tab"
					>
						<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
						</svg>
					</Button>

					<Button
						variant="secondary"
						size="sm"
						on:click={handleResize}
						title="Resize terminal"
					>
						<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
						</svg>
					</Button>

					<Button
						variant="secondary"
						size="sm"
						on:click={handleClose}
						title="Close terminal"
					>
						<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</Button>
				</div>
			</div>

			<!-- Terminal Container -->
			<div class="flex-1 p-4 bg-gray-900">
				<div 
					bind:this={terminalContainer}
					class="w-full h-full rounded border border-gray-700 bg-black"
					style="min-height: 400px;"
				>
					<!-- Terminal will be mounted here -->
					{#if !currentSession}
						<div class="flex items-center justify-center h-full text-gray-400">
							<div class="text-center">
								<svg class="animate-spin h-8 w-8 mx-auto mb-4" fill="none" viewBox="0 0 24 24">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
								</svg>
								<p class="text-sm">Connecting to terminal...</p>
							</div>
						</div>
					{/if}
				</div>
			</div>

			<!-- Footer -->
			<div class="p-4 border-t border-gray-200 bg-gray-50">
				<div class="flex items-center justify-between text-sm text-gray-600">
					<div class="flex items-center space-x-4">
						<span>• Use Ctrl+C to interrupt</span>
						<span>• Use Ctrl+D to exit</span>
						<span>• Terminal supports copy/paste</span>
					</div>
					
					{#if currentSession}
						<div class="text-xs text-gray-500">
							Session ID: {currentSession.id}
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}