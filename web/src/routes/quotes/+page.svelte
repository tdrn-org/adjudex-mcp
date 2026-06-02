<script lang="ts">
	import type { Quote, PriceHistory } from '$lib/types';
	import { getQuote, getHistory } from '$lib/api';
	import PriceChart from '$lib/PriceChart.svelte';

	let symbol = $state('');
	let quote = $state<Quote | null>(null);
	let history = $state<PriceHistory | null>(null);
	let loading = $state(false);
	let error = $state('');

	// SMA computed client-side from history closes
	let smaValues = $state<number[] | null>(null);

	function computeSma(h: PriceHistory): number[] | null {
		if (h.Quotes.length < 20) return null;
		const period = 20;
		const closes = h.Quotes.map(q => q.Close);
		const result: number[] = [];
		let sum = closes.slice(0, period).reduce((a, b) => a + b, 0);
		result.push(sum / period);
		for (let i = period; i < closes.length; i++) {
			sum = sum - closes[i - period] + closes[i];
			result.push(sum / period);
		}
		const padded = new Array(period - 1).fill(null).concat(result);
		return padded.map(v => v ?? result[0]);
	}

	async function handleSearch(e: Event) {
		e.preventDefault();
		if (!symbol.trim()) return;
		loading = true;
		error = '';
		quote = null;
		history = null;
		smaValues = null;
		try {
			const sym = symbol.trim().toUpperCase();
			const [q, h] = await Promise.all([
				getQuote(sym),
				getHistory(sym)
			]);
			quote = q;
			history = h;
			smaValues = computeSma(h);
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	function fmtPrice(v: number): string {
		return v.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function fmtChange(current: number, prev: number): { text: string; cls: string } {
		const pct = ((current - prev) / prev) * 100;
		const sign = pct >= 0 ? '+' : '';
		return {
			text: `${sign}${pct.toFixed(2)}%`,
			cls: pct >= 0 ? 'text-gain' : 'text-loss'
		};
	}
</script>

<svelte:head>
	<title>adjudex — Quotes</title>
</svelte:head>

<div class="mb-6">
	<h1 class="text-2xl font-bold text-accent-glow mb-4">Quotes</h1>

	<form onsubmit={handleSearch} class="flex gap-3">
		<input
			bind:value={symbol}
			placeholder="Symbol (e.g. CRWV, NVDA, AAPL)"
			class="flex-1 bg-adjudex-900 border border-adjudex-600 rounded-lg px-4 py-2.5 text-slate-200 placeholder-slate-500 focus:outline-none focus:border-accent transition-colors"
		/>
		<button type="submit" class="btn btn-primary px-6" disabled={loading || !symbol.trim()}>
			{loading ? 'Loading...' : 'Search'}
		</button>
	</form>
</div>

{#if error}
	<div class="card mb-6 border-red-500/30 bg-red-500/10">
		<p class="text-red-400">{error}</p>
	</div>
{/if}

{#if quote}
	{@const prevClose = history && history.Quotes.length >= 2
		? history.Quotes[history.Quotes.length - 2].Close
		: quote.Close}
	{@const change = fmtChange(quote.Close, prevClose)}

	<div class="card mb-6">
		<div class="flex items-start justify-between">
			<div>
				<div class="flex items-center gap-3 mb-1">
					<h2 class="text-2xl font-bold text-slate-100">{quote.Symbol}</h2>
					<span class="text-xs px-2 py-0.5 rounded bg-adjudex-700 text-slate-400">{quote.Source}</span>
				</div>
				<div class="flex items-baseline gap-3">
					<span class="text-3xl font-mono font-bold text-accent-glow">{fmtPrice(quote.Close)}</span>
					<span class="text-lg font-mono {change.cls}">{change.text}</span>
				</div>
			</div>
			<div class="text-right text-sm text-slate-400 space-y-0.5">
				<div>Open <span class="text-slate-300 ml-2 font-mono">{fmtPrice(quote.Open)}</span></div>
				<div>High <span class="text-slate-300 ml-2 font-mono">{fmtPrice(quote.High)}</span></div>
				<div>Low  <span class="text-slate-300 ml-2 font-mono">{fmtPrice(quote.Low)}</span></div>
				<div>Vol  <span class="text-slate-300 ml-2 font-mono">{quote.Volume.toLocaleString()}</span></div>
			</div>
		</div>
	</div>

	<!-- Chart -->
	{#if history && history.Quotes.length > 0}
		<div class="card mb-6">
			<h3 class="text-sm font-semibold text-slate-400 mb-3">
				Price History — {history.Quotes.length} days
				{#if smaValues}
					<span class="ml-3 text-amber-400">SMA(20)</span>
				{/if}
			</h3>
			<PriceChart
				quotes={history.Quotes}
				indicator={smaValues ? { label: 'SMA(20)', values: smaValues, color: '#f59e0b' } : undefined}
			/>
		</div>
	{/if}

	<!-- Quick Stats -->
	{@const qs = history?.Quotes ?? []}
	{#if qs.length >= 20}
		{@const recent = qs.slice(-20)}
		{@const avg = recent.reduce((s, q) => s + q.Close, 0) / recent.length}
		{@const high20 = Math.max(...recent.map(q => q.High))}
		{@const low20 = Math.min(...recent.map(q => q.Low))}

		<div class="grid grid-cols-3 gap-4 mb-6">
			<div class="card text-center">
				<div class="text-xs text-slate-500 mb-1">20d Average</div>
				<div class="text-lg font-mono text-slate-200">{fmtPrice(avg)}</div>
			</div>
			<div class="card text-center">
				<div class="text-xs text-slate-500 mb-1">20d High</div>
				<div class="text-lg font-mono text-gain">{fmtPrice(high20)}</div>
			</div>
			<div class="card text-center">
				<div class="text-xs text-slate-500 mb-1">20d Low</div>
				<div class="text-lg font-mono text-loss">{fmtPrice(low20)}</div>
			</div>
		</div>
	{/if}
{:else if !loading}
	<div class="card text-center py-16">
		<p class="text-slate-500 text-lg mb-2">Search for a stock symbol</p>
		<p class="text-slate-600 text-sm">Try CRWV, NVDA, or AAPL to see price data and charts.</p>
	</div>
{/if}
