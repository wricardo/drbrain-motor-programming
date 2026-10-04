import type { Instruction } from './types';

export interface InstructionMeta {
	label: string;
	/** Short text glyph shown on slots and palette buttons. */
	glyph: string;
	/** Keyboard shortcut (case-insensitive) that places this instruction into the selected slot. */
	key: string;
	/** Tailwind classes for a filled slot / palette button. */
	classes: string;
}

const META: Record<Instruction, InstructionMeta> = {
	EMPTY: { label: 'Empty', glyph: '', key: '', classes: 'bg-white text-slate-400 border-slate-300' },
	TURN_LEFT: { label: 'Turn left', glyph: '↺', key: 'L', classes: 'bg-sky-100 text-sky-900 border-sky-400' },
	TURN_RIGHT: { label: 'Turn right', glyph: '↻', key: 'R', classes: 'bg-amber-100 text-amber-900 border-amber-400' },
	MOVE_FORWARD: { label: 'Move forward', glyph: '↑', key: 'F', classes: 'bg-emerald-100 text-emerald-900 border-emerald-400' },
	CALL_SUB_1: { label: 'Call sub 1', glyph: 'S1', key: '1', classes: 'bg-violet-100 text-violet-900 border-violet-400' },
	CALL_SUB_2: { label: 'Call sub 2', glyph: 'S2', key: '2', classes: 'bg-fuchsia-100 text-fuchsia-900 border-fuchsia-400' },
	CALL_SUB_3: { label: 'Call sub 3', glyph: 'S3', key: '3', classes: 'bg-rose-100 text-rose-900 border-rose-400' }
};

/** Returns display metadata for an instruction. */
export function instructionMeta(ins: Instruction): InstructionMeta {
	return META[ins];
}

const CALLS: Instruction[] = ['CALL_SUB_1', 'CALL_SUB_2', 'CALL_SUB_3'];

/** Returns the 0-based sub index an instruction calls, or null for non-call instructions. */
export function callTarget(ins: Instruction): number | null {
	const i = CALLS.indexOf(ins);
	return i < 0 ? null : i;
}

/** True when the instruction may appear in a program with subCount subroutines. */
export function isAllowed(ins: Instruction, subCount: number): boolean {
	const target = callTarget(ins);
	return target === null || target < subCount;
}

/** Instructions offered by the palette (excluding EMPTY) for a map with subCount subs. */
export function paletteInstructions(subCount: number): Instruction[] {
	const base: Instruction[] = ['MOVE_FORWARD', 'TURN_LEFT', 'TURN_RIGHT'];
	return [...base, ...CALLS.slice(0, Math.max(0, Math.min(subCount, CALLS.length)))];
}

/** Finds the instruction bound to a keyboard key, restricted to those allowed for subCount. */
export function instructionForKey(key: string, subCount: number): Instruction | null {
	const k = key.toUpperCase();
	for (const ins of Object.keys(META) as Instruction[]) {
		if (META[ins].key === k && isAllowed(ins, subCount)) return ins;
	}
	return null;
}

/** Human label for a tape index (-1 = main). */
export function tapeLabel(tape: number): string {
	return tape < 0 ? 'Main' : `Sub ${tape + 1}`;
}
