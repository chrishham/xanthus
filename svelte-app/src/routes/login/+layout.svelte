<script>
  import { onMount } from 'svelte';
  import { authStore } from '$lib/stores/auth';
  import NotificationSystem from '$lib/components/common/NotificationSystem.svelte';
  import '../../app.css';

  // Initialize auth store on mount
  onMount(() => {
    authStore.initialize();
  });
</script>

<svelte:head>
  <link rel="icon" type="image/x-icon" href="/static/icons/favicon.ico">
  <link rel="icon" type="image/png" sizes="32x32" href="/static/icons/favicon-32x32.png">
  <link rel="icon" type="image/png" sizes="16x16" href="/static/icons/favicon-16x16.png">
  <link rel="apple-touch-icon" sizes="180x180" href="/static/icons/apple-touch-icon.png">
</svelte:head>

<main>
  <slot />
</main>

<NotificationSystem />