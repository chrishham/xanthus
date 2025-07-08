<script>
	import { auth } from '$lib/stores/auth.js';
	import { goto } from '$app/navigation';
	
	let cloudflareToken = '';
	let loading = false;
	let error = '';

	async function handleLogin() {
		if (!cloudflareToken.trim()) {
			error = 'Please enter your Cloudflare API token';
			return;
		}

		loading = true;
		error = '';

		try {
			await auth.login(cloudflareToken);
			goto('/');
		} catch (err) {
			error = 'Login failed: ' + err.message;
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Login - Xanthus</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
			<h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
				Sign in to Xanthus
			</h2>
			<p class="mt-2 text-center text-sm text-gray-600">
				Enter your Cloudflare API token to continue
			</p>
		</div>
		
		<form class="mt-8 space-y-6" on:submit|preventDefault={handleLogin}>
			<div>
				<label for="cloudflare-token" class="sr-only">Cloudflare API Token</label>
				<input
					id="cloudflare-token"
					name="cloudflare-token"
					type="password"
					required
					class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
					placeholder="Cloudflare API Token"
					bind:value={cloudflareToken}
					disabled={loading}
				/>
			</div>

			{#if error}
				<div class="text-red-600 text-sm text-center">
					{error}
				</div>
			{/if}

			<div>
				<button
					type="submit"
					disabled={loading}
					class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{#if loading}
						<span class="flex items-center">
							<svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Signing in...
						</span>
					{:else}
						Sign in
					{/if}
				</button>
			</div>
		</form>
	</div>
</div>