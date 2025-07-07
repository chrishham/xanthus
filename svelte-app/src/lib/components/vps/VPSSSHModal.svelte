<script lang="ts">
	import { sshModal, hideSSHModal } from '$lib/stores/vps';
	import Button from '../common/Button.svelte';

	$: modal = $sshModal;
	$: vps = modal.vps;

	function getSSHUser(provider: string): string {
		switch (provider) {
			case 'oracle':
				return 'ubuntu';
			case 'hetzner':
			default:
				return 'root';
		}
	}

	function getSSHCommand(): string {
		if (!vps) return '';
		const user = getSSHUser(vps.provider);
		const ip = vps.public_ip || vps.private_ip || '';
		return `ssh -i xanthus-key.pem ${user}@${ip}`;
	}

	async function copyToClipboard(text: string) {
		try {
			await navigator.clipboard.writeText(text);
		} catch (error) {
			console.error('Failed to copy to clipboard:', error);
		}
	}
</script>

{#if modal.show}
	<!-- Modal Overlay -->
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
		<div class="bg-white rounded-lg shadow-xl w-full max-w-2xl">
			<!-- Header -->
			<div class="flex items-center justify-between p-6 border-b border-gray-200">
				<div class="flex items-center space-x-3">
					<svg class="h-6 w-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 12H9v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2h2.764l4.893-4.893A6 6 0 0115 7z" />
					</svg>
					<h3 class="text-lg font-semibold text-gray-900">
						SSH Information - {vps?.name || 'Unknown Server'}
					</h3>
				</div>
				<Button
					variant="secondary"
					size="sm"
					on:click={hideSSHModal}
				>
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</Button>
			</div>

			<!-- Content -->
			<div class="p-6 space-y-6">
				{#if vps}
					<!-- Server Information -->
					<div class="bg-gray-50 rounded-lg p-4">
						<h4 class="text-sm font-medium text-gray-900 mb-3">Server Details</h4>
						<div class="grid grid-cols-2 gap-4 text-sm">
							<div>
								<span class="text-gray-500">Provider:</span>
								<span class="ml-2 font-medium capitalize">{vps.provider}</span>
							</div>
							<div>
								<span class="text-gray-500">Status:</span>
								<span class="ml-2 font-medium capitalize">{vps.status}</span>
							</div>
							<div>
								<span class="text-gray-500">Public IP:</span>
								<span class="ml-2 font-mono font-medium">{vps.public_ip || 'N/A'}</span>
							</div>
							<div>
								<span class="text-gray-500">Private IP:</span>
								<span class="ml-2 font-mono font-medium">{vps.private_ip || 'N/A'}</span>
							</div>
						</div>
					</div>

					<!-- SSH Command -->
					<div>
						<h4 class="text-sm font-medium text-gray-900 mb-3">SSH Connection</h4>
						<div class="bg-gray-900 rounded-lg p-4">
							<div class="flex items-center justify-between">
								<code class="text-green-400 text-sm font-mono">{getSSHCommand()}</code>
								<Button
									variant="secondary"
									size="sm"
									on:click={() => copyToClipboard(getSSHCommand())}
									title="Copy SSH command"
								>
									<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
									</svg>
								</Button>
							</div>
						</div>
						<p class="text-xs text-gray-500 mt-2">
							Make sure you have the xanthus-key.pem file in your current directory
						</p>
					</div>

					<!-- SSH Key Information -->
					<div>
						<h4 class="text-sm font-medium text-gray-900 mb-3">SSH Key Requirements</h4>
						<div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
							<div class="flex items-start">
								<svg class="h-5 w-5 text-blue-400 mt-0.5 mr-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
								<div class="text-sm">
									<p class="text-blue-800 font-medium mb-1">SSH Key Setup</p>
									<ul class="text-blue-700 space-y-1">
										<li>• Download the SSH private key (xanthus-key.pem) from your Xanthus dashboard</li>
										<li>• Set proper permissions: <code class="bg-blue-100 px-1 rounded">chmod 600 xanthus-key.pem</code></li>
										<li>• Use the command above to connect to your server</li>
									</ul>
								</div>
							</div>
						</div>
					</div>

					<!-- Provider-Specific Information -->
					<div>
						<h4 class="text-sm font-medium text-gray-900 mb-3">Provider Information</h4>
						<div class="text-sm text-gray-600">
							{#if vps.provider === 'oracle'}
								<p><strong>Oracle Cloud:</strong> Default user is 'ubuntu'. Root access available via sudo.</p>
							{:else if vps.provider === 'hetzner'}
								<p><strong>Hetzner Cloud:</strong> Direct root access provided.</p>
							{:else}
								<p>Provider-specific SSH information not available.</p>
							{/if}
						</div>
					</div>
				{:else}
					<div class="text-center py-8">
						<p class="text-gray-500">No server information available</p>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="p-6 border-t border-gray-200 bg-gray-50">
				<div class="flex justify-end">
					<Button variant="secondary" on:click={hideSSHModal}>
						Close
					</Button>
				</div>
			</div>
		</div>
	</div>
{/if}