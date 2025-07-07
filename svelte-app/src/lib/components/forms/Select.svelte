<script lang="ts">
	export let label: string = '';
	export let id: string = '';
	export let name: string = '';
	export let value: string | number = '';
	export let options: { value: string | number; label: string; disabled?: boolean }[] = [];
	export let placeholder: string = '';
	export let required: boolean = false;
	export let disabled: boolean = false;
	export let loading: boolean = false;
	export let error: string = '';
	export let helperText: string = '';
	export let help: string = '';
	export let size: 'sm' | 'md' | 'lg' = 'md';

	// Allow custom class overrides
	let className: string = '';
	export { className as class };

	function getSizeClasses(size: string): string {
		switch (size) {
			case 'sm':
				return 'px-3 py-1.5 text-sm';
			case 'md':
				return 'px-4 py-2 text-sm';
			case 'lg':
				return 'px-4 py-3 text-base';
			default:
				return 'px-4 py-2 text-sm';
		}
	}

	$: sizeClasses = getSizeClasses(size);
	$: selectClasses = `
		block w-full rounded-md border-gray-300 shadow-sm 
		focus:border-blue-500 focus:ring-blue-500 
		disabled:bg-gray-50 disabled:text-gray-500
		${sizeClasses}
		${error ? 'border-red-300 focus:border-red-500 focus:ring-red-500' : ''}
		${className}
	`.trim();
</script>

<div class="space-y-1">
	{#if label}
		<label for={id} class="block text-sm font-medium text-gray-700">
			{label}
			{#if required}
				<span class="text-red-500">*</span>
			{/if}
		</label>
	{/if}
	
	<div class="relative">
		<select
			{id}
			{name}
			{required}
			disabled={disabled || loading}
			bind:value
			class={selectClasses}
			on:change
			on:blur
			on:focus
		>
			{#if placeholder}
				<option value="" disabled selected={value === ''}>{placeholder}</option>
			{/if}
			
			{#if loading}
				<option value="" disabled>Loading options...</option>
			{:else}
				{#each options as option}
					<option value={option.value} disabled={option.disabled}>
						{option.label}
					</option>
				{/each}
			{/if}
		</select>
		
		{#if loading}
			<div class="absolute inset-y-0 right-0 flex items-center pr-8 pointer-events-none">
				<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
			</div>
		{/if}
	</div>
	
	{#if error}
		<p class="text-sm text-red-600">{error}</p>
	{:else if help || helperText}
		<p class="text-sm text-gray-500">{help || helperText}</p>
	{/if}
</div>