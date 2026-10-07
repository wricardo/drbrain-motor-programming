import { beforeEach, describe, expect, it } from 'vitest';
import {
	countByStatus,
	progressText,
	filterSessions,
	lastSessionId,
	recentStarted,
	rememberSession,
	sortSessions,
	statusGroup,
	statusLine,
	statusText
} from './sessions';
import type { SessionSummary } from './types';

function sess(id: string, over: Partial<SessionSummary> & { status?: string; steps?: number } = {}): SessionSummary {
	const { status = 'READY', steps = 0, ...rest } = over;
	return {
		id,
		displayName: '',
		mapId: 'm',
		playing: false,
		createdAt: '2026-01-01T00:00:00Z',
		lastActionAt: '2026-01-01T00:00:00Z',
		map: { name: 'M' },
		vm: { status: status as SessionSummary['vm']['status'], steps, treatsRemaining: [] },
		...rest
	};
}

describe('statusText / statusGroup', () => {
	it('terminal status beats playing; READY idle is ready; mid-run idle is paused', () => {
		expect(statusText(sess('a', { status: 'WON', playing: true }))).toBe('won');
		expect(statusText(sess('a', { status: 'LOST' }))).toBe('lost');
		expect(statusText(sess('a', { status: 'RUNNING', playing: true }))).toBe('running');
		expect(statusText(sess('a', { status: 'RUNNING' }))).toBe('paused');
		expect(statusText(sess('a', { status: 'READY' }))).toBe('ready');
	});
	it('groups everything unfinished as playing', () => {
		expect(statusGroup(sess('a', { status: 'READY' }))).toBe('playing');
		expect(statusGroup(sess('a', { status: 'RUNNING' }))).toBe('playing');
		expect(statusGroup(sess('a', { status: 'WON' }))).toBe('won');
	});
});

describe('filterSessions', () => {
	const list = [
		sess('aa11', { displayName: 'Alice', status: 'WON' }),
		sess('bb22', { displayName: 'bob', status: 'RUNNING', playing: true, map: { name: 'Ladder' } }),
		sess('cc33', { status: 'LOST' })
	];
	const base = { status: 'all' as const, text: '' };

	it('filters by status group and by name, id or map (case-insensitive), keeping order', () => {
		expect(filterSessions(list, base).map((s) => s.id)).toEqual(['aa11', 'bb22', 'cc33']);
		expect(filterSessions(list, { ...base, status: 'won' }).map((s) => s.id)).toEqual(['aa11']);
		expect(filterSessions(list, { ...base, status: 'playing' }).map((s) => s.id)).toEqual(['bb22']);
		expect(filterSessions(list, { ...base, text: ' ALI ' }).map((s) => s.id)).toEqual(['aa11']);
		expect(filterSessions(list, { ...base, text: 'anon' }).map((s) => s.id)).toEqual(['cc33']);
		expect(filterSessions(list, { ...base, text: 'CC3' }).map((s) => s.id)).toEqual(['cc33']);
		expect(filterSessions(list, { ...base, text: 'ladder' }).map((s) => s.id)).toEqual(['bb22']);
		expect(filterSessions(list, { status: 'lost', text: 'bob' })).toEqual([]);
	});
});

describe('sortSessions / recentStarted', () => {
	const list = [
		sess('a', { steps: 5, lastActionAt: '2026-01-02T00:00:00Z', createdAt: '2026-01-01T00:00:00Z' }),
		sess('b', { steps: 0, lastActionAt: '2026-01-04T00:00:00Z', createdAt: '2026-01-04T00:00:00Z' }),
		sess('c', { steps: 9, lastActionAt: '2026-01-03T00:00:00Z', createdAt: '2026-01-02T00:00:00Z' })
	];
	it('sorts by activity, creation or steps, descending', () => {
		expect(sortSessions(list, 'recent').map((s) => s.id)).toEqual(['b', 'c', 'a']);
		expect(sortSessions(list, 'created').map((s) => s.id)).toEqual(['b', 'c', 'a']);
		expect(sortSessions(list, 'steps').map((s) => s.id)).toEqual(['c', 'a', 'b']);
	});
	it('recentStarted hides never-started sessions and caps the list', () => {
		expect(recentStarted(list).map((s) => s.id)).toEqual(['c', 'a']);
		expect(recentStarted(list, 1).map((s) => s.id)).toEqual(['c']);
	});
});

describe('countByStatus', () => {
	it('counts every chip and the total', () => {
		expect(countByStatus([sess('1', { status: 'WON' }), sess('2', { status: 'WON' }), sess('3')])).toEqual({
			all: 3,
			playing: 1,
			won: 2,
			lost: 0
		});
	});
});

describe('statusLine', () => {
	const vm = (status: string, steps = 0) => ({ status: status as 'READY', steps, lossReason: null });
	it('describes each state, with the run speed while running', () => {
		expect(statusLine({ playing: false, vm: vm('READY') })).toBe('Ready');
		expect(statusLine({ playing: true, speedMs: 100, vm: vm('RUNNING') })).toBe('Running at 100 ms/step');
		expect(statusLine({ playing: false, vm: vm('RUNNING') })).toBe('Paused');
		expect(statusLine({ playing: false, vm: vm('WON', 46) })).toBe('Won in 46 steps');
		expect(statusLine({ playing: false, vm: { status: 'LOST', steps: 3, lossReason: 'STEP_LIMIT' } })).toMatch(/^Lost: .*steps/);
	});
});

describe('rememberSession', () => {
	beforeEach(() => localStorage.clear());
	it('round-trips the last created session id', () => {
		expect(lastSessionId()).toBeNull();
		rememberSession('abc');
		expect(lastSessionId()).toBe('abc');
	});
});

describe('progressText', () => {
	it('shows value / limit for steps and treats', () => {
		const s = sess('a', { steps: 8 });
		s.vm.treatsRemaining = [{ x: 1, y: 0 }];
		expect(progressText(s, { maxSteps: 70, treats: [{ x: 0, y: 0 }, { x: 1, y: 0 }] })).toBe('8 / 70 steps · 1 / 2 treats');
		expect(progressText(s)).toBe('8 steps · 1 treats left');
	});
});
