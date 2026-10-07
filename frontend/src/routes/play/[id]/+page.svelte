<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onDestroy, onMount, untrack } from 'svelte';
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import AiPromptCard from '$lib/components/AiPromptCard.svelte';
	import CallStack from '$lib/components/CallStack.svelte';
	import ConnectionBadge from '$lib/components/ConnectionBadge.svelte';
	import Grid from '$lib/components/Grid.svelte';
	import Palette from '$lib/components/Palette.svelte';
	import ResultModal from '$lib/components/ResultModal.svelte';
	import ShortcutsDialog from '$lib/components/ShortcutsDialog.svelte';
	import Stats from '$lib/components/Stats.svelte';
	import Tape from '$lib/components/Tape.svelte';
	import { describeError, lossReasonText, type DescribedError } from '$lib/errors';
	import { globalKeyAction } from '$lib/globalKeys';
	import { nextMap } from '$lib/maps';
	import {
		CREATE_SESSION_MUTATION,
		MAPS_QUERY,
		PAUSE_MUTATION,
		RESET_MUTATION,
		RUN_MUTATION,
		SET_PROGRAM_MUTATION,
		STEP_MUTATION
	} from '$lib/queries';
	import { rememberSession, statusLine } from '$lib/sessions';
	import { cloneProgram, programsEqual } from '$lib/program';
	import { buildSessionPrompt } from '$lib/prompt';
	import { createSessionStore } from '$lib/stores/session';
	import { createSessionSource } from '$lib/stores/source';
	import type { GameMap, Instruction, Program, Session } from '$lib/types';

	const sessionId = $page.params.id ?? '';
	const client = getContextClient();
	const store = createSessionStore(createSessionSource(client), sessionId);
	onDestroy(() => store.destroy());

	const origin = typeof window !== 'undefined' ? window.location.origin : '';
	const session = $derived($store.session);
	const connection = $derived($store.connection);

	let draft = $state<Program | null>(null);
	let selected = $state<Instruction | null>(null);
	let speedMs = $state(500);
	let busy = $state(false);
	let actionError = $state<DescribedError | null>(null);
	let dismissedAttempts = $state<number | null>(null);
	let showShortcuts = $state(false);
	let maps = $state<GameMap[]>([]);
	let initialised = false;
	let lastServer: Program | null = null;

	// Keep the editable draft in sync with the server program unless the user has unsaved edits.
	$effect(() => {
		const sp = session?.program;
		if (!sp) return;
		untrack(() => {
			if (draft === null) draft = cloneProgram(sp);
			else if (lastServer && programsEqual(draft, lastServer) && !programsEqual(draft, sp)) draft = cloneProgram(sp);
			lastServer = cloneProgram(sp);
		});
	});

	$effect(() => {
		const s = session;
		if (!s || initialised) return;
		initialised = true;
		speedMs = Math.max(50, Math.min(5000, s.speedMs || 500));
		// Do not pop the result modal for an outcome that was already there on load.
		if (s.vm.status === 'WON' || s.vm.status === 'LOST') dismissedAttempts = s.attempts;
	});

	onMount(async () => {
		try {
			maps = (await unwrap<{ maps: GameMap[] }>(client.query(MAPS_QUERY, {}))).maps;
		} catch {
			// "Next map" is hidden without the map list
		}
	});
	const next = $derived(session ? nextMap(maps, session.mapId) : null);

	const callKeys = $derived(
		!draft?.subs.length ? '' : draft.subs.length === 1 ? '1 calls the subroutine,' : `1–${draft.subs.length} call subroutines,`
	);
	const dirty = $derived(!!draft && !!session && !programsEqual(draft, session.program));
	const status = $derived(session?.vm.status ?? 'READY');
	const terminal = $derived(status === 'WON' || status === 'LOST');
	const playing = $derived(session?.playing ?? false);
	const showModal = $derived(terminal && !!session && dismissedAttempts !== session.attempts);
	const resultText = $derived(
		!session || !terminal
			? ''
			: status === 'WON'
				? `You won in ${session.vm.steps} steps.`
				: `Run lost. ${lossReasonText(session.vm.lossReason)}`
	);

	async function act(fn: () => Promise<Session | void>) {
		busy = true;
		actionError = null;
		try {
			const s = await fn();
			if (s) store.apply(s);
		} catch (e) {
			actionError = describeError(e);
		} finally {
			busy = false;
		}
	}

	/** Saves the draft when it differs from the server program and applies the result to the store. */
	async function saveIfDirty(): Promise<void> {
		if (!draft || !session || programsEqual(draft, session.program)) return;
		const data = await unwrap<{ setProgram: Session }>(
			client.mutation(SET_PROGRAM_MUTATION, { sessionID: sessionId, program: draft })
		);
		store.apply(data.setProgram);
	}

	const run = () =>
		act(async () => {
			await saveIfDirty();
			return (await unwrap<{ run: Session }>(client.mutation(RUN_MUTATION, { sessionID: sessionId, speedMs }))).run;
		});
	const pause = () =>
		act(async () => (await unwrap<{ pause: Session }>(client.mutation(PAUSE_MUTATION, { sessionID: sessionId }))).pause);
	const stepOnce = () =>
		act(async () => {
			await saveIfDirty();
			return (await unwrap<{ step: Session }>(client.mutation(STEP_MUTATION, { sessionID: sessionId }))).step;
		});
	const reset = () =>
		act(async () => {
			const s = (await unwrap<{ reset: Session }>(client.mutation(RESET_MUTATION, { sessionID: sessionId }))).reset;
			dismissedAttempts = s.attempts;
			return s;
		});
	const save = () => act(saveIfDirty);

	/** Starts a session on the next map (standard order) with the same player name. */
	const playNext = () =>
		act(async () => {
			if (!next || !session) return;
			const data = await unwrap<{ createSession: Session }>(
				client.mutation(CREATE_SESSION_MUTATION, { mapID: next.id, displayName: session.displayName || null })
			);
			rememberSession(data.createSession.id);
			// Full navigation: this page keeps one session store for its lifetime.
			window.location.assign(`/play/${data.createSession.id}`);
		});

	const shortcutKeys = $derived<[string, string][]>([
		['R', 'Reset the robot (when no slot is focused)'],
		['?', 'Show this list'],
		['Esc', 'Close a dialog or menu'],
		['Tab', 'Focus a program slot'],
		['F', 'Slot: move forward'],
		['L / R', 'Slot: turn left / right'],
		...(draft?.subs.length ? [[draft.subs.length === 1 ? '1' : `1–${draft.subs.length}`, 'Slot: call a subroutine'] as [string, string]] : []),
		['Backspace', 'Slot: clear'],
		['Arrows', 'Slot: move between slots'],
		['Enter', 'Slot: open the instruction menu']
	]);

	function onWindowKey(e: KeyboardEvent) {
		if (showShortcuts || showModal) return;
		const action = globalKeyAction(e);
		if (!action) return;
		e.preventDefault();
		if (action === 'shortcuts') showShortcuts = true;
		else if (!busy) reset();
	}

	function revert() {
		if (session) draft = cloneProgram(session.program);
	}

	const btn =
		'rounded-lg px-4 py-2 text-sm font-semibold disabled:opacity-50 disabled:cursor-not-allowed';
