<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { page } from '$app/stores';
	import LiveSessionTile from '$lib/components/LiveSessionTile.svelte';

	/** At most this many live boards (one subscription each). */
	const MAX = 12;
	const ids = $derived(
		[
			...new Set(
				($page.url.searchParams.get('ids') ?? '')
					.split(',')
					.map((s) => s.trim().toLowerCase())
					.filter((s) => /^[0-9a-f]{16}$/.test(s))
			)
		].slice(0, MAX)
	);
</script>

<svelte:head><title>{pageTitle('Watch sessions')}</title></svelte:head>

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-6">
	<div class="flex flex-wrap items-baseline justify-between gap-2">
		<div>
			<h1 class="text-xl font-semibold text-indigo-950">Watching {ids.length} {ids.length === 1 ? 'session' : 'sessions'}</h1>
			<p class="text-xs text-slate-600">Live, read-only spectator view</p>
		</div>
		<a href="/sessions" class="text-sm text-indigo-700 underline">Back to Sessions</a>
	</div>

	{#if ids.length === 0}
		<p class="mt-6 rounded-xl border border-indigo-200 bg-white px-4 py-6 text-sm text-slate-600" data-testid="multi-empty">
			Pick sessions on the Sessions page.
			<a href="/sessions" class="ml-2 inline-block rounded-full bg-indigo-600 px-4 py-1.5 font-semibold text-white hover:bg-indigo-700">Go to Sessions</a>
		</p>
	{:else}
		<div class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each ids as id (id)}
				<LiveSessionTile {id} />
			{/each}
		</div>
	{/if}
</div>
