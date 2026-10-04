import type { Facing, Position } from './types';

export type Tile = '.' | '#' | '*';

export const START_GLYPHS: Record<Facing, string> = { UP: '^', RIGHT: '>', DOWN: 'v', LEFT: '<' };
const FACING_BY_GLYPH: Record<string, Facing> = { '^': 'UP', '>': 'RIGHT', v: 'DOWN', '<': 'LEFT' };

export const MIN_SIDE = 3;
export const MAX_SIDE = 20;

export interface StartCell {
	x: number;
	y: number;
	facing: Facing;
}

export interface ParsedLayout {
	width: number;
	height: number;
	cells: Tile[][];
	/** First start glyph found, or null. */
	start: StartCell | null;
	startCount: number;
	treatCount: number;
}

/** Parses map layout strings (row 0 = top, y grows downward). Unknown glyphs read as empty. */
export function parseLayout(layout: string[]): ParsedLayout {
	const height = layout.length;
	const width = layout.reduce((w, row) => Math.max(w, row.length), 0);
	let start: StartCell | null = null;
	let startCount = 0;
	let treatCount = 0;
	const cells: Tile[][] = layout.map((row, y) =>
		Array.from({ length: width }, (_, x) => {
			const ch = row[x] ?? '.';
			if (ch === '#') return '#';
			if (ch === '*') {
				treatCount++;
				return '*';
			}
			const facing = FACING_BY_GLYPH[ch];
			if (facing) {
				startCount++;
				start ??= { x, y, facing };
			}
			return '.';
		})
	);
	return { width, height, cells, start, startCount, treatCount };
}

/** Serializes editor state back to layout strings. A start on a non-empty cell overrides it. */
export function formatLayout(cells: Tile[][], start: StartCell | null): string[] {
	return cells.map((row, y) =>
		row
			.map((tile, x) => (start && start.x === x && start.y === y ? START_GLYPHS[start.facing] : tile))
			.join('')
	);
}

/** Returns cells resized to width x height, keeping existing content and padding with empty. */
export function resizeCells(cells: Tile[][], width: number, height: number): Tile[][] {
	return Array.from({ length: height }, (_, y) =>
		Array.from({ length: width }, (_, x) => cells[y]?.[x] ?? '.')
	);
}

/** Creates an all-empty width x height grid. */
export function blankCells(width: number, height: number): Tile[][] {
	return resizeCells([], width, height);
}

/** Cheap client-side layout checks mirroring the server's rules, to give instant feedback. */
export function localLayoutIssues(p: ParsedLayout): string[] {
	const issues: string[] = [];
	if (p.width < MIN_SIDE || p.width > MAX_SIDE || p.height < MIN_SIDE || p.height > MAX_SIDE) {
		issues.push(`Grid must be ${MIN_SIDE}–${MAX_SIDE} cells per side.`);
	}
	if (p.startCount !== 1) issues.push('Place exactly one start cell.');
	if (p.treatCount < 1) issues.push('Place at least one treat.');
	return issues;
}

/** True if two positions are equal. */
export function samePos(a: Position, b: Position): boolean {
	return a.x === b.x && a.y === b.y;
}
