<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import MapCard from '$lib/components/MapCard.svelte';
	import { describeError, type DescribedError } from '$lib/errors';
	import { CREATE_SESSION_MUTATION, MAPS_QUERY } from '$lib/queries';
	import type { GameMap, Session } from '$lib/types';

	const client = getContextClient();
	let maps = $state<GameMap[]>([]);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state<DescribedError | null>(null);

	onMount(async () => {
		try {
			maps = (await unwrap<{ maps: GameMap[] }>(client.query(MAPS_QUERY, {}))).maps;
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
				client.mutation(CREATE_SESSION_MUTATION, { mapID: map.id, displayName: null })
			);
			await goto(`/play/${data.createSession.id}`);
		} catch (e) {
			error = describeError(e);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>{pageTitle('Maps')}</title></svelte:head>

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-8">
	<div class="flex items-center justify-between gap-3">
		<h1 class="text-2xl font-semibold text-indigo-950">Maps</h1>
		<a href="/editor" class="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700">New map</a>
	</div>
	{#if error}
		<p role="alert" class="mt-4 rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900">
			<strong>{error.title}.</strong> {error.message}
		</p>
	{/if}
	{#if loading}
		<p class="mt-6 text-sm text-slate-600" role="status">Loading maps…</p>
	{:else}
		<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each maps as map (map.id)}
				<MapCard {map} {busy} onplay={play} editHref="/editor?map={encodeURIComponent(map.id)}" />
			{/each}
		</div>
	{/if}
</div>
