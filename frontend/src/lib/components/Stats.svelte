<script lang="ts">
	import type { Session } from '$lib/types';

	let { session }: { session: Session } = $props();

	const total = $derived(session.map.treats.length);
	const collected = $derived(total - session.vm.treatsRemaining.length);
	const stepPct = $derived(Math.min(100, (session.vm.steps / Math.max(1, session.map.maxSteps)) * 100));
</script>

<dl class="grid grid-cols-2 gap-2 text-sm" data-testid="stats">
	<div class="rounded-lg border border-indigo-200 bg-white p-2">
		<dt class="text-xs text-slate-600">Steps</dt>
		<dd class="font-semibold" data-testid="steps">{session.vm.steps} / {session.map.maxSteps}</dd>
		<div class="mt-1 h-1.5 rounded-full bg-indigo-100 overflow-hidden" aria-hidden="true">
			<div class="h-full {stepPct >= 80 ? 'bg-rose-500' : 'bg-indigo-500'}" style="width: {stepPct}%"></div>
		</div>
	</div>
	<div class="rounded-lg border border-indigo-200 bg-white p-2">
		<dt class="text-xs text-slate-600">Treats</dt>
		<dd class="font-semibold" data-testid="treats">{collected} / {total}</dd>
	</div>
</dl>
