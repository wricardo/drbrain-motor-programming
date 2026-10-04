import type { Client } from '@urql/svelte';
import { getWsClient } from '../graphql';
import { SESSION_QUERY, SESSION_UPDATED_SUBSCRIPTION } from '../queries';
import type { ConnectionState, Session, SessionUpdate } from '../types';
import type { SessionSource } from './session';

/** Real session transport: urql for the snapshot query, graphql-ws for the subscription. */
export function createSessionSource(client: Client): SessionSource {
	return {
		async fetch(id) {
			const res = await client.query<{ session: Session | null }>(SESSION_QUERY, { id }, { requestPolicy: 'network-only' }).toPromise();
			if (res.error) throw res.error;
			return res.data?.session ?? null;
		},
		subscribe(id, sink) {
			return getWsClient().subscribe<{ sessionUpdated: SessionUpdate }>(
				{ query: SESSION_UPDATED_SUBSCRIPTION, variables: { sessionID: id } },
				{
					next(result) {
						if (result.errors?.length) sink.error(result.errors);
						else if (result.data?.sessionUpdated) sink.next(result.data.sessionUpdated);
					},
					error: (err) => sink.error(err),
					complete: () => sink.complete()
				}
			);
		},
		onConnection(cb) {
			const ws = getWsClient();
			let everConnected = false;
			const offs = [
				ws.on('connecting', () => cb(everConnected ? 'reconnecting' : 'connecting')),
				ws.on('connected', () => {
					everConnected = true;
					cb('live');
				}),
				ws.on('closed', () => cb(everConnected ? 'reconnecting' : 'offline'))
			];
			return () => offs.forEach((off) => off());
		}
	};
}

export type { ConnectionState };
