import { readable, type Readable } from 'svelte/store';
import { describeError, type DescribedError } from '../errors';
import type { ConnectionState, Session, SessionUpdate } from '../types';

/**
 * Merges an incoming snapshot into the current one. Snapshots are full, so the
 * newest wins; anything with seq <= the last seen seq is stale or a duplicate
 * (broadcasts can be delivered out of order) and is ignored.
 */
export function mergeSession(current: Session | null, incoming: Session): Session {
	if (current && incoming.seq <= current.seq) return current;
	return incoming;
}

/** Transport used by the store; the real one wraps urql + graphql-ws. */
export interface SessionSource {
	/** Fetches the current snapshot; null when the session does not exist. */
	fetch(id: string): Promise<Session | null>;
	/** Starts a subscription; the returned function stops it. */
	subscribe(
		id: string,
		sink: { next(update: SessionUpdate): void; error(err: unknown): void; complete(): void }
	): () => void;
	/** Optional connection-state feed. Returns an unsubscribe function. */
	onConnection?(cb: (state: ConnectionState) => void): () => void;
}

export interface SessionState {
	session: Session | null;
	loading: boolean;
	error: DescribedError | null;
	connection: ConnectionState;
}

export interface SessionStore extends Readable<SessionState> {
	/** Applies a snapshot from a mutation result; same seq rules as subscription updates. */
	apply(session: Session): void;
	/** Re-fetches the snapshot (merged by seq). */
	refresh(): Promise<void>;
	/** Stops the subscription and timers. Safe to call more than once. */
	destroy(): void;
}

export interface SessionStoreOptions {
	/** Initial resubscribe delay; doubles after each consecutive failure up to maxRetryMs. */
	retryMs?: number;
	maxRetryMs?: number;
}

/**
 * Creates a store of one session combining the initial query with the live
 * subscription. If the subscription ends or errors it is re-established with
 * backoff and the snapshot re-fetched, so a reconnecting client always resyncs.
 * The store is live from creation until destroy().
 */
export function createSessionStore(
	source: SessionSource,
	id: string,
	options: SessionStoreOptions = {}
): SessionStore {
	const baseRetry = options.retryMs ?? 1000;
	const maxRetry = options.maxRetryMs ?? 10000;

	let state: SessionState = { session: null, loading: true, error: null, connection: 'connecting' };
	const listeners = new Set<(s: SessionState) => void>();
	let destroyed = false;
	let fatal = false;
	let unsub: (() => void) | null = null;
	let unsubConn: (() => void) | null = null;
	let retryTimer: ReturnType<typeof setTimeout> | null = null;
	let failures = 0;

	function set(patch: Partial<SessionState>) {
		state = { ...state, ...patch };
		for (const l of listeners) l(state);
	}

	function accept(session: Session) {
		const merged = mergeSession(state.session, session);
		set({ session: merged, loading: false, error: null });
	}

	function fail(err: unknown) {
		const d = describeError(err);
		if (d.code === 'NOT_FOUND') fatal = true;
		set({ error: d, loading: false });
	}

	async function refresh(): Promise<void> {
		try {
			const s = await source.fetch(id);
			if (destroyed) return;
			if (s) accept(s);
			else {
				fatal = true;
				set({ loading: false, error: describeError([{ message: 'Session not found', extensions: { code: 'NOT_FOUND' } }]) });
			}
		} catch (e) {
			if (!destroyed) fail(e);
		}
	}

	function scheduleResubscribe() {
		if (destroyed || fatal || retryTimer) return;
		const delay = Math.min(maxRetry, baseRetry * 2 ** failures);
		failures++;
		retryTimer = setTimeout(() => {
			retryTimer = null;
			if (destroyed || fatal) return;
			start();
			void refresh();
		}, delay);
	}

	function start() {
		unsub?.();
		unsub = source.subscribe(id, {
			next(update) {
				failures = 0;
				// A delivered update proves the socket is up even if no 'connected' event was seen
				// (the shared ws client may already have been open).
				set({ connection: 'live' });
				accept(update.session);
			},
			error(err) {
				if (destroyed) return;
				set({ connection: 'reconnecting' });
				fail(err);
				scheduleResubscribe();
			},
			complete() {
				if (destroyed) return;
				set({ connection: 'reconnecting' });
				scheduleResubscribe();
			}
		});
	}

	if (source.onConnection) unsubConn = source.onConnection((connection) => set({ connection }));
	start();
	void refresh();

	return {
		subscribe(run) {
			run(state);
			listeners.add(run);
			return () => listeners.delete(run);
		},
		apply: (session) => {
			if (!destroyed) accept(session);
		},
		refresh,
		destroy() {
			destroyed = true;
			if (retryTimer) clearTimeout(retryTimer);
			retryTimer = null;
			unsub?.();
			unsub = null;
			unsubConn?.();
			unsubConn = null;
			listeners.clear();
		}
	};
}

/** A store that never emits; handy for components rendered without a live session. */
export const emptySessionState: Readable<SessionState> = readable({
	session: null,
	loading: false,
	error: null,
	connection: 'offline'
});
