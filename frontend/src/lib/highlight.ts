import type { Frame, Status, StepEvent } from './types';

/** Visual marks for one tape slot. */
export interface SlotMarks {
	/** Next instruction to run (top frame). */
	active: boolean;
	/** The CALL instruction of a suspended caller frame. */
	caller: boolean;
	/** Slot executed by the most recent step. */
	executed: boolean;
}

/** Key used in the mark map: "<tape>:<slot>" with tape -1 for main. */
export function slotKey(tape: number, slot: number): string {
	return `${tape}:${slot}`;
}

const NO_MARKS: SlotMarks = { active: false, caller: false, executed: false };

/**
 * Computes slot highlights. The top frame's PC is the next slot to run; each
 * suspended frame is parked just after its CALL, so its marked slot is pc-1.
 * No "active" mark is produced once the run is over (WON/LOST).
 */
export function computeSlotMarks(
	callStack: Frame[],
	lastEvent: StepEvent | null,
	status: Status,
	tapeLengths: Record<number, number>
): Map<string, SlotMarks> {
	const marks = new Map<string, SlotMarks>();
	const touch = (tape: number, slot: number, patch: Partial<SlotMarks>) => {
		if (slot < 0 || slot >= (tapeLengths[tape] ?? 0)) return;
		const key = slotKey(tape, slot);
		marks.set(key, { ...(marks.get(key) ?? NO_MARKS), ...patch });
	};
	const over = status === 'WON' || status === 'LOST';
	callStack.forEach((frame, i) => {
		const isTop = i === callStack.length - 1;
		if (isTop) {
			if (!over) touch(frame.tape, frame.pc, { active: true });
		} else {
			touch(frame.tape, frame.pc - 1, { caller: true });
		}
	});
	if (lastEvent) touch(lastEvent.tapeIndex, lastEvent.slotIndex, { executed: true });
	return marks;
}

/** Looks up marks for a slot with a default for unmarked ones. */
export function marksFor(marks: Map<string, SlotMarks>, tape: number, slot: number): SlotMarks {
	return marks.get(slotKey(tape, slot)) ?? NO_MARKS;
}
