<script lang="ts">
	import { untrack } from 'svelte';
	import type { GameMap, StepEvent, VMState } from '$lib/types';
	import { nextAngle } from '$lib/grid';
	import Cookie from './Cookie.svelte';

	let {
		map,
		vm = null,
		lastEvent = null
	}: { map: GameMap; vm?: VMState | null; lastEvent?: StepEvent | null } = $props();

	const CELL = 40;

	const pos = $derived(vm?.pos ?? map.start);
	const facing = $derived(vm?.facing ?? map.startFacing);
	const treats = $derived(vm?.treatsRemaining ?? map.treats);
	const trail = $derived(vm?.visited ?? []);
	const blocked = $derived(!!vm && !!lastEvent && lastEvent.blocked && lastEvent.step === vm.steps);

	let angle = $state(0);
	$effect.pre(() => {
		const f = facing;
		angle = nextAngle(untrack(() => angle), f);
	});

	const cx = $derived(pos.x * CELL + CELL / 2);
	const cy = $derived(pos.y * CELL + CELL / 2);
	const label = $derived(
		`Board ${map.width} by ${map.height}. Robot at column ${pos.x}, row ${pos.y} facing ${facing.toLowerCase()}. ${treats.length} treats left.`
	);
</script>

<svg
	viewBox="0 0 {map.width * CELL} {map.height * CELL}"
	class="w-full h-auto rounded-xl border border-indigo-200 bg-indigo-50 shadow-sm"
	style="max-width: {Math.max(280, map.width * 56)}px"
	role="img"
	aria-label={label}
	data-testid="board"
>
	{#each { length: map.height } as _, y}
		{#each { length: map.width } as __, x}
			<rect
				x={x * CELL}
				y={y * CELL}
				width={CELL}
				height={CELL}
				fill={(x + y) % 2 === 0 ? '#eef2ff' : '#e0e7ff'}
			/>
		{/each}
	{/each}

	{#each trail as p, i (i)}
		<circle cx={p.x * CELL + CELL / 2} cy={p.y * CELL + CELL / 2} r="5" fill="#6366f1" opacity="0.35" />
	{/each}

	{#each map.rocks as r (`${r.x},${r.y}`)}
		<g aria-hidden="true">
			<rect x={r.x * CELL + 3} y={r.y * CELL + 3} width={CELL - 6} height={CELL - 6} rx="8" fill="#64748b" />
			<rect x={r.x * CELL + 7} y={r.y * CELL + 7} width={CELL - 22} height="8" rx="4" fill="#94a3b8" opacity="0.7" />
		</g>
	{/each}

	{#each treats as t (`${t.x},${t.y}`)}
		<Cookie cx={t.x * CELL + CELL / 2} cy={t.y * CELL + CELL / 2} />
	{/each}

	{#if blocked}
		<rect
			x={pos.x * CELL + 2}
			y={pos.y * CELL + 2}
			width={CELL - 4}
			height={CELL - 4}
			rx="8"
			fill="none"
			stroke="#e11d48"
			stroke-width="3"
		/>
	{/if}

	<g class="robot" style="transform: translate({cx}px, {cy}px) rotate({angle}deg)" aria-hidden="true">
		<polygon points="0,-15 12,12 0,6 -12,12" fill="#4338ca" stroke="#fff" stroke-width="2" stroke-linejoin="round" />
	</g>
</svg>

<style>
	.robot {
		transition: transform 120ms ease-out;
	}
	@media (prefers-reduced-motion: reduce) {
		.robot {
			transition: none;
		}
	}
</style>
