<script lang="ts">
	import type { ConnectionState } from '$lib/types';

	let { state }: { state: ConnectionState } = $props();

	const text: Record<ConnectionState, string> = {
		connecting: 'Connecting…',
		live: 'Live',
		reconnecting: 'Reconnecting…',
		offline: 'Offline'
	};
	const dot: Record<ConnectionState, string> = {
		connecting: 'bg-amber-500',
		live: 'bg-emerald-500',
		reconnecting: 'bg-amber-500 animate-pulse',
		offline: 'bg-rose-500'
	};
</script>

<span
	class="inline-flex items-center gap-1.5 rounded-full border border-indigo-200 bg-white px-2.5 py-1 text-xs font-medium text-indigo-900"
	role="status"
	data-testid="connection"
>
	<span class="h-2 w-2 rounded-full {dot[state]}" aria-hidden="true"></span>
	{text[state]}
</span>
