<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from '$lib/api';
  import type { Portfolio, ServerInfo } from '$lib/types';

  let portfolios = $state<Portfolio[]>([]);
  let info = $state<ServerInfo | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      [portfolios, info] = await Promise.all([
        get<Portfolio[]>('/portfolios'),
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
  {:else if portfolios.length === 0}
    <div class="bg-surface rounded-xl border border-border p-12 text-center">
      <span class="text-5xl">📊</span>
      <h2 class="mt-4 text-xl font-semibold text-white">No portfolios yet</h2>
      <p class="mt-2 text-slate-400 max-w-md mx-auto">
        Create your first portfolio to start tracking stocks and analyzing performance.
      </p>
    </div>

  <!-- Portfolio Grid -->
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each portfolios as p}
        <a
          href="/portfolio/{p.id}"
          class="block bg-surface rounded-xl border border-border p-6
            hover:border-accent/50 hover:bg-surface-hover
            transition-all duration-200 no-underline group"
        >
          <div class="flex items-start justify-between">
            <h3 class="text-lg font-semibold text-white group-hover:text-accent-glow transition-colors">
              {p.name}
            </h3>
            <span class="text-xs text-slate-500 bg-surface-alt rounded-full px-2 py-1">
              {p.positions?.length ?? 0} pos
            </span>
          </div>
          {#if p.description}
            <p class="mt-2 text-sm text-slate-400 line-clamp-2">{p.description}</p>
          {/if}
          <div class="mt-4 flex items-center gap-2 text-xs text-slate-500">
            <span>Created {new Date(p.created_at).toLocaleDateString()}</span>
          </div>
        </a>
      {/each}
    </div>
  {/if}
</div>
