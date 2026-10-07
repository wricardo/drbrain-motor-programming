<script lang="ts">
	/** Small "?" dialog listing every keyboard shortcut of the page; Esc or the button closes it. */
	let { keys, onclose }: { keys: [string, string][]; onclose: () => void } = $props();

	let closeBtn: HTMLButtonElement | undefined = $state();
	$effect(() => {
		closeBtn?.focus();
	});
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape') onclose();
	}}
/>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-indigo-950/40 p-4" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
	<div role="dialog" aria-modal="true" aria-labelledby="shortcuts-title" class="w-full max-w-sm rounded-2xl bg-white p-5 shadow-xl" data-testid="shortcuts-dialog">
		<div class="flex items-center justify-between">
			<h2 id="shortcuts-title" class="text-lg font-semibold text-indigo-950">Keyboard shortcuts</h2>
			<button bind:this={closeBtn} type="button" class="rounded-full px-2 text-lg leading-none text-slate-500 hover:bg-slate-100" aria-label="Close" onclick={onclose}>×</button>
		</div>
		<dl class="mt-3 grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1.5 text-sm">
			{#each keys as [k, d] (k)}
				<dt><kbd class="rounded border border-indigo-200 bg-indigo-50 px-1.5 font-mono text-xs">{k}</kbd></dt>
				<dd class="text-slate-700">{d}</dd>
			{/each}
		</dl>
	</div>
</div>
