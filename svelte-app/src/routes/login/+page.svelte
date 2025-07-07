<script>
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { authStore } from '$lib/stores/auth';
  import { errorHandler } from '$lib/services/errorHandler';
  import Button from '$lib/components/common/Button.svelte';
  import Card from '$lib/components/common/Card.svelte';

  let cloudflareToken = '';
  let loading = false;
  let errors = {};

  // Redirect if already authenticated
  onMount(async () => {
    // Wait a bit for auth store to initialize
    setTimeout(() => {
      if ($authStore.isAuthenticated) {
        goto('/app');
      }
    }, 100);
  });

  async function handleLogin() {
    loading = true;
    errors = {};

    try {
      // Validate input
      if (!cloudflareToken.trim()) {
        errors.cf_token = 'Cloudflare API token is required';
        loading = false;
        return;
      }

      const success = await authStore.login(cloudflareToken);
      
      if (success) {
        errorHandler.showSuccess('Login Successful', 'Welcome back! Redirecting to dashboard...');
        setTimeout(() => {
          goto('/app');
        }, 1000);
      } else {
        errors.cf_token = 'Invalid Cloudflare API token. Please check your token and try again.';
      }
    } catch (error) {
      console.error('Login error:', error);
      errors.cf_token = error.message || 'Login failed. Please try again.';
    } finally {
      loading = false;
    }
  }

  function handleInputChange() {
    // Clear error when user starts typing
    if (errors.cf_token) {
      errors = { ...errors, cf_token: '' };
    }
  }
</script>

<svelte:head>
  <title>Xanthus - Login</title>
  <meta name="description" content="Login to Xanthus K3s Deployment Tool" />
</svelte:head>

<div class="min-h-screen bg-gray-100 flex items-center justify-center p-4">
  <Card class="w-full max-w-md">
    <div class="text-center mb-8">
      <img src="/static/icons/logo.png" alt="Xanthus Logo" class="w-24 h-24 mx-auto mb-4">
      <h1 class="text-3xl font-bold text-gray-900 mb-2">Xanthus</h1>
      <p class="text-gray-600">K3s Deployment Tool</p>
    </div>

    <form on:submit|preventDefault={handleLogin} class="space-y-4">
      <div>
        <label for="cf_token" class="block text-sm font-medium text-gray-700 mb-2">
          Cloudflare API Token
        </label>
        <input 
          type="password" 
          id="cf_token" 
          bind:value={cloudflareToken}
          on:input={handleInputChange}
          disabled={loading}
          required
          class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-50 disabled:cursor-not-allowed"
          class:border-red-300={errors.cf_token}
          class:focus:ring-red-500={errors.cf_token}
          class:focus:border-red-500={errors.cf_token}
          placeholder="Enter your Cloudflare API token"
        >
        <p class="mt-1 text-xs text-gray-500">
          Need a token? <a href="https://dash.cloudflare.com/profile/api-tokens" target="_blank" class="text-blue-600 hover:underline">Create one here</a>
        </p>
        {#if errors.cf_token}
          <p class="mt-1 text-sm text-red-600">{errors.cf_token}</p>
        {/if}
      </div>

      <Button 
        type="submit" 
        {loading}
        class="w-full"
        variant="primary"
      >
        {#if loading}
          Verifying API token...
        {:else}
          Login
        {/if}
      </Button>
    </form>

    <div class="mt-6 text-center">
      <p class="text-xs text-gray-500">
        Your API token is used to verify access to Cloudflare services and is stored securely.
      </p>
    </div>
  </Card>
</div>

<style>
  /* Custom focus styles for better UX */
  input:focus {
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }
  
  input.border-red-300:focus {
    box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
  }
</style>