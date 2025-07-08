<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { setupStore } from '$lib/stores/setup';
  import { authStore } from '$lib/stores/auth';
  import { errorHandler } from '$lib/services/errorHandler';
  import Button from '$lib/components/common/Button.svelte';
  import Card from '$lib/components/common/Card.svelte';

  let currentStep = 1;
  let hetznerToken = '';
  let cloudflareToken = '';
  let loading = false;
  let errors: { hetzner_token?: string; cloudflare_token?: string } = {};
  let setupStatus: any = null;

  // Subscribe to setup store
  $: setupConfig = $setupStore;
  $: currentStep = setupConfig.step;

  onMount(async () => {
    // Check if user is already authenticated and setup is complete
    if ($authStore.isAuthenticated) {
      const status = await setupStore.checkSetupStatus();
      if (!status.isSetupRequired) {
        goto('/');
        return;
      }
      setupStatus = status;
    } else {
      // If not authenticated, still check setup status to show current step
      setupStatus = await setupStore.checkSetupStatus();
    }
  });

  async function handleHetznerSetup() {
    loading = true;
    errors = {};

    try {
      if (!hetznerToken.trim()) {
        errors.hetzner_token = 'Hetzner API token is required';
        loading = false;
        return;
      }

      const success = await setupStore.configureHetzner(hetznerToken);
      if (success) {
        // Move to step 2 (Cloudflare setup)
        setupStore.nextStep();
      }
    } catch (error) {
      console.error('Hetzner setup error:', error);
      errors.hetzner_token = (error as Error).message || 'Failed to configure Hetzner token';
    } finally {
      loading = false;
    }
  }

  async function handleCloudflareSetup() {
    loading = true;
    errors = {};

    try {
      if (!cloudflareToken.trim()) {
        errors.cloudflare_token = 'Cloudflare API token is required';
        loading = false;
        return;
      }

      const success = await setupStore.configureCloudflare(cloudflareToken);
      if (success) {
        // Setup complete, redirect to app
        setTimeout(() => {
          goto('/');
        }, 1500);
      }
    } catch (error) {
      console.error('Cloudflare setup error:', error);
      errors.cloudflare_token = (error as Error).message || 'Failed to configure Cloudflare token';
    } finally {
      loading = false;
    }
  }

  function handleInputChange(field: keyof typeof errors) {
    // Clear error when user starts typing
    if (errors[field]) {
      errors = { ...errors, [field]: '' };
    }
  }

  function getProgressWidth() {
    return `${(currentStep / 3) * 100}%`;
  }

  function getStepTitle() {
    switch (currentStep) {
      case 1:
        return 'Hetzner Configuration';
      case 2:
        return 'Cloudflare Configuration';
      case 3:
        return 'Setup Complete';
      default:
        return 'Setup';
    }
  }
</script>

<svelte:head>
  <title>Xanthus - Setup</title>
  <meta name="description" content="First-time setup for Xanthus K3s Deployment Tool" />
</svelte:head>

