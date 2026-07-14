<script lang="ts">
  import type { Holding, Portfolio } from '$lib/types';

  let { data } = $props();

  function formatCurrency(value: number, currency?: string): string {
    if (value == null || isNaN(value)) return '—';
    return new Intl.NumberFormat('de-DE', {
      style: 'currency',
      currency: currency || 'EUR',
    }).format(value);
  }

  function formatPercent(value: number): string {
    if (value == null || isNaN(value)) return '—';
    const sign = value >= 0 ? '+' : '';
    return `${sign}${value.toFixed(2)}%`;
  }

  // Guard against undefined data during SPA initial render
  let portfolio = $derived(data?.portfolio as Portfolio | undefined);
  let holdings = $derived((data?.holdings as Holding[] | undefined) ?? []);

  let totalMarketValue = $derived(
    holdings.length > 0 ? holdings.reduce((sum, h) => sum + (h.market_value ?? 0), 0) : 0
  );
  let totalPnL = $derived(
    holdings.length > 0 ? holdings.reduce((sum, h) => sum + (h.pnl ?? 0), 0) : 0
  );
  let totalPnLPct = $derived(
    totalMarketValue > 0 && (totalMarketValue - totalPnL) !== 0
      ? (totalPnL / (totalMarketValue - totalPnL)) * 100
      : 0
  );
</script>

<div class="space-y-8">
  <!-- Loading state -->
  {#if !portfolio}
    <div class="bg-surface rounded-xl border border-border p-12 text-center animate-pulse">
      <p class="text-slate-400">Loading portfolio...</p>
    </div>
  {:else}
    <!-- Back link + Header -->
    <div>
      <a href="/" class="text-sm text-slate-400 hover:text-accent-glow transition-colors no-underline">
        ← Back to Portfolios
      </a>
      <h1 class="text-3xl font-bold text-white tracking-tight mt-2">
        {portfolio.name}
      </h1>
      {#if portfolio.description}
        <p class="mt-1 text-slate-400">{portfolio.description}</p>
      {/if}
    </div>

    <!-- Summary Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="bg-surface rounded-xl border border-border p-5">
        <p class="text-xs text-slate-500 uppercase tracking-wider">Market Value</p>
        <p class="mt-1 text-2xl font-bold text-white">
          {formatCurrency(totalMarketValue)}
        </p>
      </div>
      <div class="bg-surface rounded-xl border border-border p-5">
        <p class="text-xs text-slate-500 uppercase tracking-wider">Total P&amp;L</p>
        <p class="mt-1 text-2xl font-bold {totalPnL >= 0 ? 'text-positive' : 'text-negative'}">
          {formatCurrency(totalPnL)}
        </p>
      </div>
      <div class="bg-surface rounded-xl border border-border p-5">
        <p class="text-xs text-slate-500 uppercase tracking-wider">P&amp;L %</p>
        <p class="mt-1 text-2xl font-bold {totalPnLPct >= 0 ? 'text-positive' : 'text-negative'}">
          {formatPercent(totalPnLPct)}
        </p>
      </div>
    </div>

    <!-- Holdings Table -->
    <div class="bg-surface rounded-xl border border-border overflow-hidden">
      <div class="px-6 py-4 border-b border-border">
        <h2 class="text-lg font-semibold text-white">
          Holdings ({holdings.length})
        </h2>
      </div>

      {#if holdings.length === 0}
        <div class="p-12 text-center text-slate-400">
          No positions in this portfolio yet.
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full">
            <thead>
              <tr class="text-left text-xs text-slate-500 uppercase tracking-wider border-b border-border">
                <th class="px-6 py-3 font-medium">Symbol</th>
                <th class="px-6 py-3 font-medium text-right">Quantity</th>
                <th class="px-6 py-3 font-medium text-right">Entry</th>
                <th class="px-6 py-3 font-medium text-right">Current</th>
                <th class="px-6 py-3 font-medium text-right">Market Value</th>
                <th class="px-6 py-3 font-medium text-right">P&amp;L</th>
                <th class="px-6 py-3 font-medium text-right">P&amp;L %</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each holdings as h}
                <tr class="hover:bg-surface-hover transition-colors">
                  <td class="px-6 py-4">
                    <span class="font-semibold text-white">{h.symbol}</span>
                    {#if h.notes}
                      <p class="text-xs text-slate-500 mt-0.5">{h.notes}</p>
                    {/if}
                  </td>
                  <td class="px-6 py-4 text-right text-slate-300">{h.quantity}</td>
                  <td class="px-6 py-4 text-right text-slate-400">{formatCurrency(h.entry_price, h.currency)}</td>
                  <td class="px-6 py-4 text-right text-white font-medium">{formatCurrency(h.current_price, h.currency)}</td>
                  <td class="px-6 py-4 text-right text-slate-300">{formatCurrency(h.market_value, h.currency)}</td>
                  <td class="px-6 py-4 text-right font-medium {h.pnl >= 0 ? 'text-positive' : 'text-negative'}">
                    {formatCurrency(h.pnl, h.currency)}
                  </td>
                  <td class="px-6 py-4 text-right font-medium {h.pnl_percent >= 0 ? 'text-positive' : 'text-negative'}">
                    {formatPercent(h.pnl_percent)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  {/if}
</div>
