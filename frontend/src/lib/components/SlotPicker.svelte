<script lang="ts">
	import { onMount } from 'svelte';
	import { instructionForKey, instructionMeta, paletteInstructions } from '$lib/instructions';
	import { placePopover, type Rect } from '$lib/popover';
	import type { Instruction } from '$lib/types';

	let {
		anchor,
		title,
		current,
		subCount,
		onpick,
		onclose
	}: {
		/** Element the menu opens next to (the clicked slot). */
		anchor: HTMLElement;
		title: string;
		/** Instruction currently in the slot; Erase is offered when it is not EMPTY. */
		current: Instruction;
		subCount: number;
		onpick: (ins: Instruction) => void;
		onclose: () => void;
	} = $props();

	let menu: HTMLElement | undefined = $state();
	let pos = $state({ left: 0, top: 0 });
	let placed = $state(false);

	const items = $derived<Instruction[]>([...paletteInstructions(subCount), ...(current === 'EMPTY' ? [] : ['EMPTY' as Instruction])]);

	function place() {
		if (!menu) return;
		const r: Rect = anchor.getBoundingClientRect();
		const next = placePopover(r, menu.getBoundingClientRect(), { width: window.innerWidth, height: window.innerHeight });
		if (next.left !== pos.left || next.top !== pos.top) pos = next;
		placed = true;
	}

	function options(): HTMLElement[] {
		return [...(menu?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])];
	}

	onMount(() => {
		place();
		const opts = options();
		(opts[Math.max(0, items.indexOf(current))] ?? opts[0])?.focus();

		const onPointerDown = (e: PointerEvent) => {
			const t = e.target as Node;
			if (!menu?.contains(t) && !anchor.contains(t)) onclose();
		};
		// Follow the slot through scrolls, resizes and layout shifts (e.g. the
		// unsaved-changes note appearing above the tapes).
		let frame = requestAnimationFrame(function track() {
			place();
			frame = requestAnimationFrame(track);
		});
		document.addEventListener('pointerdown', onPointerDown, true);
		return () => {
			cancelAnimationFrame(frame);
			document.removeEventListener('pointerdown', onPointerDown, true);
		};
	});

	function onKeyDown(e: KeyboardEvent) {
		if (e.ctrlKey || e.metaKey || e.altKey) return;
		const opts = options();
		const i = opts.indexOf(document.activeElement as HTMLElement);
		switch (e.key) {
			case 'Escape':
			case 'Tab':
				e.preventDefault();
				onclose();
				return;
			case 'ArrowDown':
			case 'ArrowRight':
				e.preventDefault();
				opts[(i + 1) % opts.length]?.focus();
				return;
			case 'ArrowUp':
			case 'ArrowLeft':
				e.preventDefault();
				opts[(i - 1 + opts.length) % opts.length]?.focus();
				return;
			case 'Home':
				e.preventDefault();
				opts[0]?.focus();
				return;
			case 'End':
				e.preventDefault();
				opts.at(-1)?.focus();
				return;
			case 'Backspace':
			case 'Delete':
				e.preventDefault();
				onpick('EMPTY');
				return;
		}
		if (e.key.length !== 1) return;
		const ins = instructionForKey(e.key, subCount);
		if (ins) {
			e.preventDefault();
			onpick(ins);
		}
	}
</script>

<div
	bind:this={menu}
	role="menu"
	tabindex="-1"
	aria-label={title}
	data-testid="slot-picker"
	class="fixed z-50 w-52 rounded-xl border border-indigo-200 bg-white p-1.5 shadow-lg {placed ? '' : 'invisible'}"
	style="left: {pos.left}px; top: {pos.top}px"
	onkeydown={onKeyDown}
>
	<p class="px-2 pt-1 pb-1.5 text-[11px] font-semibold uppercase tracking-wide text-indigo-800">{title}</p>
	{#each items as ins (ins)}
		{@const meta = instructionMeta(ins)}
		<button
			type="button"
			role="menuitem"
			data-testid="pick-{ins}"
			class="flex w-full items-center gap-2.5 rounded-lg px-2 py-1.5 text-left text-sm text-indigo-950 hover:bg-indigo-50 focus:bg-indigo-100 focus:outline-none {ins === current
				? 'font-semibold'
				: ''} {ins === 'EMPTY' ? 'mt-1 border-t border-slate-100 pt-2' : ''}"
			onclick={() => onpick(ins)}
		>
			<span
				class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border-2 text-sm font-bold {meta.classes}"
				aria-hidden="true">{ins === 'EMPTY' ? '⌫' : meta.glyph}</span
			>
			<span class="flex-1">{ins === 'EMPTY' ? 'Erase' : meta.label}</span>
			{#if ins === current}<span class="text-indigo-600" aria-hidden="true">✓</span>{/if}
			<kbd class="rounded border border-slate-200 bg-slate-50 px-1.5 text-[10px] text-slate-500">{ins === 'EMPTY' ? '⌫' : meta.key}</kbd>
		</button>
	{/each}
</div>
