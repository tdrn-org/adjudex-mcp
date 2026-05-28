<script>
	import '../app.css';
	let { children } = $props();
</script>

<nav class="border-b border-adjudex-700 bg-adjudex-800/50 backdrop-blur">
	<div class="max-w-5xl mx-auto px-4 py-3 flex items-center justify-between">
		<a href="/" class="text-xl font-bold text-accent-glow">
			adjudex <span class="text-slate-500 text-sm font-normal">Agent juDy eXchange</span>
		</a>
	</div>
</nav>

<main class="max-w-5xl mx-auto px-4 py-8">
	{@render children()}
</main>
