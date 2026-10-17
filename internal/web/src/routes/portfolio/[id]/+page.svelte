<script lang="ts">
  import type { Holding, Portfolio } from '$lib/types';

  let { data } = $props();

  function formatCurrency(value: number | undefined, currency?: string): string {
    if (value == null || isNaN(value)) return '—';
    return new Intl.NumberFormat('de-DE', {
      style: 'currency',
      currency: currency || 'EUR',
    }).format(value);
  }

  function formatPercent(value: number | undefined): string {
    if (value == null || isNaN(value)) return '—';
    const sign = value >= 0 ? '+' : '';
    return `${sign}${value.toFixed(2)}%`;
  }

  // The API returns the listing as a technical identity string
  // ("twelvedata:isin=US0378331005:ticker=AAPL"); show the most readable part.
  function shortListing(listing: string): string {
    const ticker = listing.match(/ticker=([^:]+)/);
    if (ticker) return ticker[1];
    const isin = listing.match(/isin=([^:]+)/);
    if (isin) return isin[1];
    return listing;
  }

  // Guard against undefined data during SPA initial render
  let portfolio = $derived(data?.portfolio as Portfolio | undefined);
  let holdings = $derived((data?.holdings as Holding[] | undefined) ?? []);

  // Market values are all expressed in the requested display currency, so the
  // sum is meaningful even for a portfolio holding several currencies.
  let totalMarketValue = $derived(
    holdings.reduce((sum, h) => sum + (h.market_value?.value ?? 0), 0)
  );
  let displayCurrency = $derived(
    holdings.length > 0 ? holdings[0].market_value?.currency : undefined
  );

  // PnL is only shown when every holding has one. A partial total would look
  // complete while silently omitting positions that could not be converted.
  let pnlComplete = $derived(holdings.length > 0 && holdings.every((h) => h.pnl != null));
  let totalPnL = $derived(
    pnlComplete ? holdings.reduce((sum, h) => sum + (h.pnl?.value ?? 0), 0) : undefined
  );
  let totalPnLPct = $derived(
    pnlComplete && totalMarketValue > 0 && totalMarketValue - (totalPnL ?? 0) !== 0
      ? ((totalPnL ?? 0) / (totalMarketValue - (totalPnL ?? 0))) * 100
      : undefined
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
          {formatCurrency(totalMarketValue, displayCurrency)}
        </p>
      </div>
      <div class="bg-surface rounded-xl border border-border p-5">
        <p class="text-xs text-slate-500 uppercase tracking-wider">Total P&amp;L</p>
        <p class="mt-1 text-2xl font-bold {(totalPnL ?? 0) >= 0 ? 'text-positive' : 'text-negative'}">
          {formatCurrency(totalPnL, displayCurrency)}
        </p>
        {#if !pnlComplete}
          <p class="mt-1 text-xs text-slate-500">exchange rate missing</p>
        {/if}
      </div>
      <div class="bg-surface rounded-xl border border-border p-5">
        <p class="text-xs text-slate-500 uppercase tracking-wider">P&amp;L %</p>
        <p class="mt-1 text-2xl font-bold {(totalPnLPct ?? 0) >= 0 ? 'text-positive' : 'text-negative'}">
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
                    <span class="font-semibold text-white">{shortListing(h.listing)}</span>
                    {#if h.notes}
                      <p class="text-xs text-slate-500 mt-0.5">{h.notes}</p>
                    {/if}
                  </td>
                  <td class="px-6 py-4 text-right text-slate-300">{h.quantity}</td>
                  <td class="px-6 py-4 text-right text-slate-400">{formatCurrency(h.entry_price, h.currency)}</td>
                  <td class="px-6 py-4 text-right text-white font-medium">
                    {formatCurrency(h.current_price?.price, h.current_price?.currency)}
                  </td>
                  <td class="px-6 py-4 text-right text-slate-300">
                    {formatCurrency(h.market_value?.value, h.market_value?.currency)}
                  </td>
                  <td class="px-6 py-4 text-right font-medium {(h.pnl?.value ?? 0) >= 0 ? 'text-positive' : 'text-negative'}">
                    {formatCurrency(h.pnl?.value, h.pnl?.currency)}
                  </td>
                  <td class="px-6 py-4 text-right font-medium {(h.pnl_percent ?? 0) >= 0 ? 'text-positive' : 'text-negative'}">
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
