<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { applicationStore, hidePasswordModal } from '$lib/stores/applications';
	import { api } from '$lib/services/api';
	import { setLoading, setLoadingTitle, setLoadingMessage } from '$lib/stores/ui';
	import Button from '$lib/components/common/Button.svelte';
	import Input from '$lib/components/forms/Input.svelte';
	import type { Application } from '../../../app';

	const dispatch = createEventDispatcher();

	// Reactive state from store
	$: ({ modals } = $applicationStore);
	$: modal = modals.password;
	$: show = modal.show;
	$: app = modal.app;
	$: mode = modal.mode;

	// Modal state
	let currentPassword = '';
	let newPassword = '';
	let loading = false;
	let copyButtonText = 'Copy';

	// Reactive computed values
	$: appTypeName = app?.app_type === 'argocd' ? 'ArgoCD' : 'Code-Server';
	$: accessDescription = app?.app_type === 'argocd' 
		? 'You can use this password to access your ArgoCD admin interface'
		: 'You can use this password to access your code-server instance';
	$: openButtonText = app?.app_type === 'argocd' ? 'Open ArgoCD' : 'Open Code-Server';
	$: restartWarning = app?.app_type === 'argocd' 
		? 'This will restart your ArgoCD instance' 
		: 'This will restart your code-server instance';
	
	// Reset state when modal opens
	$: if (show && app) {
		resetModal();
		if (mode === 'view') {
			loadCurrentPassword();
		}
	}

	function resetModal() {
		currentPassword = '';
		newPassword = '';
		loading = false;
		copyButtonText = 'Copy';
	}

	async function loadCurrentPassword() {
		if (!app) return;
		
		loading = true;
		try {
			const data = await api.get(`/applications/${app.id}/password`);
			if (data.password) {
				currentPassword = data.password;
			} else {
				throw new Error(data.error || 'Failed to retrieve password');
			}
		} catch (error) {
			console.error('Error retrieving password:', error);
			dispatch('password-error', error);
		} finally {
			loading = false;
		}
	}

	async function handleChangePassword() {
		if (!app || !newPassword.trim()) return;

		setLoading(true);
		setLoadingTitle('Changing Password');
		setLoadingMessage(`Updating password for "${app.name}"...`);

		try {
			const response = await api.post(`/applications/${app.id}/password`, {
				password: newPassword.trim()
			});
			
			if (response.success) {
				dispatch('password-changed', { app, newPassword: newPassword.trim() });
				closeModal();
			} else {
				throw new Error(response.error || 'Failed to change password');
			}
		} catch (error) {
			console.error('Password change error:', error);
			dispatch('password-error', error);
		} finally {
			setLoading(false);
		}
	}

	async function copyPassword() {
		if (!currentPassword) return;
		
		try {
			await navigator.clipboard.writeText(currentPassword);
			copyButtonText = 'Copied!';
			setTimeout(() => {
				copyButtonText = 'Copy';
			}, 2000);
		} catch (error) {
			console.error('Failed to copy password:', error);
		}
	}

	function closeModal() {
		hidePasswordModal();
	}

	function openApplication() {
		if (app?.url) {
			window.open(app.url, '_blank');
		}
	}

	// Keyboard handling
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			closeModal();
		} else if (event.key === 'Enter' && mode === 'change' && newPassword.trim()) {
			handleChangePassword();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if show && app}
	<div 
		class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" 
		on:click={closeModal}
		on:keydown={(e) => e.key === 'Escape' && closeModal()}
		role="dialog"
		aria-modal="true"
		aria-labelledby="modal-title"
		tabindex="-1"
	>
		<div 
			class="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto" 
			role="document"
		>
			<!-- Header -->
			<div class="p-6 border-b border-gray-200">
				<div class="flex items-center justify-between">
					<div class="flex items-center">
						<div class="text-2xl mr-3">🔐</div>
						<div>
							<h3 id="modal-title" class="text-lg font-medium text-gray-900">
								{mode === 'view' ? 'Current Password' : `Change ${appTypeName} Password`}
							</h3>
							<p class="text-sm text-gray-600">{app.name}</p>
						</div>
					</div>
					<button 
						on:click={closeModal} 
						class="text-gray-400 hover:text-gray-600"
						aria-label="Close modal"
						title="Close modal"
					>
						<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
						</svg>
					</button>
				</div>
			</div>

			<!-- Content -->
			<div class="p-6">
				{#if mode === 'view'}
					<!-- View Password Mode -->
					<div class="space-y-6">
						<p class="text-gray-700">Current password for <strong>{app.name}</strong>:</p>
						
						{#if loading}
							<div class="flex items-center justify-center py-8">
								<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
								<span class="ml-2 text-gray-600">Loading password...</span>
							</div>
						{:else}
							<!-- Password Display -->
							<div class="p-4 bg-blue-50 border border-blue-200 rounded-lg">
								<h4 class="font-medium text-blue-900 mb-2">🔐 Password</h4>
								<div class="flex items-center space-x-2">
									<input 
										type="text" 
										value={currentPassword}
										class="flex-1 px-3 py-2 border border-blue-300 rounded font-mono text-sm bg-white" 
										readonly
									/>
									<Button 
										variant="primary" 
										size="sm"
										on:click={copyPassword}
										class="bg-blue-600 hover:bg-blue-700">
										{copyButtonText}
									</Button>
								</div>
								<p class="text-sm text-blue-700 mt-2">💡 {accessDescription}</p>
							</div>
							
							<!-- Quick Access -->
							<div class="p-4 bg-green-50 border border-green-200 rounded-lg">
								<h4 class="font-medium text-green-900 mb-2">🔗 Quick Access</h4>
								<div class="text-center">
									<Button 
										variant="primary"
										on:click={openApplication}
										class="bg-purple-600 hover:bg-purple-700 inline-flex items-center">
										<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
										</svg>
										{openButtonText}
									</Button>
								</div>
							</div>
						{/if}
					</div>
				{:else}
					<!-- Change Password Mode -->
					<div class="space-y-6">
						<p class="text-gray-700">Change the password for <strong>{app.name}</strong>:</p>
						
						<!-- Warning -->
						<div class="p-3 bg-yellow-50 border border-yellow-200 rounded-md">
							<div class="flex items-center">
								<svg class="w-5 h-5 text-yellow-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
									<path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path>
								</svg>
								<span class="text-sm text-yellow-800">{restartWarning}</span>
							</div>
						</div>
						
						<!-- New Password Input -->
						<div>
							<Input
								label="New Password"
								type="password"
								bind:value={newPassword}
								placeholder="Enter new password..."
								help="Choose a secure password for your application"
								required
								class="font-mono"
							/>
						</div>
					</div>
				{/if}
			</div>

			<!-- Actions -->
			<div class="px-6 py-4 bg-gray-50 border-t border-gray-200 rounded-b-lg">
				<div class="flex justify-end space-x-3">
					<Button variant="outline" on:click={closeModal}>
						{mode === 'view' ? 'Close' : 'Cancel'}
					</Button>
					{#if mode === 'change'}
						<Button 
							variant="primary"
							on:click={handleChangePassword}
							disabled={!newPassword.trim()}
							class="bg-green-600 hover:bg-green-700">
							Change Password
						</Button>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}