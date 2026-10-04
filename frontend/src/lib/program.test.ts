import { describe, expect, it } from 'vitest';
import { applyDrop, emptyProgram, programsEqual, setSlot } from './program';

const cfg = { mainTapeLength: 4, subTapeLengths: [3, 2] };

describe('program editing', () => {
	it('builds an empty program shaped by the map tape config', () => {
		const p = emptyProgram(cfg);
		expect(p.main).toHaveLength(4);
		expect(p.subs.map((s) => s.length)).toEqual([3, 2]);
		expect([...p.main, ...p.subs.flat()].every((i) => i === 'EMPTY')).toBe(true);
	});

	it('setSlot is immutable and targets main (-1) or a sub', () => {
		const p = emptyProgram(cfg);
		const a = setSlot(p, -1, 2, 'MOVE_FORWARD');
		const b = setSlot(a, 1, 0, 'CALL_SUB_1');
		expect(p.main[2]).toBe('EMPTY');
		expect(a.main[2]).toBe('MOVE_FORWARD');
		expect(b.subs[1][0]).toBe('CALL_SUB_1');
		expect(programsEqual(a, b)).toBe(false);
	});

	it('setSlot ignores out-of-range targets', () => {
		const p = emptyProgram(cfg);
		expect(setSlot(p, -1, 9, 'TURN_LEFT')).toBe(p);
		expect(setSlot(p, 5, 0, 'TURN_LEFT')).toBe(p);
		expect(setSlot(p, -2, 0, 'TURN_LEFT')).toBe(p);
	});

	it('palette drops place the instruction', () => {
		const p = applyDrop(emptyProgram(cfg), { kind: 'palette', instruction: 'TURN_RIGHT' }, { tape: 0, slot: 1 }, false);
		expect(p.subs[0][1]).toBe('TURN_RIGHT');
	});

	it('slot drops swap by default so reordering never loses an instruction', () => {
		let p = setSlot(emptyProgram(cfg), -1, 0, 'MOVE_FORWARD');
		p = setSlot(p, -1, 1, 'TURN_LEFT');
		const out = applyDrop(p, { kind: 'slot', tape: -1, slot: 0 }, { tape: -1, slot: 1 }, false);
		expect(out.main.slice(0, 2)).toEqual(['TURN_LEFT', 'MOVE_FORWARD']);
	});

	it('slot drops with copy duplicate and keep the source', () => {
		const p = setSlot(emptyProgram(cfg), -1, 0, 'MOVE_FORWARD');
		const out = applyDrop(p, { kind: 'slot', tape: -1, slot: 0 }, { tape: 1, slot: 1 }, true);
		expect(out.main[0]).toBe('MOVE_FORWARD');
		expect(out.subs[1][1]).toBe('MOVE_FORWARD');
	});

	it('moving into an empty slot clears the source', () => {
		const p = setSlot(emptyProgram(cfg), -1, 0, 'MOVE_FORWARD');
		const out = applyDrop(p, { kind: 'slot', tape: -1, slot: 0 }, { tape: -1, slot: 3 }, false);
		expect(out.main).toEqual(['EMPTY', 'EMPTY', 'EMPTY', 'MOVE_FORWARD']);
	});

	it('dropping a slot on itself is a no-op', () => {
		const p = setSlot(emptyProgram(cfg), -1, 0, 'MOVE_FORWARD');
		expect(applyDrop(p, { kind: 'slot', tape: -1, slot: 0 }, { tape: -1, slot: 0 }, false)).toBe(p);
	});
});
