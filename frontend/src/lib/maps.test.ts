import { describe, expect, it } from 'vitest';
import { sortMapsByDifficulty } from './maps';

describe('sortMapsByDifficulty', () => {
	it('orders easy -> medium -> hard, then by name, unknown last, without mutating', () => {
		const input = [
			{ name: 'B', difficulty: 'hard' },
			{ name: 'Z', difficulty: 'weird' },
			{ name: 'C', difficulty: 'easy' },
			{ name: 'A', difficulty: 'hard' },
			{ name: 'M', difficulty: 'medium' },
			{ name: 'A2', difficulty: 'easy' }
		];
		const copy = structuredClone(input);
		expect(sortMapsByDifficulty(input).map((m) => m.name)).toEqual(['A2', 'C', 'M', 'A', 'B', 'Z']);
		expect(input).toEqual(copy);
	});
});
