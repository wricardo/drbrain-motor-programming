import type { GameMap } from './types';

const RANK: Record<string, number> = { easy: 0, medium: 1, hard: 2 };

/** Orders maps easy -> hard, then by name (unknown difficulties last). Does not mutate the input. */
export function sortMapsByDifficulty<T extends Pick<GameMap, 'difficulty' | 'name'>>(maps: T[]): T[] {
	const rank = (m: T) => RANK[m.difficulty] ?? 99;
	return [...maps].sort((a, b) => rank(a) - rank(b) || a.name.localeCompare(b.name));
}
