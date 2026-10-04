<script lang="ts">
	import { DRAG_MIME, decodeDrag, encodeDrag } from '$lib/dnd';
	import { computeSlotMarks, marksFor, type SlotMarks } from '$lib/highlight';
	import { instructionMeta, tapeLabel } from '$lib/instructions';
	import { applyDrop, setSlot, tapeLengths } from '$lib/program';
	import { keyToAction, moveSelection } from '$lib/shortcuts';
	import type { Frame, Instruction, Program, Status, StepEvent } from '$lib/types';

	let {
		program,
		selected = null,
		disabled = false,
		callStack = [],
		lastEvent = null,
		status = 'READY',
		onchange = () => {}
	}: {
		program: Program;
		/** Palette instruction placed on slot click. */
		selected?: Instruction | null;
		/** Read-only: no edits, drag or shortcuts (highlights still show). */
		disabled?: boolean;
		callStack?: Frame[];
		lastEvent?: StepEvent | null;
		status?: Status;
		onchange?: (p: Program) => void;
	} = $props();

	let root: HTMLElement | undefined = $state();
	let dropTarget = $state<string | null>(null);

	const tapes = $derived([
		{ tape: -1, slots: program.main },
		...program.subs.map((slots, i) => ({ tape: i, slots }))
	]);
	const lengthsByTape = $derived(
		Object.fromEntries([-1, ...program.subs.keys()].map((t, i) => [t, tapeLengths(program)[i]]))
	);
	const marks = $derived(computeSlotMarks(callStack, lastEvent, status, lengthsByTape));

	function focusSlot(tape: number, slot: number) {
		root?.querySelector<HTMLElement>(`[data-slot="${tape}:${slot}"]`)?.focus();
	}

	function onKeyDown(e: KeyboardEvent, tape: number, slot: number) {
		if (e.ctrlKey || e.metaKey || e.altKey) return;
		const action = keyToAction(e.key, program.subs.length);
		if (!action) return;
		if (action.type === 'move') {
			e.preventDefault();
			const next = moveSelection(tapeLengths(program), { tape, slot }, action.dx, action.dy);
			focusSlot(next.tape, next.slot);
			return;
		}
		if (disabled) return;
		e.preventDefault();
		if (action.type === 'clear') {
			onchange(setSlot(program, tape, slot, 'EMPTY'));
			return;
		}
		onchange(setSlot(program, tape, slot, action.instruction));
		// Advance so a program can be typed in one go.
		focusSlot(tape, slot + 1);
	}

	function onClick(tape: number, slot: number) {
		if (disabled || selected === null) return;
		onchange(setSlot(program, tape, slot, selected));
	}

	function onDragStart(e: DragEvent, tape: number, slot: number) {
		if (disabled || !e.dataTransfer) {
			e.preventDefault();
			return;
		}
		e.dataTransfer.setData(DRAG_MIME, encodeDrag({ kind: 'slot', tape, slot }));
		e.dataTransfer.effectAllowed = 'copyMove';
	}

	function onDragOver(e: DragEvent, tape: number, slot: number) {
		if (disabled) return;
		e.preventDefault();
		dropTarget = `${tape}:${slot}`;
		if (e.dataTransfer) e.dataTransfer.dropEffect = e.altKey || e.ctrlKey || e.metaKey ? 'copy' : 'move';
	}

	function onDrop(e: DragEvent, tape: number, slot: number) {
		dropTarget = null;
		if (disabled) return;
		e.preventDefault();
		const source = decodeDrag(e.dataTransfer?.getData(DRAG_MIME) ?? '');
		if (!source) return;
		const copy = source.kind === 'slot' && (e.altKey || e.ctrlKey || e.metaKey || e.dataTransfer?.dropEffect === 'copy');
		onchange(applyDrop(program, source, { tape, slot }, copy));
	}

	function slotClasses(ins: Instruction, m: SlotMarks): string {
		const parts = [instructionMeta(ins).classes];
		if (m.active) parts.push('ring-4 ring-indigo-500 z-10');
		if (m.caller) parts.push('ring-2 ring-violet-600 ring-offset-2');
		if (m.executed) parts.push('inset-ring-4 inset-ring-yellow-400');
		return parts.join(' ');
	}

	function slotLabel(tape: number, slot: number, ins: Instruction, m: SlotMarks): string {
		const parts = [`${tapeLabel(tape)} slot ${slot + 1}: ${instructionMeta(ins).label}`];
		if (m.active) parts.push('next to run');
		if (m.caller) parts.push('waiting for subroutine to return');
		if (m.executed) parts.push('just executed');
		return parts.join(', ');
	}
</script>

<div bind:this={root} class="space-y-3" role="group" aria-label="Program tapes">
	{#each tapes as t (t.tape)}
		<div>
			<h3 class="text-xs font-semibold uppercase tracking-wide text-indigo-800 mb-1">
				{tapeLabel(t.tape)} <span class="font-normal text-slate-500">({t.slots.length} slots)</span>
			</h3>
			<div class="flex flex-wrap gap-1.5">
				{#each t.slots as ins, slot (slot)}
					{@const m = marksFor(marks, t.tape, slot)}
					<button
						type="button"
						data-slot="{t.tape}:{slot}"
						data-testid="slot-{t.tape}-{slot}"
						class="relative h-12 w-12 rounded-md border-2 text-base font-bold {slotClasses(ins, m)} {dropTarget === `${t.tape}:${slot}`
							? 'outline-2 outline-dashed outline-indigo-600'
							: ''} {disabled ? 'cursor-default' : 'cursor-pointer'}"
						aria-label={slotLabel(t.tape, slot, ins, m)}
						aria-disabled={disabled}
						draggable={!disabled && ins !== 'EMPTY'}
						onclick={() => onClick(t.tape, slot)}
						onkeydown={(e) => onKeyDown(e, t.tape, slot)}
						ondragstart={(e) => onDragStart(e, t.tape, slot)}
						ondragover={(e) => onDragOver(e, t.tape, slot)}
						ondragleave={() => (dropTarget = null)}
						ondrop={(e) => onDrop(e, t.tape, slot)}
						ondragend={() => (dropTarget = null)}
					>
						<span aria-hidden="true">{instructionMeta(ins).glyph}</span>
						<span class="absolute bottom-0 right-0.5 text-[9px] font-normal opacity-60" aria-hidden="true">{slot + 1}</span>
					</button>
				{/each}
			</div>
		</div>
	{/each}
</div>
