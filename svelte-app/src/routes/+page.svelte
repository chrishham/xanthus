<script lang="ts">
	import DashboardCard from '$lib/components/common/DashboardCard.svelte';
	import { browser } from '$app/environment';
	import { api } from '$lib/services/api';

	// Mock data for Hetzner status - in real app this would come from API
	let hetznerStatus = 'Connected';

	async function showVersionModal() {
		if (browser) {
			const { default: Swal } = await import('sweetalert2');
			Swal.fire({
				title: 'Version Management',
				text: 'Version management functionality will be implemented here',
				icon: 'info'
			});
		}
	}
</script>

<svelte:head>
	<title>Xanthus - Dashboard</title>
</svelte:head>

<main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<!-- Header -->
	<div class="mb-8">
		<h2 class="text-3xl font-bold text-gray-900 mb-2">Dashboard</h2>
		<p class="text-gray-600">K3s Deployment Tool - Manage your infrastructure and applications</p>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
		<DashboardCard
			title="Cloudflare DNS"
			description="Manage DNS records and SSL certificates"
			variant="blue"
			buttonText="Configure"
			buttonHref="/dns"
		/>

		<DashboardCard
			title="Hetzner VPS"
			description="Provision and manage VPS instances"
			variant="green"
			buttonText="Manage"
			buttonHref="/vps"
		/>

		<DashboardCard
			title="Applications"
			description="Deploy, manage, and monitor applications"
			variant="purple"
			buttonText="Manage"
			buttonHref="/applications"
		/>

		<DashboardCard
			title="Platform Version"
			description="Update and manage Xanthus platform version"
			variant="orange"
			buttonText="Manage Version"
			onClick={showVersionModal}
		/>
	</div>

	<div class="mt-8">
		<h2 class="text-xl font-semibold text-gray-900 mb-4">Status</h2>
		<div class="bg-gray-50 p-4 rounded-lg space-y-3">
			<div class="flex items-center">
				<div class="w-3 h-3 bg-green-500 rounded-full mr-3"></div>
				<span class="text-gray-700">Connected to Cloudflare API</span>
			</div>
			<div class="flex items-center">
				{#if hetznerStatus === 'Connected'}
					<div class="w-3 h-3 bg-green-500 rounded-full mr-3"></div>
					<span class="text-gray-700">Connected to Hetzner API</span>
				{:else if hetznerStatus === 'Invalid key'}
					<div class="w-3 h-3 bg-red-500 rounded-full mr-3"></div>
					<span class="text-gray-700">Hetzner API key invalid</span>
				{:else}
					<div class="w-3 h-3 bg-gray-400 rounded-full mr-3"></div>
					<span class="text-gray-700">Hetzner API not configured</span>
				{/if}
			</div>
		</div>
	</div>
</main>