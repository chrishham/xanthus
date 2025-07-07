<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { errorHandler, type ErrorNotification } from '$lib/services/errorHandler';
	import { fly, fade } from 'svelte/transition';

	let notifications: ErrorNotification[] = [];
	let unsubscribe: (() => void) | null = null;

	onMount(() => {
		unsubscribe = errorHandler.subscribe((updatedNotifications) => {
			notifications = updatedNotifications;
		});
	});

	onDestroy(() => {
		if (unsubscribe) {
			unsubscribe();
		}
	});

	function getNotificationClasses(type: string): string {
		const baseClasses = 'rounded-lg shadow-lg p-4 border-l-4';
		
		switch (type) {
			case 'error':
				return `${baseClasses} bg-red-50 border-red-500 text-red-800`;
			case 'warning':
				return `${baseClasses} bg-yellow-50 border-yellow-500 text-yellow-800`;
			case 'success':
				return `${baseClasses} bg-green-50 border-green-500 text-green-800`;
			case 'info':
				return `${baseClasses} bg-blue-50 border-blue-500 text-blue-800`;
			default:
				return `${baseClasses} bg-gray-50 border-gray-500 text-gray-800`;
		}
	}

	function getIconForType(type: string): string {
		switch (type) {
			case 'error':
				return 'M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z';
			case 'warning':
				return 'M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z';
			case 'success':
				return 'M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z';
			case 'info':
				return 'M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z';
			default:
				return 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z';
		}
	}

	function formatTime(timestamp: number): string {
		const now = Date.now();
		const diff = now - timestamp;
		
		if (diff < 60000) { // Less than 1 minute
			return 'Just now';
		} else if (diff < 3600000) { // Less than 1 hour
			const minutes = Math.floor(diff / 60000);
			return `${minutes}m ago`;
		} else if (diff < 86400000) { // Less than 1 day
			const hours = Math.floor(diff / 3600000);
			return `${hours}h ago`;
		} else {
			return new Date(timestamp).toLocaleDateString();
		}
	}

	function dismiss(id: string) {
		errorHandler.dismiss(id);
	}

	function executeAction(action: () => void, id: string) {
		action();
		dismiss(id);
	}
</script>

<!-- Notification Container -->
{#if notifications.length > 0}
	<div class="fixed top-4 right-4 z-50 space-y-3 max-w-sm w-full" role="alert" aria-live="polite">
		{#each notifications as notification (notification.id)}
			<div
				class={getNotificationClasses(notification.type)}
				in:fly={{ x: 300, duration: 300 }}
				out:fade={{ duration: 200 }}
			>
				<div class="flex items-start">
					<!-- Icon -->
					<div class="flex-shrink-0">
						<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
							<path fill-rule="evenodd" d={getIconForType(notification.type)} clip-rule="evenodd" />
						</svg>
					</div>
					
					<!-- Content -->
					<div class="ml-3 flex-1">
						<div class="flex items-center justify-between">
							<h4 class="text-sm font-medium">{notification.title}</h4>
							<div class="flex items-center space-x-2">
								<span class="text-xs opacity-70">{formatTime(notification.timestamp)}</span>
								{#if notification.dismissible}
									<button
										on:click={() => dismiss(notification.id)}
										class="text-current opacity-70 hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-current rounded"
										aria-label="Dismiss notification"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								{/if}
							</div>
						</div>
						
						<p class="text-sm mt-1 opacity-90">{notification.message}</p>
						
						<!-- Actions -->
						{#if notification.actions && notification.actions.length > 0}
							<div class="mt-3 flex space-x-2">
								{#each notification.actions as action}
									<button
										on:click={() => executeAction(action.action, notification.id)}
										class="text-xs px-3 py-1 rounded-md border {action.primary 
											? 'bg-current text-white border-current hover:bg-opacity-90' 
											: 'border-current text-current hover:bg-current hover:bg-opacity-10'} 
											focus:outline-none focus:ring-2 focus:ring-current focus:ring-offset-2 transition-colors"
									>
										{action.label}
									</button>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			</div>
		{/each}
		
		<!-- Clear All Button (when multiple notifications) -->
		{#if notifications.length > 1}
			<div class="text-center">
				<button
					on:click={() => errorHandler.clearAll()}
					class="text-xs text-gray-500 hover:text-gray-700 underline focus:outline-none focus:ring-2 focus:ring-gray-500 rounded"
				>
					Clear all notifications
				</button>
			</div>
		{/if}
	</div>
{/if}

<style>
	/* Ensure proper z-index and positioning */
	:global(.notification-container) {
		pointer-events: none;
	}
	
	:global(.notification-container > *) {
		pointer-events: auto;
	}
</style>