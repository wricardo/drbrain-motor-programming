import { describe, expect, it } from 'vitest';
import { computeSlotMarks, marksFor } from './highlight';
import type { StepEvent } from './types';

const lengths = { [-1]: 4, 0: 3, 1: 3 };

describe('computeSlotMarks', () => {
	it('marks the top frame pc as active and suspended frames at the CALL (pc-1)', () => {
		// main waiting after its CALL at slot 1 (pc 2); sub1 waiting after its CALL at slot 0; sub2 running slot 2.
		const marks = computeSlotMarks(
			[
				{ tape: -1, pc: 2 },
				{ tape: 0, pc: 1 },
				{ tape: 1, pc: 2 }
			],
			null,
			'RUNNING',
			lengths
		);
		expect(marksFor(marks, -1, 1)).toMatchObject({ caller: true, active: false });
		expect(marksFor(marks, -1, 2).caller).toBe(false);
		expect(marksFor(marks, 0, 0)).toMatchObject({ caller: true });
		expect(marksFor(marks, 1, 2)).toMatchObject({ active: true, caller: false });
	});

	it('marks the last executed slot, which can coincide with a caller slot', () => {
		const ev = { tapeIndex: -1, slotIndex: 1 } as StepEvent;
		const marks = computeSlotMarks(
			[
				{ tape: -1, pc: 2 },
				{ tape: 0, pc: 0 }
			],
			ev,
			'RUNNING',
			lengths
		);
		expect(marksFor(marks, -1, 1)).toEqual({ active: false, caller: true, executed: true });
		expect(marksFor(marks, 0, 0).active).toBe(true);
	});

	it('shows no active slot once the run is over and ignores out-of-range pcs', () => {
		const over = computeSlotMarks([{ tape: -1, pc: 1 }], null, 'WON', lengths);
		expect(marksFor(over, -1, 1).active).toBe(false);
		const ended = computeSlotMarks([{ tape: -1, pc: 4 }], null, 'RUNNING', lengths);
		expect(ended.size).toBe(0);
	});

	it('before the first step the main tape pc is active', () => {
		const marks = computeSlotMarks([{ tape: -1, pc: 0 }], null, 'READY', lengths);
		expect(marksFor(marks, -1, 0).active).toBe(true);
	});
});
