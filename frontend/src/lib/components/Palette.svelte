<script lang="ts">
	import { DRAG_MIME, encodeDrag } from '$lib/dnd';
	import { instructionMeta, paletteInstructions } from '$lib/instructions';
	import type { Instruction } from '$lib/types';

	let {
		selected = $bindable(null),
		subCount,
		disabled = false
	}: { selected?: Instruction | null; subCount: number; disabled?: boolean } = $props();

	const items = $derived<Instruction[]>([...paletteInstructions(subCount), 'EMPTY']);

	function toggle(ins: Instruction) {
		if (disabled) return;
		selected = selected === ins ? null : ins;
	}

	function onDragStart(e: DragEvent, ins: Instruction) {
		if (disabled || !e.dataTransfer) {
			e.preventDefault();
			return;
		}
		e.dataTransfer.setData(DRAG_MIME, encodeDrag({ kind: 'palette', instruction: ins }));
		e.dataTransfer.effectAllowed = 'copy';
	}
</script>

<div role="group" aria-label="Instruction palette" class="flex flex-wrap gap-2">
	{#each items as ins (ins)}
		{@const meta = instructionMeta(ins)}
		<button
			type="button"
			class="min-w-16 rounded-lg border-2 px-3 py-2 text-sm font-semibold transition-colors {meta.classes} {selected === ins
				? 'ring-4 ring-indigo-500'
				: ''} {disabled ? 'opacity-50 cursor-not-allowed' : 'hover:brightness-95 cursor-pointer'}"
			aria-pressed={selected === ins}
			aria-disabled={disabled}
			draggable={!disabled}
			ondragstart={(e) => onDragStart(e, ins)}
			onclick={() => toggle(ins)}
			title="{meta.label}{meta.key ? ` (${meta.key})` : ' (Backspace)'}"
		>
			<span class="block text-lg leading-none" aria-hidden="true">{ins === 'EMPTY' ? '⌫' : meta.glyph}</span>
			<span class="block text-[11px] mt-1">{ins === 'EMPTY' ? 'Erase' : meta.label}</span>
			{#if meta.key}<kbd class="block text-[10px] opacity-70">{meta.key}</kbd>{/if}
		</button>
	{/each}
</div>
