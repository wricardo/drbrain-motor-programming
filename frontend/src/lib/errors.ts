export interface DescribedError {
	/** GraphQL extensions.code when present (NOT_FOUND, SESSION_PLAYING, ...). */
	code: string | null;
	/** Short, user-facing explanation chosen by code. */
	title: string;
	/** Raw server/network message for details. */
	message: string;
}

const TITLES: Record<string, string> = {
	NOT_FOUND: 'Not found',
	INVALID_PROGRAM: 'Invalid program',
	SESSION_PLAYING: 'Session is running — pause it first',
	SESSION_TERMINAL: 'Run is over — reset to try again',
	INVALID_ARGUMENT: 'Invalid input',
	FORBIDDEN: 'Forbidden — check the admin key'
};

interface GqlErrorLike {
	message?: string;
	extensions?: { code?: unknown };
}

function fromList(list: GqlErrorLike[]): DescribedError | null {
	const first = list.find((e) => e && (e.message || e.extensions?.code));
	if (!first) return null;
	const code = typeof first.extensions?.code === 'string' ? first.extensions.code : null;
	const message = first.message ?? '';
	return { code, title: (code && TITLES[code]) || 'Request failed', message };
}

/**
 * Normalises urql CombinedErrors, graphql-ws error arrays and plain errors into one shape.
 * Codes come from GraphQL extensions.code.
 */
export function describeError(err: unknown): DescribedError {
	if (Array.isArray(err)) {
		const d = fromList(err as GqlErrorLike[]);
		if (d) return d;
	} else if (err && typeof err === 'object') {
		const e = err as { graphQLErrors?: GqlErrorLike[]; networkError?: { message?: string } | null; message?: string };
		if (e.graphQLErrors?.length) {
			const d = fromList(e.graphQLErrors);
			if (d) return d;
		}
		if (e.networkError) {
			return { code: null, title: 'Cannot reach the server', message: e.networkError.message ?? '' };
		}
		if (e.message) return { code: null, title: 'Request failed', message: e.message };
	}
	return { code: null, title: 'Request failed', message: typeof err === 'string' ? err : '' };
}

/** Human explanation of why a run was lost. */
export function lossReasonText(reason: string | null): string {
	switch (reason) {
		case 'STEP_LIMIT':
			return 'The robot ran out of steps before collecting every treat.';
		case 'CALL_DEPTH':
			return 'Subroutine calls nested too deep (call depth limit exceeded).';
		case 'PROGRAM_ENDED':
			return 'The program finished with treats still left on the board.';
		default:
			return 'The run ended without collecting every treat.';
	}
}
