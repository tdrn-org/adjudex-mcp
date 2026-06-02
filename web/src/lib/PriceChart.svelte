<script lang="ts">
	import type { Quote } from '$lib/types';

	interface Props {
		quotes: Quote[];
		indicator?: { label: string; values: number[]; color: string };
		height?: number;
	}

	let { quotes, indicator, height = 300 }: Props = $props();

	// Chart dimensions
	const pad = { top: 20, right: 16, bottom: 30, left: 60 };
	const w = $derived(800);
	const h = $derived(height);
	const iw = $derived(w - pad.left - pad.right);
	const ih = $derived(h - pad.top - pad.bottom);

	// Price scale
	const closes = $derived(quotes.map(q => q.Close));
	const minPrice = $derived(Math.min(...closes) * 0.995);
	const maxPrice = $derived(Math.max(...closes) * 1.005);
	const priceRange = $derived(maxPrice - minPrice || 1);

	function x(i: number): number {
		return pad.left + (quotes.length > 1 ? (i / (quotes.length - 1)) * iw : iw / 2);
	}

	function y(price: number): number {
		return pad.top + ih - ((price - minPrice) / priceRange) * ih;
	}

	// Build polyline points
	const pricePoints = $derived(
		quotes.map((q, i) => `${x(i)},${y(q.Close)}`).join(' ')
	);

	const indicatorPoints = $derived(
		indicator && indicator.values.length > 0
			? indicator.values.map((v, i) => {
					const idx = quotes.length - indicator.values.length + i;
					return idx >= 0 ? `${x(idx)},${y(v)}` : '';
			  }).filter(Boolean).join(' ')
			: ''
	);

	// Y-axis ticks
	const yTicks = $derived(() => {
		const ticks: number[] = [];
		const step = priceRange / 5;
		for (let i = 0; i <= 5; i++) ticks.push(minPrice + step * i);
		return ticks;
	});

	// X-axis labels (dates)
	const xLabels = $derived(() => {
		if (quotes.length <= 1) return [];
		const step = Math.max(1, Math.floor(quotes.length / 6));
		return quotes.filter((_, i) => i % step === 0 || i === quotes.length - 1);
	});

	function fmtPrice(v: number): string {
		return v >= 100 ? v.toFixed(1) : v.toFixed(2);
	}

	function fmtDate(ts: string): string {
		const d = new Date(ts);
		return `${d.getDate()}.${d.getMonth() + 1}.`;
	}
</script>

<svg viewBox="0 0 {w} {h}" class="w-full h-auto" style="max-height:{height}px">
	<!-- Grid lines -->
	{#each yTicks() as tick}
		<line x1={pad.left} y1={y(tick)} x2={w - pad.right} y2={y(tick)}
			class="stroke-slate-700" stroke-width="0.5" />
	{/each}

	<!-- Y-axis labels -->
	{#each yTicks() as tick}
		<text x={pad.left - 6} y={y(tick) + 4} text-anchor="end"
			class="fill-slate-500" font-size="10">{fmtPrice(tick)}</text>
	{/each}

	<!-- X-axis labels -->
	{#each xLabels() as q (q.Timestamp)}
		<text x={x(quotes.indexOf(q))} y={h - pad.bottom + 16} text-anchor="middle"
			class="fill-slate-500" font-size="10">{fmtDate(q.Timestamp)}</text>
	{/each}

	<!-- Price line -->
	<polyline points={pricePoints}
		class="fill-none stroke-accent" stroke-width="2"
		stroke-linejoin="round" stroke-linecap="round" />

	<!-- SMA/Indicator line -->
	{#if indicator && indicatorPoints}
		<polyline points={indicatorPoints}
			class="fill-none" stroke-width="1.5"
			stroke-linejoin="round" stroke-linecap="round"
			stroke-dasharray="4 2"
			style="stroke: {indicator.color}" />
	{/if}

	<!-- Legend -->
	<rect x={pad.left} y="6" width="12" height="3" rx="1" class="fill-accent" />
	<text x={pad.left + 16} y="11" class="fill-slate-300" font-size="9">Price</text>
	{#if indicator}
		<rect x={pad.left + 56} y="6" width="12" height="3" rx="1"
			style="fill: {indicator.color}" />
		<text x={pad.left + 72} y="11" class="fill-slate-300" font-size="9">{indicator.label}</text>
	{/if}
</svg>
