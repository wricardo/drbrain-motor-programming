<script lang="ts">
	import { parseLayout } from '$lib/layout';

	/** fit: scale to fit inside a square parent (width as a share of the parent) instead of capping at 200px. */
	let { layout, label = '', fit = false }: { layout: string[]; label?: string; fit?: boolean } = $props();

	const parsed = $derived(parseLayout(layout));
	const arrow = $derived(parsed.start ? { UP: '▲', RIGHT: '▶', DOWN: '▼', LEFT: '◀' }[parsed.start.facing] : '');
</script>

<div
	class="grid gap-px bg-indigo-200 p-px rounded-md overflow-hidden {fit ? '' : 'w-full max-w-[200px]'}"
	style="{fit ? `width: ${Math.min(1, parsed.width / Math.max(1, parsed.height)) * 100}%; ` : ''}aspect-ratio: {parsed.width} / {parsed.height}; grid-template-columns: repeat({parsed.width}, minmax(0, 1fr)); grid-template-rows: repeat({parsed.height}, minmax(0, 1fr))"
	role="img"
	aria-label={label || `Map preview ${parsed.width} by ${parsed.height}`}
>
	{#each parsed.cells as row, y}
		{#each row as tile, x}
			{@const isStart = parsed.start?.x === x && parsed.start?.y === y}
			<div
				class="flex items-center justify-center leading-none text-[8px] {tile === '#'
					? 'bg-slate-500'
					: tile === '*'
						? 'bg-amber-400'
						: isStart
							? 'bg-indigo-600 text-white'
							: 'bg-indigo-50'}"
			>
				{isStart ? arrow : ''}
			</div>
		{/each}
	{/each}
</div>
