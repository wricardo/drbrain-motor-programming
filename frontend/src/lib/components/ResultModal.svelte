<script lang="ts">
	import { lossReasonText } from '$lib/errors';
	import type { LossReason, Status } from '$lib/types';

	let {
		status,
		reason,
		steps,
		maxSteps,
		bestSteps = null,
		treats = null,
		nextName = null,
		busy = false,
		onreset,
		onnext,
		onclose
	}: {
		status: Status;
		reason: LossReason | null;
		steps: number;
		maxSteps: number;
		bestSteps?: number | null;
		/** Collected / total treats, shown on a loss. */
		treats?: { collected: number; total: number } | null;
		/** Name of the next map in the standard order; null hides "Next map". */
		nextName?: string | null;
		busy?: boolean;
		onreset: () => void;
		onnext?: () => void;
		onclose: () => void;
	} = $props();

	let primary: HTMLButtonElement | undefined = $state();
	$effect(() => {
		primary?.focus();
	});

	function onKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') onclose();
	}
</script>

<svelte:window onkeydown={onKeyDown} />

<div class="fixed inset-0 z-40 flex items-center justify-center bg-indigo-950/50 p-4">
	<div
		role="dialog"
		aria-modal="true"
		aria-labelledby="result-title"
		class="relative w-full max-w-md rounded-2xl bg-white p-6 shadow-xl text-center"
		data-testid="result-modal"
	>
		<button
			type="button"
			class="absolute right-3 top-3 rounded-full px-2 text-lg leading-none text-slate-500 hover:bg-slate-100 hover:text-slate-800"
			aria-label="Close and keep the board visible"
			onclick={onclose}>×</button
		>
		<h2 id="result-title" class="text-2xl font-bold {status === 'WON' ? 'text-emerald-700' : 'text-rose-700'}">
			{status === 'WON' ? 'You won!' : 'Run lost'}
		</h2>
		<p class="mt-2 text-sm text-slate-700">
			{status === 'WON' ? 'Every treat collected.' : lossReasonText(reason)}
		</p>
		<dl class="mt-4 flex justify-center gap-6 text-sm" data-testid="result-numbers">
			<div>
				<dt class="text-xs text-slate-600">Steps</dt>
				<dd class="font-semibold tabular-nums">{steps} / {maxSteps}</dd>
			</div>
			{#if treats && status !== 'WON'}
				<div>
					<dt class="text-xs text-slate-600">Treats</dt>
					<dd class="font-semibold tabular-nums">{treats.collected} / {treats.total}</dd>
				</div>
			{/if}
			<div>
				<dt class="text-xs text-slate-600">Best</dt>
				<dd class="font-semibold tabular-nums">{bestSteps != null ? `${bestSteps} steps` : '—'}</dd>
			</div>
		</dl>
		<div class="mt-5 flex flex-wrap justify-center gap-2">
			<button
				bind:this={primary}
				type="button"
				class="rounded-lg px-4 py-2 text-sm font-semibold {status === 'WON'
					? 'border border-indigo-300 text-indigo-900 hover:bg-indigo-50'
					: 'bg-indigo-600 text-white hover:bg-indigo-700'}"
				disabled={busy}
				onclick={onreset}>Try again</button
			>
			{#if nextName && onnext}
				<button
					type="button"
					class="rounded-lg px-4 py-2 text-sm font-semibold disabled:opacity-50 {status === 'WON'
						? 'bg-indigo-600 text-white hover:bg-indigo-700'
						: 'border border-indigo-300 text-indigo-900 hover:bg-indigo-50'}"
					title={nextName}
					disabled={busy}
					onclick={onnext}>Next map</button
				>
			{/if}
			<a href="/maps" class="rounded-lg border border-indigo-300 px-4 py-2 text-sm text-indigo-900 hover:bg-indigo-50">All maps</a>
		</div>
		{#if nextName}
			<p class="mt-3 text-xs text-slate-600">Next: {nextName}</p>
		{/if}
	</div>
</div>
