<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';
	import { setContextClient } from '@urql/svelte';
	import { makeClient } from '$lib/graphql';
	import { browser } from '$app/environment';
	import { page } from '$app/stores';
	import { GAME, SITE, TITLE } from '$lib/brand';

	let { children } = $props();

	const navItems = [
		{ href: '/', label: 'Play' },
		{ href: '/learn', label: 'How it works' },
		{ href: '/sessions', label: 'Sessions' },
		{ href: '/maps', label: 'Maps' },
		{ href: '/editor', label: 'Editor' }
	];

	function isActive(href: string, pathname: string): boolean {
		if (href === '/') return pathname === '/' || pathname.startsWith('/play/') || pathname.startsWith('/watch/');
		return pathname === href || pathname.startsWith(`${href}/`);
	}

	// ssr=false so this always runs in the browser; safe to set the client synchronously.
	if (browser) {
		setContextClient(makeClient());
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>{TITLE}</title>
</svelte:head>

<div class="min-h-screen flex flex-col">
	<a
		href="#main"
		class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:bg-white focus:px-3 focus:py-2 focus:rounded"
		>Skip to content</a
	>
	<header class="border-b border-indigo-200 bg-white px-4 sm:px-6 py-3 flex flex-wrap items-center justify-between gap-2">
		<a href="/" class="flex items-center gap-2 no-underline">
			<img src={favicon} alt="" class="h-7 w-7" />
			<span class="text-lg font-semibold tracking-wide text-indigo-900">{SITE}</span>
			<span class="hidden sm:inline border-l border-indigo-200 pl-2 text-sm text-indigo-600">{GAME}</span>
		</a>
		<nav class="flex items-center gap-1 text-sm" aria-label="Main">
			{#each navItems as item}
				{@const active = isActive(item.href, $page.url.pathname)}
				<a
					href={item.href}
					aria-current={active ? 'page' : undefined}
					class="px-3 py-1.5 rounded-full transition-colors {active
						? 'bg-indigo-600 text-white'
						: 'text-indigo-900 hover:bg-indigo-100'}">{item.label}</a
				>
			{/each}
		</nav>
	</header>

	<main id="main" class="flex-1">
		{@render children()}
	</main>

	<footer class="border-t border-indigo-200 bg-white px-6 py-4 text-xs text-slate-600 flex flex-wrap gap-4">
		<span>Program the robot to collect every treat.</span>
		<a href="/learn" class="underline">Rules</a>
		<a href="/llms.txt" target="_blank" rel="noreferrer" class="underline">/llms.txt</a>
		<a href="/graphql" target="_blank" rel="noreferrer" class="underline">GraphQL endpoint</a>
	</footer>
</div>
