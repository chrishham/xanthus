<script lang="ts">
	export let label: string = '';
	export let id: string = '';
	export let name: string = '';
	export let type: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url' | 'search' = 'text';
	export let value: string | number = '';
	export let placeholder: string = '';
	export let required: boolean = false;
	export let disabled: boolean = false;
	export let readonly: boolean = false;
	export let error: string = '';
	export let helperText: string = '';
	export let help: string = '';
	export let size: 'sm' | 'md' | 'lg' = 'md';

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
	$: inputClasses = `
		block w-full rounded-md border-gray-300 shadow-sm 
		focus:border-blue-500 focus:ring-blue-500 
		disabled:bg-gray-50 disabled:text-gray-500
		${sizeClasses}
		${error ? 'border-red-300 focus:border-red-500 focus:ring-red-500' : ''}
	`;
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
	
	<input
		{id}
		{name}
		{type}
		{placeholder}
		{required}
		{disabled}
		{readonly}
		bind:value
		class={inputClasses}
		on:input
		on:change
		on:blur
		on:focus
	/>
	
	{#if error}
		<p class="text-sm text-red-600">{error}</p>
	{:else if help || helperText}
		<p class="text-sm text-gray-500">{help || helperText}</p>
	{/if}
</div>