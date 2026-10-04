import { describe, expect, it } from 'vitest';
import { describeError } from './errors';

describe('describeError', () => {
	it('extracts extensions.code from urql CombinedError-like objects', () => {
		const d = describeError({
			graphQLErrors: [{ message: 'session is playing', extensions: { code: 'SESSION_PLAYING' } }]
		});
		expect(d.code).toBe('SESSION_PLAYING');
		expect(d.message).toBe('session is playing');
		expect(d.title).toMatch(/pause/i);
	});

	it('handles graphql-ws error arrays', () => {
		expect(describeError([{ message: 'nope', extensions: { code: 'NOT_FOUND' } }]).code).toBe('NOT_FOUND');
	});

	it('reports network errors without a code', () => {
		const d = describeError({ graphQLErrors: [], networkError: { message: 'Failed to fetch' } });
		expect(d).toMatchObject({ code: null, message: 'Failed to fetch' });
	});

	it('falls back for unknown codes and plain errors', () => {
		expect(describeError({ graphQLErrors: [{ message: 'x', extensions: { code: 'WEIRD' } }] }).title).toBe('Request failed');
		expect(describeError(new Error('boom')).message).toBe('boom');
	});
});
