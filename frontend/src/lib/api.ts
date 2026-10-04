import type { OperationResult } from '@urql/svelte';

/** Awaits an urql operation and returns its data, throwing the CombinedError on failure. */
export async function unwrap<T>(op: { toPromise(): Promise<OperationResult<T>> }): Promise<T> {
	const res = await op.toPromise();
	if (res.error) throw res.error;
	if (!res.data) throw new Error('Empty response from server');
	return res.data;
}
