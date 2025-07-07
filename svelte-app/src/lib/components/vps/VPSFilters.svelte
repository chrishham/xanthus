<script lang="ts">
	import { vpsStore, setVPSFilters, setVPSSort } from '$lib/stores/vps';
	import Input from '../forms/Input.svelte';
	import Select from '../forms/Select.svelte';

	$: state = $vpsStore;
	$: filters = state.filters;
	$: sort = state.sort;

	const providerOptions = [
		{ value: '', label: 'All Providers' },
		{ value: 'hetzner', label: 'Hetzner Cloud' },
		{ value: 'oracle', label: 'Oracle Cloud' }
	];

	const statusOptions = [
		{ value: '', label: 'All Statuses' },
		{ value: 'running', label: 'Running' },
		{ value: 'stopped', label: 'Stopped' },
		{ value: 'starting', label: 'Starting' },
		{ value: 'stopping', label: 'Stopping' },
		{ value: 'rebooting', label: 'Rebooting' }
	];

	const sortOptions = [
		{ value: 'name', label: 'Name' },
		{ value: 'status', label: 'Status' },
		{ value: 'provider', label: 'Provider' },
		{ value: 'created_at', label: 'Created Date' },
		{ value: 'monthly_cost', label: 'Monthly Cost' }
	];

	function handleFilterChange(key: keyof typeof filters, value: string) {
		setVPSFilters({ [key]: value });
	}

	function handleSortChange(field: string) {
		const direction = sort.field === field && sort.direction === 'asc' ? 'desc' : 'asc';
		setVPSSort(field, direction);
	}

	function clearFilters() {
		setVPSFilters({
			provider: '',
			status: '',
			search: ''
		});
	}

	$: hasActiveFilters = filters.provider || filters.status || filters.search;
</script>

<div class="bg-white shadow-sm border border-gray-200 rounded-lg p-6">
	<div class="flex flex-col lg:flex-row lg:items-center lg:justify-between space-y-4 lg:space-y-0 lg:space-x-6">
		<!-- Filters -->
		<div class="flex flex-col sm:flex-row space-y-4 sm:space-y-0 sm:space-x-4 flex-1">
			<!-- Search -->
			<div class="flex-1 min-w-0">
				<Input
					type="text"
					placeholder="Search servers by name or IP..."
					value={filters.search}
					on:input={(e) => handleFilterChange('search', e.detail)}
				/>
			</div>

			<!-- Provider Filter -->
			<div class="w-full sm:w-48">
				<Select
					options={providerOptions}
					value={filters.provider}
					on:change={(e) => handleFilterChange('provider', e.detail)}
				/>
			</div>

			<!-- Status Filter -->
			<div class="w-full sm:w-48">
				<Select
					options={statusOptions}
					value={filters.status}
					on:change={(e) => handleFilterChange('status', e.detail)}
				/>
			</div>
		</div>

		<!-- Sort and Actions -->
		<div class="flex items-center space-x-4">
			<!-- Sort -->
			<div class="flex items-center space-x-2">
				<span class="text-sm text-gray-700">Sort by:</span>
				<div class="relative">
					<select
						class="appearance-none bg-white border border-gray-300 rounded-md px-3 py-2 pr-8 text-sm focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
						value={sort.field}
						on:change={(e) => handleSortChange(e.currentTarget.value)}
					>
						{#each sortOptions as option}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
					<div class="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
						<svg class="h-4 w-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4" />
						</svg>
					</div>
				</div>

				<!-- Sort Direction -->
				<button
					class="inline-flex items-center px-2 py-1 text-sm text-gray-600 hover:text-gray-900 transition-colors"
					on:click={() => handleSortChange(sort.field)}
					title="Toggle sort direction"
				>
					{#if sort.direction === 'asc'}
						<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
						</svg>
					{:else}
						<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h9m5-4v12m0 0l-4-4m4 4l4-4" />
						</svg>
					{/if}
				</button>
			</div>

			<!-- Clear Filters -->
			{#if hasActiveFilters}
				<button
					class="inline-flex items-center px-3 py-2 text-sm text-gray-600 hover:text-gray-900 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
					on:click={clearFilters}
				>
					<svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
					Clear
				</button>
			{/if}
		</div>
	</div>

	<!-- Active filters summary -->
	{#if hasActiveFilters}
		<div class="mt-4 flex flex-wrap items-center gap-2">
			<span class="text-sm text-gray-500">Active filters:</span>
			
			{#if filters.search}
				<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
					Search: "{filters.search}"
					<button
						class="ml-1.5 inline-flex items-center justify-center w-4 h-4 rounded-full text-purple-400 hover:text-purple-600 hover:bg-purple-200"
						on:click={() => handleFilterChange('search', '')}
					>
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</span>
			{/if}

			{#if filters.provider}
				<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
					Provider: {providerOptions.find(o => o.value === filters.provider)?.label}
					<button
						class="ml-1.5 inline-flex items-center justify-center w-4 h-4 rounded-full text-purple-400 hover:text-purple-600 hover:bg-purple-200"
						on:click={() => handleFilterChange('provider', '')}
					>
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</span>
			{/if}

			{#if filters.status}
				<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
					Status: {statusOptions.find(o => o.value === filters.status)?.label}
					<button
						class="ml-1.5 inline-flex items-center justify-center w-4 h-4 rounded-full text-purple-400 hover:text-purple-600 hover:bg-purple-200"
						on:click={() => handleFilterChange('status', '')}
					>
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</span>
			{/if}
		</div>
	{/if}
</div>