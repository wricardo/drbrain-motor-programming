<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import { getAdminKey, setAdminKey } from '$lib/admin';
	import MiniGrid from '$lib/components/MiniGrid.svelte';
	import { describeError, type DescribedError } from '$lib/errors';
	import {
		MAX_SIDE,
		MIN_SIDE,
		START_GLYPHS,
		blankCells,
		formatLayout,
		localLayoutIssues,
		parseLayout,
		resizeCells,
		type StartCell,
		type Tile
	} from '$lib/layout';
	import {
		CREATE_MAP_MUTATION,
		DELETE_MAP_MUTATION,
		MAP_QUERY,
		UPDATE_MAP_MUTATION,
		VALIDATE_MAP_MUTATION
	} from '$lib/queries';
	import type { Facing, GameMap, MapInput, MapValidationResult } from '$lib/types';
	import Cookie from '$lib/components/Cookie.svelte';

	type Tool = 'empty' | 'rock' | 'treat' | 'start';

	const client = getContextClient();

	let id = $state('');
	let name = $state('');
	let description = $state('');
	let difficulty = $state('easy');
	let width = $state(10);
	let height = $state(10);
	let cells = $state<Tile[][]>(blankCells(10, 10));
	let start = $state<StartCell | null>({ x: 0, y: 9, facing: 'RIGHT' });
	let mainTapeLength = $state(10);
	let subTapeLengths = $state<number[]>([10]);
	let maxSteps = $state(200);
	let maxCallDepth = $state(16);
	let allowRecursion = $state(false);

	let tool = $state<Tool>('rock');
	let facing = $state<Facing>('RIGHT');
	let painting = $state(false);
	let existing = $state(false);
	let adminKey = $state(getAdminKey());
	let busy = $state(false);
	let error = $state<DescribedError | null>(null);
	let notice = $state('');
	let validation = $state<MapValidationResult | null>(null);

	const layout = $derived(formatLayout(cells, start));
	const parsed = $derived(parseLayout(layout));
	const localIssues = $derived(localLayoutIssues(parsed));
	const unreachable = $derived(new Set((validation?.unreachableTreats ?? []).map((p) => `${p.x},${p.y}`)));

	const tools: { id: Tool; label: string }[] = [
		{ id: 'empty', label: 'Empty (.)' },
		{ id: 'rock', label: 'Rock (#)' },
		{ id: 'treat', label: 'Treat (*)' },
		{ id: 'start', label: 'Start' }
	];
	const facings: Facing[] = ['UP', 'RIGHT', 'DOWN', 'LEFT'];
	const arrows: Record<Facing, string> = { UP: '▲', RIGHT: '▶', DOWN: '▼', LEFT: '◀' };

	onMount(async () => {
		const params = $page.url.searchParams;
		const mapId = params.get('map');
		if (!mapId) return;
		try {
			const data = await unwrap<{ map: GameMap | null }>(client.query(MAP_QUERY, { id: mapId }));
			if (!data.map) throw new Error(`Map "${mapId}" not found`);
			load(data.map, ['1', 'true', 'yes'].includes((params.get('duplicate') ?? '').toLowerCase()));
		} catch (e) {
			error = describeError(e);
		}
	});

	function load(m: GameMap, duplicate: boolean) {
		const p = parseLayout(m.layout);
		width = p.width;
		height = p.height;
		cells = p.cells;
		start = p.start;
		if (p.start) facing = p.start.facing;
		id = duplicate ? `${m.id}_copy` : m.id;
		name = duplicate ? `${m.name} (copy)` : m.name;
		description = m.description;
		difficulty = m.difficulty;
		mainTapeLength = m.mainTapeLength;
		subTapeLengths = [...m.subTapeLengths];
		maxSteps = m.maxSteps;
		maxCallDepth = m.maxCallDepth;
		allowRecursion = m.allowRecursion;
		existing = !duplicate;
		validation = null;
	}

	function invalidate() {
		validation = null;
		notice = '';
	}

	function paint(x: number, y: number) {
		invalidate();
		if (tool === 'start') {
			cells[y][x] = '.';
			start = { x, y, facing };
			return;
		}
		if (start && start.x === x && start.y === y) start = null;
		cells[y][x] = tool === 'rock' ? '#' : tool === 'treat' ? '*' : '.';
	}

	function setFacing(f: Facing) {
		facing = f;
		if (start) start = { ...start, facing: f };
		invalidate();
	}

	function resize(nextWidth: number, nextHeight: number) {
		const w = Math.max(MIN_SIDE, Math.min(MAX_SIDE, Math.floor(nextWidth) || MIN_SIDE));
		const h = Math.max(MIN_SIDE, Math.min(MAX_SIDE, Math.floor(nextHeight) || MIN_SIDE));
		width = w;
		height = h;
		cells = resizeCells(cells, w, h);
		if (start && (start.x >= w || start.y >= h)) start = null;
		invalidate();
	}

	function clearAll() {
		if (!confirm('Clear the whole grid?')) return;
		cells = blankCells(width, height);
		start = null;
		invalidate();
	}

	function setSubCount(n: number) {
		const next = subTapeLengths.slice(0, n);
		while (next.length < n) next.push(6);
		subTapeLengths = next;
		invalidate();
	}

	function toInput(): MapInput {
		return {
			id: id.trim(),
			name: name.trim(),
			description,
			difficulty,
			layout,
			mainTapeLength: Math.floor(mainTapeLength),
			subTapeLengths: subTapeLengths.map((n) => Math.floor(n)),
			maxSteps: Math.floor(maxSteps),
			maxCallDepth: Math.floor(maxCallDepth),
			allowRecursion
		};
	}

	async function run(fn: () => Promise<void>) {
		busy = true;
		error = null;
		notice = '';
		try {
			await fn();
		} catch (e) {
			error = describeError(e);
		} finally {
			busy = false;
		}
	}

	const validate = () =>
		run(async () => {
			const data = await unwrap<{ validateMap: MapValidationResult }>(
				client.mutation(VALIDATE_MAP_MUTATION, { map: toInput() })
			);
			validation = data.validateMap;
		});

	const save = () =>
		run(async () => {
			if (existing) await unwrap(client.mutation(UPDATE_MAP_MUTATION, { map: toInput() }));
			else await unwrap(client.mutation(CREATE_MAP_MUTATION, { map: toInput() }));
			existing = true;
			notice = `Saved map "${id.trim()}".`;
		});

	const remove = () =>
		run(async () => {
			if (!confirm(`Delete map "${id}"? Existing sessions keep their own copy.`)) return;
			await unwrap(client.mutation(DELETE_MAP_MUTATION, { id: id.trim() }));
			await goto('/maps');
		});

	const field = 'mt-1 w-full rounded-lg border border-indigo-300 bg-white px-3 py-2 text-sm';
	const btn = 'rounded-lg px-4 py-2 text-sm font-semibold disabled:opacity-50';
