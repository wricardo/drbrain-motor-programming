import { instructionForKey } from './instructions';
import type { Instruction } from './types';

export type KeyAction =
	| { type: 'place'; instruction: Instruction }
	| { type: 'clear' }
	| { type: 'move'; dx: number; dy: number };

/** Maps a keyboard key to a tape-editing action, or null when the key is not a shortcut. */
export function keyToAction(key: string, subCount: number): KeyAction | null {
	switch (key) {
		case 'Backspace':
		case 'Delete':
			return { type: 'clear' };
		case 'ArrowLeft':
			return { type: 'move', dx: -1, dy: 0 };
		case 'ArrowRight':
			return { type: 'move', dx: 1, dy: 0 };
		case 'ArrowUp':
			return { type: 'move', dx: 0, dy: -1 };
		case 'ArrowDown':
			return { type: 'move', dx: 0, dy: 1 };
	}
	if (key.length !== 1) return null;
	const ins = instructionForKey(key, subCount);
	return ins && ins !== 'EMPTY' ? { type: 'place', instruction: ins } : null;
}

/**
 * Moves a selection across tapes. lengths is ordered [main, sub1, ...]; tape
 * -1 is row 0. Movement clamps at the edges; vertical moves clamp the slot to the
 * destination tape's length.
 */
export function moveSelection(
	lengths: number[],
	cur: { tape: number; slot: number },
	dx: number,
	dy: number
): { tape: number; slot: number } {
	const row = Math.max(0, Math.min(lengths.length - 1, cur.tape + 1 + dy));
	const slot = Math.max(0, Math.min(lengths[row] - 1, cur.slot + dx));
	return { tape: row - 1, slot };
}
