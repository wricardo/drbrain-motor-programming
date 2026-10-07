<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import AiPromptCard from '$lib/components/AiPromptCard.svelte';
	import CallStack from '$lib/components/CallStack.svelte';
	import ConnectionBadge from '$lib/components/ConnectionBadge.svelte';
	import Grid from '$lib/components/Grid.svelte';
	import Stats from '$lib/components/Stats.svelte';
	import Tape from '$lib/components/Tape.svelte';
	import { buildSessionPrompt } from '$lib/prompt';
	import { statusLine } from '$lib/sessions';
	import { createSessionStore } from '$lib/stores/session';
	import { createSessionSource } from '$lib/stores/source';

	const sessionId = $page.params.id ?? '';
	const store = createSessionStore(createSessionSource(getContextClient()), sessionId);
	onDestroy(() => store.destroy());

	const origin = typeof window !== 'undefined' ? window.location.origin : '';
	const session = $derived($store.session);
	const status = $derived(session?.vm.status ?? 'READY');
</script>

<svelte:head><title>{pageTitle(session ? `Watching ${session.map.name}` : 'Watch')}</title></svelte:head>

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-6">
	{#if $store.loading}
		<p role="status" class="text-sm text-slate-600">Loading session…</p>
	{:else if !session}
		<div role="alert" class="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-900">
			<strong>{$store.error?.title ?? 'Session unavailable'}.</strong>
			{$store.error?.message}
			<a href="/" class="ml-2 underline">Back to maps</a>
		</div>
	{:else}
		<div class="flex flex-wrap items-center justify-between gap-2">
			<div>
				<h1 class="text-2xl font-semibold text-indigo-950">Watching: {session.map.name}</h1>
				<p class="text-sm text-slate-600">{session.displayName || 'Anonymous'} · read-only spectator view</p>
			</div>
			<ConnectionBadge state={$store.connection} />
		</div>
		{#if $store.error && $store.connection !== 'live'}
			<p role="status" class="mt-2 text-xs text-amber-800">{$store.error.title}: retrying…</p>
		{/if}

		<div class="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,28rem)]">
			<section aria-label="Board" class="space-y-3 lg:sticky lg:top-4 lg:self-start">
				<Grid map={session.map} vm={session.vm} lastEvent={session.lastEvent} />
				<p class="text-sm text-slate-600">{session.map.description}</p>
			</section>
			<section aria-label="Program" class="space-y-4">
				<p
					class="rounded-lg px-3 py-2 text-sm font-semibold {status === 'WON'
						? 'bg-emerald-50 text-emerald-900'
						: status === 'LOST'
							? 'bg-rose-50 text-rose-900'
							: session.playing
								? 'bg-indigo-600 text-white'
								: 'bg-indigo-50 text-indigo-900'}"
					role="status"
					aria-live="polite"
					data-testid="watch-status"
				>
					{statusLine(session)}
				</p>
				<Stats {session} />
				<div class="flex flex-wrap items-center gap-3">
					<a href="/play/{session.id}" class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700" data-testid="take-over">Take over</a>
					<a href="/" class="text-sm text-indigo-700 underline">Pick your own map</a>
				</div>
				<div class="rounded-xl border border-indigo-200 bg-white p-3">
					<Tape
						program={session.program}
						disabled
						callStack={session.vm.callStack}
						lastEvent={session.lastEvent}
						{status}
					/>
				</div>
				<CallStack callStack={session.vm.callStack} maxCallDepth={session.map.maxCallDepth} />
				<AiPromptCard
					prompt={buildSessionPrompt(origin, session)}
					blurb="Copy this prompt into an AI chat and it will program the robot to win this session."
				/>
			</section>
		</div>
	{/if}
</div>
