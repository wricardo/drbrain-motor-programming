<script lang="ts">
	import type { GameMap } from '$lib/types';
	import MiniGrid from './MiniGrid.svelte';

	let {
		map,
		busy = false,
		onplay,
		editHref = null
	}: { map: GameMap; busy?: boolean; onplay?: (map: GameMap) => void; editHref?: string | null } = $props();

	const badge: Record<string, string> = {
		easy: 'bg-emerald-100 text-emerald-800',
		medium: 'bg-amber-100 text-amber-800',
		hard: 'bg-rose-100 text-rose-800'
	};
</script>

<article class="flex flex-col gap-3 rounded-xl border border-indigo-200 bg-white p-4 shadow-sm" data-testid="map-card">
	<div class="flex justify-center"><MiniGrid layout={map.layout} label="Preview of {map.name}" /></div>
	<div>
		<h3 class="font-semibold text-indigo-950">{map.name}</h3>
		<p class="text-sm text-slate-600 line-clamp-3">{map.description}</p>
	</div>
	<div class="flex flex-wrap gap-1.5 text-xs">
		<span class="rounded-full px-2 py-0.5 {badge[map.difficulty] ?? 'bg-slate-100 text-slate-700'}">{map.difficulty}</span>
		{#if map.allowRecursion}
			<span class="rounded-full bg-violet-100 text-violet-800 px-2 py-0.5">recursion</span>
		{/if}
		<span class="rounded-full bg-indigo-50 text-indigo-800 px-2 py-0.5">{map.width}×{map.height}</span>
		<span class="rounded-full bg-indigo-50 text-indigo-800 px-2 py-0.5"
			>main {map.mainTapeLength}{map.subTapeLengths.length ? ` + subs ${map.subTapeLengths.join('/')}` : ''}</span
		>
		<span class="rounded-full bg-indigo-50 text-indigo-800 px-2 py-0.5">{map.treats.length} {map.treats.length === 1 ? 'treat' : 'treats'}</span>
		<span class="rounded-full bg-indigo-50 text-indigo-800 px-2 py-0.5">≤ {map.maxSteps} steps</span>
	</div>
	<div class="mt-auto flex gap-2">
		{#if onplay}
			<button
				type="button"
				class="flex-1 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
				disabled={busy}
				onclick={() => onplay?.(map)}>Play</button
			>
		{/if}
		{#if editHref}
			<a href={editHref} class="rounded-lg border border-indigo-300 px-3 py-2 text-sm text-indigo-800 hover:bg-indigo-50">Edit</a>
			<a href="{editHref}&duplicate=1" class="rounded-lg border border-indigo-300 px-3 py-2 text-sm text-indigo-800 hover:bg-indigo-50">Duplicate</a>
		{/if}
	</div>
</article>
