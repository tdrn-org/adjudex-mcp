<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from '$lib/api';
  import type { PortfolioSummary, ServerInfo } from '$lib/types';

  let summaries = $state<PortfolioSummary[]>([]);
  let info = $state<ServerInfo | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  function formatCurrency(value: number, currency?: string): string {
    if (value == null || isNaN(value)) return '—';
    return new Intl.NumberFormat('de-DE', {
      style: 'currency',
      currency: currency || 'EUR',
      maximumFractionDigits: 0,
    }).format(value);
  }

  function formatPercent(value: number): string {
    if (value == null || isNaN(value)) return '—';
    const sign = value >= 0 ? '+' : '';
    return `${sign}${value.toFixed(2)}%`;
  }

  let totalValue = $derived(
    summaries.reduce((s, sm) => s + sm.market_value, 0)
  );
  let totalPnL = $derived(
    summaries.reduce((s, sm) => s + sm.pnl, 0)
  );

  onMount(async () => {
    try {
      [summaries, info] = await Promise.all([
        get<PortfolioSummary[]>('/portfolios/summaries'),
        get<ServerInfo>('/info'),
      ]);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load';
    } finally {
      loading = false;
    }
  });
</script>

<div class="space-y-8">
  <!-- Page Header -->
  <div>
    <h1 class="text-3xl font-bold text-white tracking-tight">Portfolios</h1>
    <p class="mt-1 text-slate-400">
      Your tracked portfolios and watchlists
      {#if info}
        <span class="text-slate-600 text-xs ml-2">v{info.version}</span>
      {/if}
    </p>
  </div>

  <!-- Loading State -->
  {#if loading}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each Array(3) as _}
        <div class="bg-surface rounded-xl border border-border p-6 animate-pulse">
          <div class="h-5 bg-surface-alt rounded w-2/3 mb-3"></div>
          <div class="h-4 bg-surface-alt rounded w-full mb-2"></div>
          <div class="h-4 bg-surface-alt rounded w-1/2"></div>
        </div>
      {/each}
    </div>

  <!-- Error State -->
  {:else if error}
    <div class="bg-negative/10 border border-negative/30 rounded-xl p-6 text-center">
      <p class="text-negative font-medium">Failed to load portfolios</p>
      <p class="text-slate-400 text-sm mt-1">{error}</p>
    </div>

  <!-- Empty State -->
  {:else if summaries.length === 0}
    <div class="bg-surface rounded-xl border border-border p-12 text-center">
      <span class="text-5xl">📊</span>
      <h2 class="mt-4 text-xl font-semibold text-white">No portfolios yet</h2>
      <p class="mt-2 text-slate-400 max-w-md mx-auto">
        Create your first portfolio to start tracking stocks and analyzing performance.
      </p>
    </div>

  <!-- Portfolio Grid with Summaries -->
  {:else}
    <!-- Global Total -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="bg-surface rounded-xl border border-border p-4">
        <p class="text-xs text-slate-500 uppercase tracking-wider">Total Value</p>
        <p class="mt-1 text-xl font-bold text-white">{formatCurrency(totalValue)}</p>
      </div>
      <div class="bg-surface rounded-xl border border-border p-4">
        <p class="text-xs text-slate-500 uppercase tracking-wider">Total P&amp;L</p>
        <p class="mt-1 text-xl font-bold {totalPnL >= 0 ? 'text-positive' : 'text-negative'}">
          {formatCurrency(totalPnL)}
        </p>
      </div>
      <div class="bg-surface rounded-xl border border-border p-4">
        <p class="text-xs text-slate-500 uppercase tracking-wider">Portfolios</p>
        <p class="mt-1 text-xl font-bold text-white">{summaries.length}</p>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each summaries as s}
        <a
          href="/portfolio/{s.id}"
          class="block bg-surface rounded-xl border border-border p-6
            hover:border-accent/50 hover:bg-surface-hover
            transition-all duration-200 no-underline group"
        >
          <div class="flex items-start justify-between">
            <h3 class="text-lg font-semibold text-white group-hover:text-accent-glow transition-colors">
              {s.name}
            </h3>
            <span class="text-xs text-slate-500 bg-surface-alt rounded-full px-2 py-1">
              {s.position_count} pos
            </span>
          </div>

          <!-- Summary Stats -->
          <div class="mt-4 grid grid-cols-2 gap-3">
            <div>
              <p class="text-xs text-slate-500">Value</p>
              <p class="text-lg font-semibold text-white">
                {formatCurrency(s.market_value, s.currency)}
              </p>
            </div>
            <div>
              <p class="text-xs text-slate-500">P&amp;L</p>
              <p class="text-lg font-semibold {s.pnl >= 0 ? 'text-positive' : 'text-negative'}">
                {formatPercent(s.pnl_percent)}
              </p>
            </div>
          </div>
        </a>
      {/each}
    </div>
  {/if}
</div>
