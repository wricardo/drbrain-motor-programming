import type { GameMap } from './types';

const RANK: Record<string, number> = { easy: 0, medium: 1, hard: 2 };
const collator = new Intl.Collator('en', { numeric: true, sensitivity: 'base' });

/**
 * The one map order used everywhere (home, maps page, dropdowns, "Next map"):
 * easy -> medium -> hard (unknown last), then by name with numeric compare so
 * the series number in the name ("Patterns 2" < "Patterns 10") keeps series order.
 * Does not mutate the input.
 */
export function sortMaps<T extends Pick<GameMap, 'difficulty' | 'name'>>(maps: T[]): T[] {
	const rank = (m: T) => RANK[m.difficulty] ?? 99;
	return [...maps].sort((a, b) => rank(a) - rank(b) || collator.compare(a.name, b.name));
}

/** Map after currentId in sortMaps order, or null when it is the last (or unknown). */
export function nextMap<T extends Pick<GameMap, 'id' | 'difficulty' | 'name'>>(maps: T[], currentId: string): T | null {
	const sorted = sortMaps(maps);
	const i = sorted.findIndex((m) => m.id === currentId);
	return i >= 0 && i + 1 < sorted.length ? sorted[i + 1] : null;
}

export type DifficultyFilter = 'all' | 'easy' | 'medium' | 'hard';
export const DIFFICULTY_FILTERS: DifficultyFilter[] = ['all', 'easy', 'medium', 'hard'];

/** Case-insensitive search on name/description plus a difficulty filter; keeps input order. */
export function filterMaps<T extends Pick<GameMap, 'difficulty' | 'name' | 'description'>>(
	maps: T[],
	f: { text: string; difficulty: DifficultyFilter }
): T[] {
	const needle = f.text.trim().toLowerCase();
	return maps.filter(
		(m) =>
			(f.difficulty === 'all' || m.difficulty === f.difficulty) &&
			(!needle || m.name.toLowerCase().includes(needle) || m.description.toLowerCase().includes(needle))
	);
}

/** Counts maps per difficulty filter (for chip badges). */
export function countByDifficulty(maps: Pick<GameMap, 'difficulty'>[]): Record<DifficultyFilter, number> {
	const out: Record<DifficultyFilter, number> = { all: maps.length, easy: 0, medium: 0, hard: 0 };
	for (const m of maps) if (m.difficulty in out && m.difficulty !== 'all') out[m.difficulty as DifficultyFilter]++;
	return out;
}

/** Splits "Patterns 3 - Pinwheel" into its series label and title; series is null when the name has none. */
export function splitMapName(name: string): { series: string | null; title: string } {
	const m = /^(.+?\s\d+)\s+[-–—]\s+(.+)$/.exec(name.trim());
	return m ? { series: m[1], title: m[2] } : { series: null, title: name };
}
