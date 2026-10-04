import type { Instruction, Program } from './types';

/** Tape lengths of a map (the subset of GameMap this module needs). */
export interface TapeConfig {
	mainTapeLength: number;
	subTapeLengths: number[];
}

/** Where a drag started: the palette, or an existing slot. */
export type DragSource =
	| { kind: 'palette'; instruction: Instruction }
	| { kind: 'slot'; tape: number; slot: number };

export interface SlotRef {
	tape: number;
	slot: number;
}

/** Builds an all-EMPTY program shaped for the map's tape config. */
export function emptyProgram(cfg: TapeConfig): Program {
	return {
		main: Array<Instruction>(cfg.mainTapeLength).fill('EMPTY'),
		subs: cfg.subTapeLengths.map((n) => Array<Instruction>(n).fill('EMPTY'))
	};
}

/** Deep-copies a program (also strips any reactive proxy wrappers). */
export function cloneProgram(p: Program): Program {
	return { main: [...p.main], subs: p.subs.map((s) => [...s]) };
}

/** Structural equality of two programs. */
export function programsEqual(a: Program, b: Program): boolean {
	if (a.main.length !== b.main.length || a.subs.length !== b.subs.length) return false;
	if (a.main.some((v, i) => v !== b.main[i])) return false;
	return a.subs.every((s, i) => s.length === b.subs[i].length && s.every((v, j) => v === b.subs[i][j]));
}

/** Returns the tape (-1 main, 0.. subs) or undefined when out of range. */
export function tapeOf(p: Program, tape: number): Instruction[] | undefined {
	return tape < 0 ? (tape === -1 ? p.main : undefined) : p.subs[tape];
}

/** Tape lengths ordered [main, sub1, sub2, ...]. */
export function tapeLengths(p: Program): number[] {
	return [p.main.length, ...p.subs.map((s) => s.length)];
}

/** Returns a new program with one slot replaced; out-of-range targets return the input unchanged. */
export function setSlot(p: Program, tape: number, slot: number, ins: Instruction): Program {
	const t = tapeOf(p, tape);
	if (!t || slot < 0 || slot >= t.length) return p;
	const next = cloneProgram(p);
	const nt = tapeOf(next, tape)!;
	nt[slot] = ins;
	return next;
}

/**
 * Applies a drag-and-drop. Palette drops place the instruction. Slot drops move
 * (swapping with the target so nothing is lost) or, when copy is true, duplicate.
 */
export function applyDrop(p: Program, source: DragSource, target: SlotRef, copy: boolean): Program {
	if (source.kind === 'palette') return setSlot(p, target.tape, target.slot, source.instruction);
	const from = tapeOf(p, source.tape)?.[source.slot];
	const to = tapeOf(p, target.tape)?.[target.slot];
	if (from === undefined || to === undefined) return p;
	if (source.tape === target.tape && source.slot === target.slot) return p;
	let next = setSlot(p, target.tape, target.slot, from);
	if (!copy) next = setSlot(next, source.tape, source.slot, to);
	return next;
}
