<script lang="ts">
	export let variant: 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let disabled = false;
	export let loading = false;
	export let href: string | undefined = undefined;
	export let type: 'button' | 'submit' | 'reset' = 'button';
	let className = '';
	export { className as class };

	function getVariantClasses(variant: string): string {
		switch (variant) {
			case 'primary':
				return 'bg-blue-600 hover:bg-blue-700 text-white';
			case 'secondary':
				return 'bg-gray-600 hover:bg-gray-700 text-white';
			case 'success':
				return 'bg-green-600 hover:bg-green-700 text-white';
			case 'danger':
				return 'bg-red-600 hover:bg-red-700 text-white';
			case 'warning':
				return 'bg-yellow-600 hover:bg-yellow-700 text-white';
			case 'info':
				return 'bg-purple-600 hover:bg-purple-700 text-white';
			default:
				return 'bg-blue-600 hover:bg-blue-700 text-white';
		}
	}

	function getSizeClasses(size: string): string {
		switch (size) {
			case 'sm':
				return 'px-3 py-1.5 text-sm';
			case 'md':
				return 'px-4 py-2 text-sm';
			case 'lg':
				return 'px-6 py-3 text-base';
			default:
				return 'px-4 py-2 text-sm';
		}
	}

	$: baseClasses = 'inline-flex items-center justify-center rounded-md font-medium transition duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2';
	$: variantClasses = getVariantClasses(variant);
	$: sizeClasses = getSizeClasses(size);
	$: disabledClasses = disabled || loading ? 'opacity-50 cursor-not-allowed' : '';
	$: allClasses = `${baseClasses} ${variantClasses} ${sizeClasses} ${disabledClasses} ${className}`;
</script>

{#if href}
	<a {href} class={allClasses} class:pointer-events-none={disabled || loading}>
		{#if loading}
			<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
		{/if}
		<slot />
	</a>
{:else}
	<button {type} class={allClasses} disabled={disabled || loading} on:click>
		{#if loading}
			<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
		{/if}
		<slot />
	</button>
{/if}