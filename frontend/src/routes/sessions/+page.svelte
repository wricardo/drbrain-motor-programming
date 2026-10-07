<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onDestroy, onMount } from 'svelte';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import { describeError, type DescribedError } from '$lib/errors';
	import { sortMaps } from '$lib/maps';
	import { MAPS_QUERY, SESSIONS_QUERY } from '$lib/queries';
	import {
		SESSION_SORTS,
		STATUS_FILTERS,
		STATUS_FILTER_LABELS,
		countByStatus,
		filterSessions,
		isFinished,
		progressText,
		sortSessions,
		statusText,
		type SessionSort,
		type StatusFilter
	} from '$lib/sessions';
	import { absoluteTime, relativeTime } from '$lib/time';
	import type { GameMap, SessionSummary } from '$lib/types';

	/** Server-side maximum for the sessions query. */
	const LIMIT = 500;
	const PAGE = 25;
	const REFRESH_MS = 10_000;

	const client = getContextClient();

	let sessions = $state<SessionSummary[]>([]);
	let maps = $state<GameMap[]>([]);
	let loading = $state(true);
	let error = $state<DescribedError | null>(null);
	let now = $state(new Date());

	let sort = $state<SessionSort>('recent');
	let mapId = $state('');
	let status = $state<StatusFilter>('all');
	let text = $state('');
	let pageSize = $state(PAGE);
	let selected = $state<string[]>([]);

	const counts = $derived(countByStatus(sessions));
	const mapById = $derived(new Map(maps.map((m) => [m.id, m])));
	const shown = $derived(sortSessions(filterSessions(sessions, { status, text }), sort));
	const visible = $derived(shown.slice(0, pageSize));
	const allVisibleSelected = $derived(visible.length > 0 && visible.every((s) => selected.includes(s.id)));

	/** Loads sessions; quiet refreshes keep the table on screen instead of showing "Loading". */
	async function load(quiet = false) {
		if (!quiet) loading = true;
		error = null;
		try {
			const data = await unwrap<{ sessions: SessionSummary[] }>(
				client.query(
					SESSIONS_QUERY,
					{ limit: LIMIT, sort: sort === 'created' ? 'CREATED' : 'RECENT', mapId: mapId || null },
					{ requestPolicy: 'network-only' }
				)
			);
			sessions = data.sessions;
			now = new Date();
		} catch (e) {
			error = describeError(e);
		} finally {
			loading = false;
		}
	}

	let timer: ReturnType<typeof setInterval> | undefined;
	onMount(async () => {
		try {
			const m = await unwrap<{ maps: GameMap[] }>(client.query(MAPS_QUERY, {}));
			maps = sortMaps(m.maps);
		} catch (e) {
			error = describeError(e);
		}
		await load();
		timer = setInterval(() => void load(true), REFRESH_MS);
	});
	onDestroy(() => clearInterval(timer));

	function toggle(id: string) {
		selected = selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id];
	}

	function toggleVisible() {
		const ids = visible.map((s) => s.id);
		selected = allVisibleSelected ? selected.filter((id) => !ids.includes(id)) : [...new Set([...selected, ...ids])];
	}

	function setStatus(s: StatusFilter) {
		status = s;
		pageSize = PAGE;
	}

	function clearFilters() {
		status = 'all';
		text = '';
		pageSize = PAGE;
		if (mapId) {
			mapId = '';
			void load();
		}
	}

	const badge: Record<string, string> = {
		won: 'bg-emerald-100 text-emerald-900',
		lost: 'bg-rose-100 text-rose-900',
		running: 'bg-indigo-600 text-white'
	};
</script>

