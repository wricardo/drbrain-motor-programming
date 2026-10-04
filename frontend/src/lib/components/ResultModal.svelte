<script lang="ts">
	import { lossReasonText } from '$lib/errors';
	import type { LossReason, Status } from '$lib/types';

	let {
		status,
		reason,
		steps,
		onreset,
		onclose
	}: {
		status: Status;
		reason: LossReason | null;
		steps: number;
		onreset: () => void;
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
		class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl text-center"
		data-testid="result-modal"
	>
		<h2 id="result-title" class="text-2xl font-bold {status === 'WON' ? 'text-emerald-700' : 'text-rose-700'}">
			{status === 'WON' ? 'You won!' : 'Run lost'}
		</h2>
		<p class="mt-2 text-sm text-slate-700">
			{#if status === 'WON'}
				Every treat collected in {steps} steps.
			{:else}
				{lossReasonText(reason)}
			{/if}
		</p>
		<div class="mt-5 flex justify-center gap-2">
			<button
				bind:this={primary}
				type="button"
				class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
				onclick={onreset}>Reset &amp; try again</button
			>
			<button
				type="button"
				class="rounded-lg border border-indigo-300 px-4 py-2 text-sm text-indigo-900 hover:bg-indigo-50"
				onclick={onclose}>Close</button
			>
		</div>
	</div>
</div>
