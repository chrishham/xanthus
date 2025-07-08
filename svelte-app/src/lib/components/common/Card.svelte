<script lang="ts">
	export let variant: 'default' | 'blue' | 'green' | 'purple' | 'orange' | 'red' = 'default';
	export let padding: 'sm' | 'md' | 'lg' = 'md';
	export let shadow: 'sm' | 'md' | 'lg' | 'none' = 'md';
	let className = '';
	export { className as class };

	function getVariantClasses(variant: string): string {
		switch (variant) {
			case 'blue':
				return 'bg-blue-50 border-blue-200';
			case 'green':
				return 'bg-green-50 border-green-200';
			case 'purple':
				return 'bg-purple-50 border-purple-200';
			case 'orange':
				return 'bg-orange-50 border-orange-200';
			case 'red':
				return 'bg-red-50 border-red-200';
			case 'default':
			default:
				return 'bg-white border-gray-200';
		}
	}

	function getPaddingClasses(padding: string): string {
		switch (padding) {
			case 'sm':
				return 'p-4';
			case 'md':
				return 'p-6';
			case 'lg':
				return 'p-8';
			default:
				return 'p-6';
		}
	}

	function getShadowClasses(shadow: string): string {
		switch (shadow) {
			case 'sm':
				return 'shadow-sm';
			case 'md':
				return 'shadow-md';
			case 'lg':
				return 'shadow-lg';
			case 'none':
				return '';
			default:
				return 'shadow-md';
		}
	}

	$: variantClasses = getVariantClasses(variant);
	$: paddingClasses = getPaddingClasses(padding);
	$: shadowClasses = getShadowClasses(shadow);
	$: allClasses = `rounded-lg border ${variantClasses} ${paddingClasses} ${shadowClasses} ${className}`;
</script>

<div class={allClasses}>
	<slot />
</div>