<svelte:head><title>{pageTitle('Sessions')}</title></svelte:head>

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-8 space-y-5">
	<header class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold text-indigo-950">Sessions</h1>
			<p class="mt-1 text-sm text-slate-700">Every run on this server. Anyone can watch or play any session.</p>
		</div>
		<div class="flex items-center gap-3">
			<span class="text-xs text-slate-600">Auto-refreshes every 10 s</span>
			<button
				type="button"
				onclick={() => load()}
				disabled={loading}
				class="rounded-full border border-indigo-300 bg-white px-4 py-1.5 text-sm font-medium text-indigo-900 hover:bg-indigo-50 disabled:opacity-60"
				>Refresh</button
			>
			<a
				href={selected.length ? `/multi?ids=${selected.join(',')}` : undefined}
				aria-disabled={selected.length === 0}
				class="rounded-full px-4 py-1.5 text-sm font-semibold {selected.length
					? 'bg-indigo-600 text-white hover:bg-indigo-700'
					: 'pointer-events-none bg-indigo-100 text-indigo-400'}"
				data-testid="watch-selected">Watch selected ({selected.length})</a
			>
		</div>
	</header>

	<div class="flex flex-wrap items-center gap-3">
		<div class="flex flex-wrap gap-1.5" role="group" aria-label="Status">
			{#each STATUS_FILTERS as s (s)}
				<button
					type="button"
					aria-pressed={status === s}
					class="rounded-full border px-3 py-1 text-sm {status === s
						? 'border-indigo-600 bg-indigo-600 text-white'
						: 'border-indigo-200 bg-white text-indigo-900 hover:bg-indigo-50'}"
					onclick={() => setStatus(s)}>{STATUS_FILTER_LABELS[s]} <span class="opacity-75">({counts[s]})</span></button
				>
			{/each}
		</div>
		<label class="sr-only" for="f-text">Search sessions</label>
		<input
			id="f-text"
			type="search"
			bind:value={text}
			oninput={() => (pageSize = PAGE)}
			placeholder="Search name, id or map"
			class="w-full sm:w-64 rounded-lg border border-indigo-300 bg-white px-3 py-1.5 text-sm"
		/>
		<label class="sr-only" for="f-map">Map</label>
		<select
			id="f-map"
			bind:value={mapId}
			onchange={(e) => {
				mapId = e.currentTarget.value;
				pageSize = PAGE;
				load();
			}}
			class="rounded-lg border border-indigo-300 bg-white px-3 py-1.5 text-sm"
		>
			<option value="">All maps</option>
			{#each maps as m (m.id)}
				<option value={m.id}>{m.name}</option>
			{/each}
		</select>
		<label class="flex items-center gap-2 text-sm text-indigo-900" for="f-sort">
			Sort
			<select
				id="f-sort"
				bind:value={sort}
				onchange={(e) => {
					const next = e.currentTarget.value as SessionSort;
					const reload = (next === 'created') !== (sort === 'created');
					sort = next;
					if (reload) load();
				}}
				class="rounded-lg border border-indigo-300 bg-white px-3 py-1.5 text-sm"
			>
				{#each SESSION_SORTS as [value, label] (value)}
					<option {value}>{label}</option>
				{/each}
			</select>
		</label>
	</div>

	{#if error}
		<p role="alert" class="rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900">
			<strong>{error.title}.</strong> {error.message}
		</p>
	{/if}

	<p class="text-sm text-slate-600" role="status" aria-live="polite">
		{#if loading}
			Loading sessions…
		{:else}
			Showing {visible.length} of {shown.length}{shown.length !== sessions.length ? ` matching (${sessions.length} loaded)` : ''}{sessions.length >= LIMIT
				? ` — the server returns at most ${LIMIT}; narrow by map to see others`
				: ''}.
		{/if}
	</p>

	{#if !loading && shown.length === 0}
		<p class="rounded-xl border border-indigo-200 bg-white px-4 py-6 text-sm text-slate-600">
			{#if sessions.length === 0}
				No sessions yet.
				<a href="/" class="ml-2 inline-block rounded-full bg-indigo-600 px-4 py-1.5 font-semibold text-white hover:bg-indigo-700">Start a map</a>
			{:else}
				No sessions match these filters.
				<button type="button" class="ml-2 rounded-full bg-indigo-600 px-4 py-1.5 font-semibold text-white hover:bg-indigo-700" onclick={clearFilters}>Clear filters</button>
			{/if}
		</p>
	{:else if shown.length > 0}
		<div class="overflow-x-auto rounded-xl border border-indigo-200 bg-white">
			<table class="w-full text-left text-sm">
				<caption class="sr-only">Sessions</caption>
				<thead class="bg-indigo-50 text-xs uppercase tracking-wide text-indigo-900">
					<tr>
						<th scope="col" class="w-8 px-3 py-2">
							<input type="checkbox" class="h-4 w-4 accent-indigo-600" checked={allVisibleSelected} onchange={toggleVisible} aria-label="Select all shown sessions" />
						</th>
						<th scope="col" class="px-4 py-2">Player</th>
						<th scope="col" class="px-4 py-2">Map</th>
						<th scope="col" class="px-4 py-2">Status</th>
						<th scope="col" class="px-4 py-2">Progress</th>
						<th scope="col" class="px-4 py-2">{sort === 'created' ? 'Created' : 'Last activity'}</th>
						<th scope="col" class="px-4 py-2"><span class="sr-only">Actions</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-indigo-100">
					{#each visible as s (s.id)}
						{@const st = statusText(s)}
						{@const at = sort === 'created' ? s.createdAt : s.lastActionAt}
						<tr class={selected.includes(s.id) ? 'bg-indigo-50/60' : ''}>
							<td class="px-3 py-2">
								<input
									type="checkbox"
									class="h-4 w-4 accent-indigo-600"
									checked={selected.includes(s.id)}
									onchange={() => toggle(s.id)}
									aria-label="Select {s.displayName || 'Anonymous'} on {s.map.name}"
								/>
							</td>
							<td class="px-4 py-2 font-medium">{s.displayName || 'Anonymous'}</td>
							<td class="px-4 py-2">{s.map.name}</td>
							<td class="px-4 py-2">
								<span class="rounded-full px-2 py-0.5 text-xs {badge[st] ?? 'bg-indigo-50 text-indigo-800'}">{st}</span>
							</td>
							<td class="px-4 py-2 text-slate-700">{progressText(s, mapById.get(s.mapId))}</td>
							<td class="px-4 py-2 text-slate-700" title={absoluteTime(at)}>{relativeTime(at, now)}</td>
							<td class="px-4 py-2 whitespace-nowrap text-right">
								{#if !isFinished(s)}
									<a class="font-semibold text-indigo-700 underline" href="/play/{s.id}">Play</a>
									<span aria-hidden="true" class="text-slate-400"> · </span>
								{/if}
								<a class="text-indigo-700 underline" href="/watch/{s.id}">Watch</a>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if shown.length > visible.length}
			<div class="flex justify-center">
				<button
					type="button"
					class="rounded-full border border-indigo-300 bg-white px-5 py-2 text-sm font-medium text-indigo-900 hover:bg-indigo-50"
					onclick={() => (pageSize += PAGE)}>Load more ({shown.length - visible.length} more)</button
				>
			</div>
		{/if}
	{/if}
</div>
