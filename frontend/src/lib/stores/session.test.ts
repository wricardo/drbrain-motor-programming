import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createSessionStore, mergeSession, type SessionSource } from './session';
import type { Session, SessionUpdate } from '../types';

function snap(seq: number, steps = seq): Session {
	return { id: 's1', seq, vm: { steps } } as unknown as Session;
}

type Sink = Parameters<SessionSource['subscribe']>[1];

function fakeSource(initial: Session | null) {
	const sinks: Sink[] = [];
	let unsubscribed = 0;
	let fetches = 0;
	const source: SessionSource = {
		async fetch() {
			fetches++;
			return initial;
		},
		subscribe(_id, sink) {
			sinks.push(sink);
			return () => {
				unsubscribed++;
			};
		}
	};
	return { source, sinks, stats: () => ({ unsubscribed, fetches }) };
}

const update = (s: Session): SessionUpdate => ({ seq: s.seq, session: s, event: null });

describe('mergeSession', () => {
	it('keeps the newest snapshot and ignores seq <= last seen', () => {
		const a = snap(5);
		expect(mergeSession(null, a)).toBe(a);
		expect(mergeSession(a, snap(4))).toBe(a);
		expect(mergeSession(a, snap(5, 99))).toBe(a);
		const b = snap(6);
		expect(mergeSession(a, b)).toBe(b);
	});
});

describe('createSessionStore', () => {
	beforeEach(() => vi.useFakeTimers());
	afterEach(() => vi.useRealTimers());

	it('merges the query result with out-of-order subscription updates by seq', async () => {
		const { source, sinks } = fakeSource(snap(2));
		const store = createSessionStore(source, 's1');
		let seen = 0;
		store.subscribe((s) => (seen = s.session?.seq ?? 0));
		sinks[0].next(update(snap(5)));
		sinks[0].next(update(snap(4))); // late delivery of an older tick
		await vi.advanceTimersByTimeAsync(0); // initial fetch (seq 2) resolves after the update
		expect(seen).toBe(5);
		store.apply(snap(3)); // stale mutation result
		expect(seen).toBe(5);
		store.apply(snap(6));
		expect(seen).toBe(6);
		store.destroy();
	});

	it('reports live once an update arrives and clears loading', async () => {
		const { source, sinks } = fakeSource(null);
		const store = createSessionStore(source, 's1');
		let state: { loading: boolean; connection: string } | undefined;
		store.subscribe((s) => (state = s));
		sinks[0].next(update(snap(1)));
		expect(state).toMatchObject({ loading: false, connection: 'live' });
		store.destroy();
	});

	it('resubscribes with backoff after an error and re-fetches the snapshot', async () => {
		const { source, sinks, stats } = fakeSource(snap(1));
		const store = createSessionStore(source, 's1', { retryMs: 100 });
		let state: { connection: string } | undefined;
		store.subscribe((s) => (state = s));
		await vi.advanceTimersByTimeAsync(0);
		expect(sinks).toHaveLength(1);
		sinks[0].error(new Error('socket closed'));
		expect(state?.connection).toBe('reconnecting');
		await vi.advanceTimersByTimeAsync(99);
		expect(sinks).toHaveLength(1);
		await vi.advanceTimersByTimeAsync(1);
		expect(sinks).toHaveLength(2);
		expect(stats().fetches).toBe(2);
		expect(stats().unsubscribed).toBe(1); // old subscription torn down
		// a completion (server ended the stream) also resubscribes, with doubled delay
		sinks[1].complete();
		await vi.advanceTimersByTimeAsync(199);
		expect(sinks).toHaveLength(2);
		await vi.advanceTimersByTimeAsync(1);
		expect(sinks).toHaveLength(3);
		store.destroy();
	});

	it('does not retry when the session does not exist', async () => {
		const { source, sinks } = fakeSource(null);
		const store = createSessionStore(source, 's1', { retryMs: 10 });
		let state: { error: { code: string | null } | null; session: unknown } | undefined;
		store.subscribe((s) => (state = s));
		await vi.advanceTimersByTimeAsync(0);
		expect(state?.error?.code).toBe('NOT_FOUND');
		sinks[0].complete();
		await vi.advanceTimersByTimeAsync(1000);
		expect(sinks).toHaveLength(1);
		store.destroy();
	});

	it('stops everything on destroy', async () => {
		const { source, sinks, stats } = fakeSource(snap(1));
		const store = createSessionStore(source, 's1', { retryMs: 10 });
		sinks[0].error(new Error('x'));
		store.destroy();
		await vi.advanceTimersByTimeAsync(1000);
		expect(sinks).toHaveLength(1);
		expect(stats().unsubscribed).toBeGreaterThanOrEqual(1);
		sinks[0].next(update(snap(9))); // must not throw or resurrect state
	});
});
