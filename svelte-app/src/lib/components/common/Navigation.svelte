<script lang="ts">
	import { showModal } from '$lib/stores/ui';
	import { api } from '$lib/services/api';
	import { browser } from '$app/environment';

	export let currentPage = '';

	async function showAboutModal() {
		try {
			const data = await api.get('/about');
			
			if (browser) {
				const { default: Swal } = await import('sweetalert2');
				
				Swal.fire({
					title: 'About Xanthus',
					html: `
						<div class="text-left space-y-4">
							<div class="text-center mb-4">
								<img src="/static/icons/logo.png" alt="Xanthus Logo" class="w-16 h-16 mx-auto mb-2">
								<h3 class="text-xl font-bold text-gray-900">Xanthus</h3>
								<p class="text-gray-600">Configuration-Driven Infrastructure Management Platform</p>
							</div>
							
							<div class="grid grid-cols-2 gap-4 text-sm">
								<div>
									<span class="font-semibold text-gray-700">Version:</span>
									<span class="text-gray-900">${data.version}</span>
								</div>
								<div>
									<span class="font-semibold text-gray-700">Build Date:</span>
									<span class="text-gray-900">${data.build_date}</span>
								</div>
								<div>
									<span class="font-semibold text-gray-700">Go Version:</span>
									<span class="text-gray-900">${data.go_version}</span>
								</div>
								<div>
									<span class="font-semibold text-gray-700">Platform:</span>
									<span class="text-gray-900">${data.platform}</span>
								</div>
							</div>
							
							<div class="border-t pt-4">
								<h4 class="font-semibold text-gray-700 mb-2">Features</h4>
								<ul class="text-sm text-gray-600 space-y-1">
									<li>• VPS provisioning (Hetzner Cloud & Oracle Cloud)</li>
									<li>• DNS & SSL management via Cloudflare</li>
									<li>• K3s Kubernetes orchestration</li>
									<li>• Configuration-driven application deployment</li>
									<li>• Self-updating platform capabilities</li>
								</ul>
							</div>
							
							<div class="border-t pt-4 text-center">
								<p class="text-xs text-gray-500">
									Open source project licensed under MIT<br>
									<a href="https://github.com/your-org/xanthus" target="_blank" class="text-blue-600 hover:text-blue-800">View on GitHub</a>
								</p>
							</div>
						</div>
					`,
					width: 500,
					showCancelButton: false,
					confirmButtonText: 'Close',
					confirmButtonColor: '#6B7280'
				});
			}
		} catch (error) {
			console.error('Error showing about modal:', error);
			if (browser) {
				const { default: Swal } = await import('sweetalert2');
				Swal.fire('Error', 'Failed to load about information: ' + error.message, 'error');
			}
		}
	}

	function isActive(page: string): boolean {
		return currentPage === page;
	}

	function getNavLinkClass(page: string): string {
		if (isActive(page)) {
			switch (page) {
				case 'dns':
					return 'text-blue-600 bg-blue-50 px-3 py-2 rounded-md text-sm font-medium';
				case 'vps':
					return 'text-blue-600 bg-blue-50 px-3 py-2 rounded-md text-sm font-medium';
				case 'applications':
					return 'text-purple-600 bg-purple-50 px-3 py-2 rounded-md text-sm font-medium';
				case 'main':
				default:
					return 'text-gray-900 bg-gray-50 px-3 py-2 rounded-md text-sm font-medium';
			}
		}
		return 'text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium';
	}
</script>

<nav class="bg-white shadow-md border-b">
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
		<div class="flex justify-between h-16">
			<div class="flex items-center">
				<img src="/static/icons/logo.png" alt="Xanthus Logo" class="w-8 h-8 mr-3">
				<h1 class="text-xl font-semibold text-gray-900">Xanthus</h1>
			</div>
			<div class="flex items-center space-x-4">
				<a href="/main" class={getNavLinkClass('main')}>Dashboard</a>
				<a href="/dns" class={getNavLinkClass('dns')}>DNS Config</a>
				<a href="/vps" class={getNavLinkClass('vps')}>VPS Management</a>
				<a href="/applications" class={getNavLinkClass('applications')}>Applications</a>
				<button 
					on:click={showAboutModal} 
					class="text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
				>
					About
				</button>
				<a href="/logout" class="text-red-600 hover:text-red-800 px-3 py-2 rounded-md text-sm font-medium">
					Logout
				</a>
			</div>
		</div>
	</div>
</nav>