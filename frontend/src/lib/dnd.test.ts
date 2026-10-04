import { describe, expect, it } from 'vitest';
import { decodeDrag, encodeDrag } from './dnd';

describe('drag payload', () => {
	it('round-trips palette and slot sources', () => {
		const a = { kind: 'palette', instruction: 'CALL_SUB_2' } as const;
		const b = { kind: 'slot', tape: -1, slot: 3 } as const;
		expect(decodeDrag(encodeDrag(a))).toEqual(a);
		expect(decodeDrag(encodeDrag(b))).toEqual(b);
	});

	it('rejects foreign or malformed payloads', () => {
		expect(decodeDrag('hello')).toBeNull();
		expect(decodeDrag('{"kind":"palette","instruction":"DROP_TABLE"}')).toBeNull();
		expect(decodeDrag('{"kind":"slot","tape":"a","slot":1}')).toBeNull();
	});
});