<div class="min-h-screen bg-gray-100 flex items-center justify-center p-4">
  <Card class="w-full max-w-md">
    <div class="text-center mb-8">
      <img src="/static/icons/logo.png" alt="Xanthus Logo" class="w-20 h-20 mx-auto mb-4">
      <h1 class="text-3xl font-bold text-gray-900 mb-2">Xanthus Setup</h1>
      <p class="text-gray-600">First-time configuration</p>
    </div>

    <!-- Progress indicator -->
    <div class="mb-6">
      <div class="flex items-center justify-between text-sm">
        <span class="text-blue-600 font-medium">Step {currentStep} of 3</span>
        <span class="text-gray-500">{getStepTitle()}</span>
      </div>
      <div class="mt-2 w-full bg-gray-200 rounded-full h-2">
        <div class="bg-blue-600 h-2 rounded-full transition-all duration-300" style="width: {getProgressWidth()}"></div>
      </div>
    </div>

    {#if currentStep === 1}
      <!-- Step 1: Hetzner Configuration -->
      <div class="mb-6">
        <div class="bg-blue-50 border border-blue-200 rounded-md p-4">
          <div class="flex">
            <div class="flex-shrink-0">
              <svg class="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
              </svg>
            </div>
            <div class="ml-3">
              <h3 class="text-sm font-medium text-blue-800">Hetzner API Key Required</h3>
              <div class="mt-2 text-sm text-blue-700">
                <p>We need your Hetzner Cloud API key to provision VPS instances for your K3s cluster.</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <form on:submit|preventDefault={handleHetznerSetup} class="space-y-4">
        <div>
          <label for="hetzner_token" class="block text-sm font-medium text-gray-700 mb-2">
            Hetzner Cloud API Token
          </label>
          
          {#if setupStatus?.hasHetznerToken}
            <div class="mb-3 p-3 bg-green-50 border border-green-200 rounded-md">
              <p class="text-sm text-green-800 font-medium">✓ API Key already configured</p>
              <p class="text-xs text-green-600 mt-1">You can continue with the current key or enter a new one below.</p>
            </div>
          {/if}
          
          <input 
            type="password" 
            id="hetzner_token" 
            bind:value={hetznerToken}
            on:input={() => handleInputChange('hetzner_token')}
            disabled={loading}
            required={!setupStatus?.hasHetznerToken}
            class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-50 disabled:cursor-not-allowed"
            class:border-red-300={errors.hetzner_token}
            class:focus:ring-red-500={errors.hetzner_token}
            class:focus:border-red-500={errors.hetzner_token}
            placeholder={setupStatus?.hasHetznerToken ? "Enter new Hetzner API token" : "Enter your Hetzner API token"}
          >
          <p class="mt-1 text-xs text-gray-500">
            Need a token? <a href="https://console.hetzner.cloud/" target="_blank" class="text-blue-600 hover:underline">Create one in Hetzner Console</a>
          </p>
          {#if errors.hetzner_token}
            <p class="mt-1 text-sm text-red-600">{errors.hetzner_token}</p>
          {/if}
        </div>

        <Button 
          type="submit" 
          {loading}
          class="w-full"
          variant="primary"
        >
          {#if loading}
            Configuring Hetzner...
          {:else if setupStatus?.hasHetznerToken}
            Continue with Hetzner Setup
          {:else}
            Configure Hetzner
          {/if}
        </Button>
      </form>

    {:else if currentStep === 2}
      <!-- Step 2: Cloudflare Configuration -->
      <div class="mb-6">
        <div class="bg-blue-50 border border-blue-200 rounded-md p-4">
          <div class="flex">
            <div class="flex-shrink-0">
              <svg class="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
              </svg>
            </div>
            <div class="ml-3">
              <h3 class="text-sm font-medium text-blue-800">Cloudflare API Token Required</h3>
              <div class="mt-2 text-sm text-blue-700">
                <p>We need your Cloudflare API token for DNS management and SSL certificate provisioning.</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <form on:submit|preventDefault={handleCloudflareSetup} class="space-y-4">
        <div>
          <label for="cloudflare_token" class="block text-sm font-medium text-gray-700 mb-2">
            Cloudflare API Token
          </label>
          <input 
            type="password" 
            id="cloudflare_token" 
            bind:value={cloudflareToken}
            on:input={() => handleInputChange('cloudflare_token')}
            disabled={loading}
            required
            class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-50 disabled:cursor-not-allowed"
            class:border-red-300={errors.cloudflare_token}
            class:focus:ring-red-500={errors.cloudflare_token}
            class:focus:border-red-500={errors.cloudflare_token}
            placeholder="Enter your Cloudflare API token"
          >
          <p class="mt-1 text-xs text-gray-500">
            Need a token? <a href="https://dash.cloudflare.com/profile/api-tokens" target="_blank" class="text-blue-600 hover:underline">Create one here</a>
          </p>
          {#if errors.cloudflare_token}
            <p class="mt-1 text-sm text-red-600">{errors.cloudflare_token}</p>
          {/if}
        </div>

        <div class="flex space-x-3">
          <Button 
            type="button" 
            on:click={() => setupStore.previousStep()}
            disabled={loading}
            class="flex-1"
            variant="secondary"
          >
            Back
          </Button>
          <Button 
            type="submit" 
            {loading}
            class="flex-1"
            variant="primary"
          >
            {#if loading}
              Completing Setup...
            {:else}
              Complete Setup
            {/if}
          </Button>
        </div>
      </form>

    {:else if currentStep === 3}
      <!-- Step 3: Setup Complete -->
      <div class="text-center">
        <div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-green-100 mb-4">
          <svg class="h-6 w-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
          </svg>
        </div>
        <h3 class="text-lg font-medium text-gray-900 mb-2">Setup Complete!</h3>
        <p class="text-sm text-gray-500 mb-6">Your Xanthus platform is now configured and ready to use.</p>
        
        <Button 
          on:click={() => goto('/')}
          class="w-full"
          variant="primary"
        >
          Go to Dashboard
        </Button>
      </div>
    {/if}

    <div class="mt-6 text-center">
      <p class="text-xs text-gray-500">
        Your API tokens are stored securely and used only for managing your cloud resources.
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