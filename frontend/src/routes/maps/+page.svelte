<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import MapCard from '$lib/components/MapCard.svelte';
	import { describeError, type DescribedError } from '$lib/errors';
	import { DIFFICULTY_FILTERS, countByDifficulty, filterMaps, sortMaps, type DifficultyFilter } from '$lib/maps';
	import { CREATE_SESSION_MUTATION, MAPS_QUERY } from '$lib/queries';
	import { rememberSession, savedDisplayName } from '$lib/sessions';
	import type { GameMap, Session } from '$lib/types';

	const client = getContextClient();
	let maps = $state<GameMap[]>([]);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state<DescribedError | null>(null);
	let text = $state('');
	let difficulty = $state<DifficultyFilter>('all');

	const counts = $derived(countByDifficulty(maps));
	const shown = $derived(filterMaps(maps, { text, difficulty }));
	const chipLabel: Record<DifficultyFilter, string> = { all: 'All', easy: 'Easy', medium: 'Medium', hard: 'Hard' };

	onMount(async () => {
		try {
			maps = sortMaps((await unwrap<{ maps: GameMap[] }>(client.query(MAPS_QUERY, {}))).maps);
		} catch (e) {
			error = describeError(e);
		} finally {
			loading = false;
		}
	});

	async function play(map: GameMap) {
		busy = true;
		error = null;
		try {
			const data = await unwrap<{ createSession: Session }>(
				client.mutation(CREATE_SESSION_MUTATION, { mapID: map.id, displayName: savedDisplayName().trim() || null })
			);
			rememberSession(data.createSession.id);
			await goto(`/play/${data.createSession.id}`);
		} catch (e) {
			error = describeError(e);
		} finally {
			busy = false;
		}
	}

	function clearFilters() {
		text = '';
		difficulty = 'all';
	}
</script>

<svelte:head><title>{pageTitle('Maps')}</title></svelte:head>

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-8">
	<div class="flex items-center justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold text-indigo-950">Maps</h1>
			<p class="mt-1 text-sm text-slate-700">Ordered from easy to hard. Press Play to start a new session.</p>
		</div>
		<a href="/editor" class="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700">New map</a>
	</div>

	<div class="mt-5 flex flex-wrap items-center gap-3">
		<label class="sr-only" for="map-search">Search maps</label>
		<input
			id="map-search"
			type="search"
			bind:value={text}
			placeholder="Search name or description"
			class="w-full sm:w-72 rounded-lg border border-indigo-300 bg-white px-3 py-2 text-sm"
		/>
		<div class="flex flex-wrap gap-1.5" role="group" aria-label="Difficulty">
			{#each DIFFICULTY_FILTERS as d (d)}
				<button
					type="button"
					aria-pressed={difficulty === d}
					class="rounded-full border px-3 py-1 text-sm {difficulty === d
						? 'border-indigo-600 bg-indigo-600 text-white'
						: 'border-indigo-200 bg-white text-indigo-900 hover:bg-indigo-50'}"
					onclick={() => (difficulty = d)}>{chipLabel[d]} <span class="opacity-75">({counts[d]})</span></button
				>
			{/each}
		</div>
	</div>

	{#if error}
		<p role="alert" class="mt-4 rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900">
			<strong>{error.title}.</strong> {error.message}
		</p>
	{/if}
	{#if loading}
		<p class="mt-6 text-sm text-slate-600" role="status">Loading maps…</p>
	{:else if maps.length === 0 && !error}
		<p class="mt-6 rounded-xl border border-indigo-200 bg-white px-4 py-6 text-sm text-slate-600">
			No maps yet.
			<a href="/editor" class="ml-2 inline-block rounded-full bg-indigo-600 px-4 py-1.5 font-semibold text-white hover:bg-indigo-700">Create a map</a>
		</p>
	{:else if shown.length === 0}
		<p class="mt-6 rounded-xl border border-indigo-200 bg-white px-4 py-6 text-sm text-slate-600">
			No maps match these filters.
			<button type="button" class="ml-2 rounded-full bg-indigo-600 px-4 py-1.5 font-semibold text-white hover:bg-indigo-700" onclick={clearFilters}>Clear filters</button>
		</p>
	{:else}
		<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each shown as map (map.id)}
				<MapCard {map} {busy} onplay={play} editHref="/editor?map={encodeURIComponent(map.id)}" />
			{/each}
		</div>
	{/if}
</div>
