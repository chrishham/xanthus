<script lang="ts">
	import Button from './Button.svelte';

	export let title: string;
	export let description: string;
	export let variant: 'blue' | 'green' | 'purple' | 'orange' | 'red' = 'blue';
	export let buttonText: string = 'Manage';
	export let buttonHref: string | undefined = undefined;
	export let onClick: (() => void) | undefined = undefined;

	function getVariantClasses(variant: string): string {
		switch (variant) {
			case 'blue':
				return 'bg-blue-50';
			case 'green':
				return 'bg-green-50';
			case 'purple':
				return 'bg-purple-50';
			case 'orange':
				return 'bg-orange-50';
			case 'red':
				return 'bg-red-50';
			default:
				return 'bg-blue-50';
		}
	}

	function getTitleClasses(variant: string): string {
		switch (variant) {
			case 'blue':
				return 'text-blue-900';
			case 'green':
				return 'text-green-900';
			case 'purple':
				return 'text-purple-900';
			case 'orange':
				return 'text-orange-900';
			case 'red':
				return 'text-red-900';
			default:
				return 'text-blue-900';
		}
	}

	function getDescriptionClasses(variant: string): string {
		switch (variant) {
			case 'blue':
				return 'text-blue-700';
			case 'green':
				return 'text-green-700';
			case 'purple':
				return 'text-purple-700';
			case 'orange':
				return 'text-orange-700';
			case 'red':
				return 'text-red-700';
			default:
				return 'text-blue-700';
		}
	}

	function getButtonVariant(variant: string): 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info' {
		switch (variant) {
			case 'blue':
				return 'primary';
			case 'green':
				return 'success';
			case 'purple':
				return 'info';
			case 'orange':
				return 'warning';
			case 'red':
				return 'danger';
			default:
				return 'primary';
		}
	}

	$: cardClasses = `${getVariantClasses(variant)} p-6 rounded-lg`;
	$: titleClasses = `text-lg font-semibold ${getTitleClasses(variant)} mb-2`;
	$: descriptionClasses = `${getDescriptionClasses(variant)} text-sm mb-4`;
	$: buttonVariant = getButtonVariant(variant);
</script>

<div class={cardClasses}>
	<h3 class={titleClasses}>{title}</h3>
	<p class={descriptionClasses}>{description}</p>
	
	{#if buttonHref}
		<Button variant={buttonVariant} href={buttonHref}>
			{buttonText}
		</Button>
	{:else}
		<Button variant={buttonVariant} on:click={onClick}>
			{buttonText}
		</Button>
	{/if}
</div>