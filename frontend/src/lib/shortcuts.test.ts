import { describe, expect, it } from 'vitest';
import { keyToAction, moveSelection } from './shortcuts';

describe('keyToAction', () => {
	it('maps instruction keys case-insensitively', () => {
		expect(keyToAction('f', 1)).toEqual({ type: 'place', instruction: 'MOVE_FORWARD' });
		expect(keyToAction('L', 1)).toEqual({ type: 'place', instruction: 'TURN_LEFT' });
		expect(keyToAction('r', 1)).toEqual({ type: 'place', instruction: 'TURN_RIGHT' });
	});

	it('only offers calls for subs the map has', () => {
		expect(keyToAction('2', 2)).toEqual({ type: 'place', instruction: 'CALL_SUB_2' });
		expect(keyToAction('3', 2)).toBeNull();
		expect(keyToAction('1', 0)).toBeNull();
	});

	it('maps clear and arrow keys; ignores others', () => {
		expect(keyToAction('Backspace', 0)).toEqual({ type: 'clear' });
		expect(keyToAction('ArrowDown', 0)).toEqual({ type: 'move', dx: 0, dy: 1 });
		expect(keyToAction('Enter', 0)).toBeNull();
		expect(keyToAction('x', 0)).toBeNull();
	});
});

describe('moveSelection', () => {
	const lengths = [5, 3, 2]; // main, sub1, sub2

	it('moves horizontally and clamps at the tape ends', () => {
		expect(moveSelection(lengths, { tape: -1, slot: 4 }, 1, 0)).toEqual({ tape: -1, slot: 4 });
		expect(moveSelection(lengths, { tape: -1, slot: 0 }, -1, 0)).toEqual({ tape: -1, slot: 0 });
		expect(moveSelection(lengths, { tape: 0, slot: 1 }, 1, 0)).toEqual({ tape: 0, slot: 2 });
	});

	it('moves between tapes, clamping the slot to the destination length', () => {
		expect(moveSelection(lengths, { tape: -1, slot: 4 }, 0, 1)).toEqual({ tape: 0, slot: 2 });
		expect(moveSelection(lengths, { tape: 0, slot: 2 }, 0, 1)).toEqual({ tape: 1, slot: 1 });
		expect(moveSelection(lengths, { tape: 1, slot: 1 }, 0, 1)).toEqual({ tape: 1, slot: 1 });
		expect(moveSelection(lengths, { tape: -1, slot: 0 }, 0, -1)).toEqual({ tape: -1, slot: 0 });
	});
});