</script>

<svelte:head><title>{pageTitle(session?.map.name)}</title></svelte:head>
<svelte:window onkeydown={onWindowKey} />

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
				<h1 class="text-2xl font-semibold text-indigo-950">{session.map.name}</h1>
				<p class="text-sm text-slate-600">
					{session.displayName || 'Anonymous'} ·
					<a class="underline" href="/watch/{session.id}">spectator link</a>
				</p>
			</div>
			<ConnectionBadge state={connection} />
		</div>

		{#if actionError}
			<p role="alert" class="mt-3 rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900" data-testid="action-error">
				<strong>{actionError.title}</strong>{#if actionError.code}<span class="ml-1 font-mono text-xs">[{actionError.code}]</span>{/if}
				{#if actionError.message}— {actionError.message}{/if}
			</p>
		{/if}

		<div class="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,28rem)]">
			<section aria-label="Board" class="space-y-3 lg:sticky lg:top-4 lg:self-start">
				<Grid map={session.map} vm={session.vm} lastEvent={session.lastEvent} />
				<p class="text-sm text-slate-600">{session.map.description}</p>
				<p class="sr-only" role="status" aria-live="polite" data-testid="result-live">{resultText}</p>
			</section>

			<section aria-label="Program" class="space-y-4">
				<p
					class="rounded-lg px-3 py-2 text-sm font-semibold {status === 'WON'
						? 'bg-emerald-50 text-emerald-900'
						: status === 'LOST'
							? 'bg-rose-50 text-rose-900'
							: playing
								? 'bg-indigo-600 text-white'
								: 'bg-indigo-50 text-indigo-900'}"
					data-testid="status-line"
				>
					{statusLine({ playing, speedMs: session.speedMs, vm: session.vm })}
				</p>
				<Stats {session} />

				<div class="rounded-xl border border-indigo-200 bg-white p-3 space-y-3">
					<div class="flex flex-wrap gap-2" role="group" aria-label="Run controls">
						{#if playing}
							<button type="button" class="{btn} bg-amber-500 text-white hover:bg-amber-600" disabled={busy} onclick={pause}>Pause</button>
						{:else}
							<button type="button" class="{btn} bg-indigo-600 text-white hover:bg-indigo-700" disabled={busy || terminal} onclick={run}>Run</button>
						{/if}
						<button type="button" class="{btn} border border-indigo-300 text-indigo-900 hover:bg-indigo-50" disabled={busy || playing || terminal} onclick={stepOnce}>Step</button>
						<button type="button" class="{btn} border border-indigo-300 text-indigo-900 hover:bg-indigo-50" disabled={busy} onclick={reset}>Reset</button>
						{#if dirty}
							<button type="button" class="{btn} border border-emerald-400 text-emerald-900 hover:bg-emerald-50" disabled={busy || playing} onclick={save}>Save program</button>
							<button type="button" class="{btn} text-slate-700 underline" disabled={busy || playing} onclick={revert}>Revert</button>
						{/if}
					</div>
					<div>
						<label for="speed" class="flex justify-between text-xs text-slate-700">
							<span>Delay per step</span><span data-testid="speed-label">{speedMs} ms</span>
						</label>
						<input id="speed" type="range" min="50" max="5000" step="50" bind:value={speedMs} disabled={playing} class="w-full accent-indigo-600" />
					</div>
					{#if dirty}
						<p class="text-xs text-amber-800" data-testid="dirty-note">Unsaved changes — saved automatically on Run/Step (this resets the robot to the start).</p>
					{/if}
					{#if playing}
						<p class="text-xs text-slate-600">Editing is locked while the program runs. Pause to edit.</p>
					{/if}
				</div>

				{#if draft}
					<div class="rounded-xl border border-indigo-200 bg-white p-3 space-y-3">
						<Palette bind:selected subCount={draft.subs.length} disabled={playing} />
						<Tape
							program={draft}
							{selected}
							disabled={playing}
							callStack={dirty ? [] : session.vm.callStack}
							lastEvent={dirty ? null : session.lastEvent}
							{status}
							onchange={(p) => (draft = p)}
						/>
					</div>
				{/if}

				<CallStack callStack={session.vm.callStack} maxCallDepth={session.map.maxCallDepth} />

				<div class="rounded-xl border border-indigo-200 bg-white p-4" data-testid="help">
					<h2 class="text-xs font-semibold uppercase tracking-widest text-indigo-900">Help</h2>
					<ul class="mt-2 list-disc space-y-1 pl-5 text-sm text-slate-700">
						<li>Click a slot to pick an instruction, or arm one in the palette and click slots to stamp it.</li>
						<li>Drag to move an instruction (hold <kbd>Alt</kbd> to copy).</li>
						<li>With a slot focused: <kbd>F</kbd> forward, <kbd>L</kbd>/<kbd>R</kbd> turn, {callKeys} <kbd>Backspace</kbd> clear, arrows move.</li>
						<li><kbd>R</kbd> resets the robot when no slot is focused.</li>
						<li>Press <kbd>?</kbd> for shortcuts.</li>
					</ul>
					<a href="/learn" class="mt-2 inline-block text-sm font-medium text-indigo-700 underline">All rules →</a>
				</div>

				<AiPromptCard
					prompt={buildSessionPrompt(origin, session)}
					blurb="Copy this prompt into an AI chat and it will program the robot to win this session."
				/>
			</section>
		</div>

		{#if showModal}
			<ResultModal
				{status}
				reason={session.vm.lossReason}
				steps={session.vm.steps}
				maxSteps={session.map.maxSteps}
				bestSteps={session.bestSteps}
				treats={{ collected: session.map.treats.length - session.vm.treatsRemaining.length, total: session.map.treats.length }}
				nextName={next?.name ?? null}
				{busy}
				onreset={reset}
				onnext={playNext}
				onclose={() => (dismissedAttempts = session.attempts)}
			/>
		{/if}
		{#if showShortcuts}
			<ShortcutsDialog keys={shortcutKeys} onclose={() => (showShortcuts = false)} />
		{/if}
	{/if}
</div>