</script>

<svelte:head><title>{pageTitle('Map editor')}</title></svelte:head>
<svelte:window onpointerup={() => (painting = false)} />

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-6">
	<h1 class="text-2xl font-semibold text-indigo-950">Map editor</h1>
	<p class="mt-1 text-sm text-slate-700">
		Paint the board, set the start cell and tape sizes, validate, then save. Saving needs the admin key.
	</p>

	{#if error}
		<p role="alert" class="mt-3 rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900" data-testid="editor-error">
			<strong>{error.title}</strong>{#if error.code}<span class="ml-1 font-mono text-xs">[{error.code}]</span>{/if}
			{#if error.message}— {error.message}{/if}
		</p>
	{/if}
	{#if notice}
		<p role="status" class="mt-3 rounded-lg border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm text-emerald-900">{notice}</p>
	{/if}

	<div class="mt-5 grid gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)]">
		<section aria-label="Board painter" class="space-y-3">
			<div class="flex flex-wrap items-center gap-2" role="radiogroup" aria-label="Paint tool">
				{#each tools as t (t.id)}
					<button
						type="button"
						role="radio"
						aria-checked={tool === t.id}
						class="rounded-lg border-2 px-3 py-1.5 text-sm {tool === t.id ? 'border-indigo-600 bg-indigo-100 font-semibold' : 'border-indigo-200 bg-white'}"
						onclick={() => (tool = t.id)}>{t.label}</button
					>
				{/each}
				<span class="mx-1 text-slate-400" aria-hidden="true">|</span>
				<span class="text-xs text-slate-700" id="facing-label">Start facing</span>
				<div class="flex gap-1" role="radiogroup" aria-labelledby="facing-label">
					{#each facings as f (f)}
						<button
							type="button"
							role="radio"
							aria-checked={facing === f}
							aria-label={f.toLowerCase()}
							class="h-8 w-8 rounded-lg border-2 text-sm {facing === f ? 'border-indigo-600 bg-indigo-100' : 'border-indigo-200 bg-white'}"
							onclick={() => setFacing(f)}>{arrows[f]}</button
						>
					{/each}
				</div>
			</div>

			<div class="overflow-auto rounded-xl border border-indigo-200 bg-white p-2">
				<div
					class="inline-grid gap-px bg-indigo-200 p-px select-none touch-none"
					style="grid-template-columns: repeat({width}, 1.75rem)"
					role="grid"
					aria-label="Map cells, {width} by {height}"
				>
					{#each cells as row, y}
						{#each row as tile, x}
							{@const isStart = start?.x === x && start?.y === y}
							{@const bad = unreachable.has(`${x},${y}`)}
							<button
								type="button"
								class="flex h-7 w-7 items-center justify-center text-xs {tile === '#'
									? 'bg-slate-500'
									: isStart
										? 'bg-indigo-600 text-white'
										: 'bg-indigo-50 hover:bg-indigo-100'} {bad ? 'ring-2 ring-inset ring-rose-600' : ''}"
								aria-label="Column {x}, row {y}: {isStart ? `start facing ${start?.facing.toLowerCase()}` : tile === '#' ? 'rock' : tile === '*' ? 'treat' : 'empty'}{bad ? ', unreachable treat' : ''}"
								onpointerdown={() => {
									painting = true;
									paint(x, y);
								}}
								onpointerenter={() => painting && paint(x, y)}
								onkeydown={(e) => {
									if (e.key === 'Enter' || e.key === ' ') {
										e.preventDefault();
										paint(x, y);
									}
								}}
							>
								{#if isStart && start}
									<span aria-hidden="true">{arrows[start.facing]}</span>
								{:else if tile === '*'}
									<svg viewBox="-16 -16 32 32" class="h-5 w-5" aria-hidden="true"><Cookie cx={0} cy={0} /></svg>
								{/if}
							</button>
						{/each}
					{/each}
				</div>
			</div>

			<div class="flex flex-wrap items-end gap-3 text-sm">
				<label class="block">Width
					<input type="number" min={MIN_SIDE} max={MAX_SIDE} bind:value={width} onchange={(e) => resize(e.currentTarget.valueAsNumber, height)} class="mt-1 block w-20 rounded-lg border border-indigo-300 px-2 py-1" />
				</label>
				<label class="block">Height
					<input type="number" min={MIN_SIDE} max={MAX_SIDE} bind:value={height} onchange={(e) => resize(width, e.currentTarget.valueAsNumber)} class="mt-1 block w-20 rounded-lg border border-indigo-300 px-2 py-1" />
				</label>
				<button type="button" class="{btn} border border-indigo-300 text-indigo-900" onclick={clearAll}>Clear grid</button>
				<span class="text-xs text-slate-600">{parsed.treatCount} treats · glyphs: <code>. # * {Object.values(START_GLYPHS).join(' ')}</code>; y grows downward</span>
			</div>

			{#if localIssues.length}
				<ul class="text-xs text-amber-800 list-disc pl-5" data-testid="local-issues">
					{#each localIssues as issue}<li>{issue}</li>{/each}
				</ul>
			{/if}

			<div>
				<h2 class="text-sm font-semibold text-indigo-900">Preview</h2>
				<MiniGrid {layout} />
			</div>
		</section>

		<section aria-label="Map settings" class="space-y-3">
			<div class="grid grid-cols-2 gap-3">
				<label class="col-span-2 block text-sm">Map id <span class="text-xs text-slate-600">(a-z 0-9 _ -)</span>
					<input class={field} bind:value={id} oninput={invalidate} readonly={existing} placeholder="my_map" />
				</label>
				<label class="col-span-2 block text-sm">Name
					<input class={field} bind:value={name} oninput={invalidate} />
				</label>
				<label class="col-span-2 block text-sm">Description
					<textarea class={field} rows="2" bind:value={description} oninput={invalidate}></textarea>
				</label>
				<label class="block text-sm">Difficulty
					<select class={field} bind:value={difficulty} onchange={invalidate}>
						<option>easy</option><option>medium</option><option>hard</option>
					</select>
				</label>
				<label class="block text-sm">Main tape length (1–32)
					<input type="number" min="1" max="32" class={field} bind:value={mainTapeLength} oninput={invalidate} />
				</label>
				<label class="block text-sm">Subroutines (0–3)
					<select class={field} value={subTapeLengths.length} onchange={(e) => setSubCount(Number(e.currentTarget.value))}>
						{#each [0, 1, 2, 3] as n}<option value={n}>{n}</option>{/each}
					</select>
				</label>
				<div class="grid gap-2">
					{#each subTapeLengths as _, i}
						<label class="block text-sm">Sub {i + 1} length (1–16)
							<input type="number" min="1" max="16" class={field} bind:value={subTapeLengths[i]} oninput={invalidate} />
						</label>
					{/each}
				</div>
				<label class="block text-sm">Max steps (≤ 10000)
					<input type="number" min="1" max="10000" class={field} bind:value={maxSteps} oninput={invalidate} />
				</label>
				<label class="block text-sm">Max call depth (≤ 64)
					<input type="number" min="1" max="64" class={field} bind:value={maxCallDepth} oninput={invalidate} />
				</label>
				<label class="col-span-2 flex items-center gap-2 text-sm">
					<input type="checkbox" bind:checked={allowRecursion} onchange={invalidate} class="h-4 w-4 accent-indigo-600" />
					Allow recursion (a sub may call itself or a sub already on the stack)
				</label>
			</div>

			<div class="rounded-xl border border-indigo-200 bg-white p-3 space-y-3">
				<label class="block text-sm">Admin key
					<input
						type="password"
						class={field}
						bind:value={adminKey}
						oninput={(e) => setAdminKey(e.currentTarget.value)}
						autocomplete="off"
						placeholder="X-Admin-Key (kept for this tab only)"
					/>
				</label>
				<div class="flex flex-wrap gap-2">
					<button type="button" class="{btn} border border-indigo-300 text-indigo-900 hover:bg-indigo-50" disabled={busy} onclick={validate}>Validate</button>
					<button type="button" class="{btn} bg-indigo-600 text-white hover:bg-indigo-700" disabled={busy || localIssues.length > 0 || !id.trim() || !name.trim()} onclick={save}>
						{existing ? 'Update map' : 'Create map'}
					</button>
					{#if existing}
						<button type="button" class="{btn} border border-rose-300 text-rose-800 hover:bg-rose-50" disabled={busy} onclick={remove}>Delete</button>
					{/if}
				</div>
			</div>

			{#if validation}
				<div
					class="rounded-xl border p-3 text-sm {validation.valid ? 'border-emerald-300 bg-emerald-50 text-emerald-900' : 'border-rose-300 bg-rose-50 text-rose-900'}"
					role="status"
					data-testid="validation"
				>
					<strong>{validation.valid ? 'Map is valid.' : 'Map has problems:'}</strong>
					{#if validation.issues.length}
						<ul class="mt-1 list-disc pl-5">
							{#each validation.issues as issue}<li>{issue.message}</li>{/each}
						</ul>
					{/if}
					{#if validation.unreachableTreats.length}
						<p class="mt-1">Unreachable treats (ringed in red): {validation.unreachableTreats.map((p) => `(${p.x},${p.y})`).join(', ')}</p>
					{/if}
				</div>
			{/if}
		</section>
	</div>
</div>
