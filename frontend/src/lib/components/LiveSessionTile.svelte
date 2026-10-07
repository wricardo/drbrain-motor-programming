<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import { getContextClient } from '@urql/svelte';
	import ConnectionBadge from './ConnectionBadge.svelte';
	import Grid from './Grid.svelte';
	import { statusLine } from '$lib/sessions';
	import { createSessionStore } from '$lib/stores/session';
	import { createSessionSource } from '$lib/stores/source';

	/** Compact live board for the multi-watch grid; one subscription per tile. */
	let { id }: { id: string } = $props();

	const store = createSessionStore(createSessionSource(getContextClient()), untrack(() => id)); // tiles are keyed by id
	onDestroy(() => store.destroy());

	const session = $derived($store.session);
	const status = $derived(session?.vm.status ?? 'READY');
</script>

<article class="flex flex-col gap-2 rounded-xl border border-indigo-200 bg-white p-3 shadow-sm" data-testid="multi-tile">
	{#if $store.loading}
		<p role="status" class="text-sm text-slate-600">Loading {id}…</p>
	{:else if !session}
		<p role="alert" class="text-sm text-rose-900">
			<strong>{$store.error?.title ?? 'Session unavailable'}.</strong>
			<span class="font-mono text-xs">{id}</span>
		</p>
	{:else}
		<div class="flex items-start justify-between gap-2">
			<div class="min-w-0">
				<h2 class="truncate font-semibold text-indigo-950">{session.map.name}</h2>
				<p class="truncate text-xs text-slate-600">{session.displayName || 'Anonymous'}</p>
			</div>
			<ConnectionBadge state={$store.connection} />
		</div>
		<p
			class="rounded-lg px-2 py-1 text-xs font-medium {status === 'WON'
				? 'bg-emerald-50 text-emerald-900'
				: status === 'LOST'
					? 'bg-rose-50 text-rose-900'
					: 'bg-indigo-50 text-indigo-900'}"
			data-testid="tile-status"
		>
			{statusLine(session)}
		</p>
		<p class="text-xs text-slate-700">
			Steps <strong>{session.vm.steps} / {session.map.maxSteps}</strong> · Treats
			<strong>{session.map.treats.length - session.vm.treatsRemaining.length} / {session.map.treats.length}</strong>
		</p>
		<div class="flex h-64 items-center justify-center"><Grid map={session.map} vm={session.vm} lastEvent={session.lastEvent} compact /></div>
		<a href="/watch/{session.id}" class="mt-auto text-sm font-medium text-indigo-700 underline">Open full watch page →</a>
	{/if}
</article>
