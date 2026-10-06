import { describe, expect, it } from 'vitest';
import { buildGenericPrompt, buildSessionPrompt, layoutBlock } from './prompt';
import type { GameMap } from './types';

function map(over: Partial<GameMap> = {}): GameMap {
	return {
		id: 'tiny',
		name: 'Tiny',
		description: 'A tiny map.',
		difficulty: 'easy',
		width: 5,
		height: 3,
		layout: ['>...*', '.###.', '.....'],
		start: { x: 0, y: 0 },
		startFacing: 'RIGHT',
		rocks: [],
		treats: [{ x: 4, y: 0 }],
		mainTapeLength: 3,
		subTapeLengths: [2, 4],
		maxSteps: 50,
		maxCallDepth: 6,
		allowRecursion: false,
		...over
	};
}

describe('layoutBlock', () => {
	it('adds an x ruler and y prefixes aligned with the cells', () => {
		expect(layoutBlock(['>.*', '...'])).toBe(['    012', 'y0  >.*', 'y1  ...'].join('\n'));
	});
	it('keeps columns aligned when there are 10+ rows', () => {
		const rows = Array.from({ length: 11 }, () => '...');
		const lines = layoutBlock(rows).split('\n');
		expect(lines[0].indexOf('0')).toBe(lines[1].indexOf('.'));
		expect(lines[11].indexOf('.')).toBe(lines[1].indexOf('.'));
	});
});

describe('buildSessionPrompt', () => {
	const p = buildSessionPrompt('http://h:1', { id: 'abc123', map: map() });

	it('carries the session id, endpoints and map facts the agent needs', () => {
		expect(p).toContain('Session ID: abc123');
		expect(p).toContain('http://h:1/graphql');
		expect(p).toContain('http://h:1/llms.txt');
		expect(p).toContain('http://h:1/watch/abc123');
		expect(p).toContain('Robot starts at (0,0) facing right (x+1)');
		expect(p).toContain('Treats (1): (4,0)');
		expect(p).toContain('main = 3 slots; sub1 (subs[0]) = 2 slots; sub2 (subs[1]) = 4 slots');
	});

	it('builds program skeletons with exactly the map tape lengths', () => {
		expect(p).toContain('main: [EMPTY, EMPTY, EMPTY]');
		expect(p).toContain('subs: [[EMPTY, EMPTY], [EMPTY, EMPTY, EMPTY, EMPTY]]');
		expect(p).toContain('setProgram(sessionID: "abc123"');
	});

	it('only offers calls to subs the map has', () => {
		expect(p).toContain('CALL_SUB_1, CALL_SUB_2.');
		expect(p).not.toContain('CALL_SUB_3');
		const none = buildSessionPrompt('http://h', { id: 'x', map: map({ subTapeLengths: [] }) });
		expect(none).not.toContain('CALL_SUB_1');
		expect(none).toContain('subs: []');
	});

	it('states the recursion rule that matches the map', () => {
		expect(p).toContain('Recursion is NOT allowed');
		const rec = buildSessionPrompt('http://h', { id: 'x', map: map({ allowRecursion: true }) });
		expect(rec).toContain('Recursion is ALLOWED');
		expect(rec).not.toContain('Recursion is NOT allowed');
	});

	it('does not depend on session state (stable text)', () => {
		expect(buildSessionPrompt('http://h:1', { id: 'abc123', map: map() })).toBe(p);
	});
});

describe('buildGenericPrompt', () => {
	it('points the agent at llms.txt and the create-session flow', () => {
		const g = buildGenericPrompt('http://h:1');
		expect(g).toContain('http://h:1/llms.txt');
		expect(g).toContain('createSession');
		expect(g).toContain('http://h:1/watch/SESSION_ID');
	});
});
