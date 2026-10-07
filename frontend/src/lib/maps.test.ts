import { describe, expect, it } from 'vitest';
import { countByDifficulty, filterMaps, nextMap, sortMaps, splitMapName } from './maps';

describe('sortMaps', () => {
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
		expect(sortMaps(input).map((m) => m.name)).toEqual(['A2', 'C', 'M', 'A', 'B', 'Z']);
		expect(input).toEqual(copy);
	});

	it('keeps series order by number within a difficulty', () => {
		const input = [
			{ name: 'Patterns 10 - X', difficulty: 'hard' },
			{ name: 'Patterns 4 - Comb', difficulty: 'hard' },
			{ name: 'Patterns 1 - Serpentine', difficulty: 'hard' }
		];
		expect(sortMaps(input).map((m) => m.name)).toEqual(['Patterns 1 - Serpentine', 'Patterns 4 - Comb', 'Patterns 10 - X']);
	});
});

describe('nextMap', () => {
	const maps = [
		{ id: 'h', name: 'H', difficulty: 'hard' },
		{ id: 'e', name: 'E', difficulty: 'easy' },
		{ id: 'm', name: 'M', difficulty: 'medium' }
	];
	it('returns the following map in sort order, null at the end or when unknown', () => {
		expect(nextMap(maps, 'e')?.id).toBe('m');
		expect(nextMap(maps, 'm')?.id).toBe('h');
		expect(nextMap(maps, 'h')).toBeNull();
		expect(nextMap(maps, 'x')).toBeNull();
	});
});

describe('filterMaps / countByDifficulty', () => {
	const maps = [
		{ name: 'Straight Line', description: 'Walk', difficulty: 'easy' },
		{ name: 'Comb', description: 'Teeth of a comb', difficulty: 'medium' },
		{ name: 'Spiral', description: 'Recursion', difficulty: 'hard' }
	];
	it('matches name or description case-insensitively and by difficulty', () => {
		expect(filterMaps(maps, { text: '', difficulty: 'all' })).toHaveLength(3);
		expect(filterMaps(maps, { text: ' TEETH ', difficulty: 'all' }).map((m) => m.name)).toEqual(['Comb']);
		expect(filterMaps(maps, { text: 'comb', difficulty: 'hard' })).toEqual([]);
		expect(filterMaps(maps, { text: '', difficulty: 'easy' }).map((m) => m.name)).toEqual(['Straight Line']);
	});
	it('counts per difficulty', () => {
		expect(countByDifficulty(maps)).toEqual({ all: 3, easy: 1, medium: 1, hard: 1 });
	});
});

describe('splitMapName', () => {
	it('separates the series label from the title', () => {
		expect(splitMapName('Patterns 3 - Pinwheel')).toEqual({ series: 'Patterns 3', title: 'Pinwheel' });
		expect(splitMapName('Recursion 10 – Long')).toEqual({ series: 'Recursion 10', title: 'Long' });
		expect(splitMapName('My map')).toEqual({ series: null, title: 'My map' });
	});
});
