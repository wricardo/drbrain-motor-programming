import type { Instruction } from './types';
import type { DragSource } from './program';

/** Custom MIME type carrying a DragSource as JSON. */
export const DRAG_MIME = 'application/x-drbrain-slot';

/** Serializes a drag source for dataTransfer. */
export function encodeDrag(source: DragSource): string {
	return JSON.stringify(source);
}

const INSTRUCTIONS: readonly string[] = [
	'EMPTY',
	'TURN_LEFT',
	'TURN_RIGHT',
	'MOVE_FORWARD',
	'CALL_SUB_1',
	'CALL_SUB_2',
	'CALL_SUB_3'
];

/** Parses dataTransfer text back into a DragSource; returns null for anything malformed. */
export function decodeDrag(raw: string): DragSource | null {
	try {
		const v = JSON.parse(raw) as Record<string, unknown>;
		if (v.kind === 'palette' && typeof v.instruction === 'string' && INSTRUCTIONS.includes(v.instruction)) {
			return { kind: 'palette', instruction: v.instruction as Instruction };
		}
		if (v.kind === 'slot' && Number.isInteger(v.tape) && Number.isInteger(v.slot)) {
			return { kind: 'slot', tape: v.tape as number, slot: v.slot as number };
		}
	} catch {
		/* not ours */
	}
	return null;
}
