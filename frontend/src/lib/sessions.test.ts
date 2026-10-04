import { describe, expect, it } from 'vitest';
import { countByStatus, filterSessions, statusText } from './sessions';
import type { SessionSummary } from './types';

function sess(id: string, over: Partial<SessionSummary> & { status?: string } = {}): SessionSummary {
	const { status = 'READY', ...rest } = over;
	return {
		id,
		displayName: '',
		mapId: 'm',
		playing: false,
		createdAt: '2026-01-01T00:00:00Z',
		lastActionAt: '2026-01-01T00:00:00Z',
		map: { name: 'M' },
		vm: { status: status as SessionSummary['vm']['status'], steps: 0, treatsRemaining: [] },
		...rest
	};
}

describe('statusText', () => {
	it('terminal status beats playing; READY idle is ready; mid-run idle is paused', () => {
		expect(statusText(sess('a', { status: 'WON', playing: true }))).toBe('won');
		expect(statusText(sess('a', { status: 'LOST' }))).toBe('lost');
		expect(statusText(sess('a', { status: 'RUNNING', playing: true }))).toBe('running');
		expect(statusText(sess('a', { status: 'RUNNING' }))).toBe('paused');
		expect(statusText(sess('a', { status: 'READY' }))).toBe('ready');
	});
});

describe('filterSessions', () => {
	const list = [
		sess('1', { displayName: 'Alice', status: 'WON' }),
		sess('2', { displayName: 'bob', status: 'RUNNING', playing: true }),
		sess('3', { status: 'LOST' })
	];
	const base = { status: 'all' as const, text: '' };

	it('filters by status and name (case-insensitive, anonymous fallback), keeping order', () => {
		expect(filterSessions(list, base).map((s) => s.id)).toEqual(['1', '2', '3']);
		expect(filterSessions(list, { ...base, status: 'won' }).map((s) => s.id)).toEqual(['1']);
		expect(filterSessions(list, { ...base, text: ' ALI ' }).map((s) => s.id)).toEqual(['1']);
		expect(filterSessions(list, { ...base, text: 'anon' }).map((s) => s.id)).toEqual(['3']);
		expect(filterSessions(list, { status: 'lost', text: 'bob' })).toEqual([]);
	});
});

describe('countByStatus', () => {
	it('counts every status and the total', () => {
		expect(countByStatus([sess('1', { status: 'WON' }), sess('2', { status: 'WON' }), sess('3')])).toEqual({
			all: 3, running: 0, ready: 1, paused: 0, won: 2, lost: 0
		});
	});
});
