<script>
	import '../app.css';
	import { page } from '$app/stores';
	let { children } = $props();

	const tabs = [
		{ href: '/', label: 'Portfolio' },
		{ href: '/quotes', label: 'Quotes' }
	];
</script>

<nav class="border-b border-adjudex-700 bg-adjudex-800/50 backdrop-blur">
	<div class="max-w-5xl mx-auto px-4 py-3 flex items-center justify-between">
		<a href="/" class="text-xl font-bold text-accent-glow">
			adjudex <span class="text-slate-500 text-sm font-normal">Agent juDy eXchange</span>
		</a>
		<div class="flex gap-1">
			{#each tabs as tab}
				<a href={tab.href}
					class="px-4 py-1.5 rounded-lg text-sm font-medium transition-colors"
					class:text-accent-glow={$page.url.pathname === tab.href}
					class:text-slate-400={$page.url.pathname !== tab.href}
					class:hover:text-slate-200={$page.url.pathname !== tab.href}
				>
					{tab.label}
				</a>
			{/each}
		</div>
	</div>
</nav>

<main class="max-w-5xl mx-auto px-4 py-8">
	{@render children()}
</main>
