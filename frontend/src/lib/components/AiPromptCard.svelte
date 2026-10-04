<script lang="ts">
	let {
		prompt,
		title = 'Play with an AI',
		blurb
	}: { prompt: string; title?: string; blurb: string } = $props();

	let copied = $state(false);
	let failed = $state(false);
	let open = $state(false);
	let area = $state<HTMLTextAreaElement | null>(null);
	let timer: ReturnType<typeof setTimeout> | undefined;

	async function copy() {
		failed = false;
		try {
			await navigator.clipboard.writeText(prompt);
			copied = true;
			clearTimeout(timer);
			timer = setTimeout(() => (copied = false), 2000);
		} catch {
			// Clipboard API unavailable (insecure context / permissions): show the text selected instead.
			failed = true;
			open = true;
			queueMicrotask(() => area?.select());
		}
	}
</script>

<div class="rounded-xl border border-indigo-200 bg-white p-4" data-testid="ai-prompt-card">
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<h2 class="text-xs font-semibold uppercase tracking-widest text-indigo-900">{title}</h2>
			<p class="mt-1 text-sm text-slate-700">{blurb}</p>
		</div>
		<button
			type="button"
			onclick={copy}
			class="shrink-0 rounded-full border px-4 py-2 text-sm font-semibold transition-colors {copied
				? 'border-emerald-300 bg-emerald-50 text-emerald-800'
				: 'border-indigo-300 text-indigo-800 hover:bg-indigo-50'}"
			data-testid="ai-prompt-copy">{copied ? 'Copied!' : 'Copy prompt'}</button
		>
	</div>
	<p class="sr-only" role="status" aria-live="polite">{copied ? 'Prompt copied to clipboard' : ''}</p>
	{#if failed}
		<p role="alert" class="mt-2 text-xs text-amber-800">Could not access the clipboard — the prompt is selected below, press Ctrl/Cmd+C.</p>
	{/if}
	<details class="mt-3" bind:open>
		<summary class="cursor-pointer select-none text-xs font-medium text-slate-700 hover:text-slate-900">Show full prompt</summary>
		<textarea
			bind:this={area}
			readonly
			rows="14"
			aria-label="Prompt for an AI"
			value={prompt}
			class="mt-2 w-full resize-y rounded-lg border border-indigo-100 bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-700"
			onclick={(e) => (e.currentTarget as HTMLTextAreaElement).select()}
		></textarea>
	</details>
</div>
