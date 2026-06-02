<script lang="ts">
	import { onMount } from 'svelte';
	import type { Portfolio, Holding } from '$lib/types';
	import { listPortfolios, createPortfolio, deletePortfolio, getHoldings } from '$lib/api';

	let portfolios = $state<Portfolio[]>([]);
	let loading = $state(true);
	let error = $state('');
	let selected = $state<Portfolio | null>(null);
	let holdings = $state<Holding[]>([]);
	let holdingsLoading = $state(false);

	let newName = $state('');
	let newDesc = $state('');
	let showCreate = $state(false);

	onMount(() => loadPortfolios());

	async function loadPortfolios() {
		loading = true;
		error = '';
		try {
			portfolios = await listPortfolios();
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function loadHoldings(id: string) {
		holdingsLoading = true;
		try {
			holdings = await getHoldings(id);
		} catch (e: any) {
			holdings = [];
		} finally {
			holdingsLoading = false;
		}
	}

	function select(portfolio: Portfolio) {
		window.location.hash = portfolio.ID;
		selected = portfolio;
		loadHoldings(portfolio.ID);
	}

	function deselect() {
		window.location.hash = '';
		selected = null;
		holdings = [];
	}

	async function handleCreate() {
		if (!newName.trim()) return;
		try {
			await createPortfolio(newName.trim(), newDesc.trim());
			newName = '';
			newDesc = '';
			showCreate = false;
			await loadPortfolios();
		} catch (e: any) {
			error = e.message;
		}
	}

	async function handleDelete(id: string) {
		try {
			await deletePortfolio(id);
			portfolios = portfolios.filter(p => p.ID !== id);
			if (selected?.ID === id) deselect();
		} catch (e: any) {
			error = e.message;
		}
	}

	function formatMoney(v: number): string {
		return v.toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
	}

	function formatPct(v: number): string {
		return `${v >= 0 ? '+' : ''}${v.toFixed(2)}%`;
	}
</script>

<svelte:head>
	<title>adjudex — Portfolio</title>
</svelte:head>

<div class="flex items-center justify-between mb-6">
	<h1 class="text-2xl font-bold text-accent-glow">Portfolios</h1>
	<button class="btn btn-primary" onclick={() => showCreate = !showCreate}>
		{showCreate ? '✕ Cancel' : '+ New'}
	</button>
</div>

{#if showCreate}
	<div class="card mb-6">
		<input
			bind:value={newName}
			placeholder="Portfolio name"
			class="w-full bg-adjudex-900 border border-adjudex-600 rounded-lg px-3 py-2 mb-3 text-slate-200 placeholder-slate-500"
		/>
		<input
			bind:value={newDesc}
			placeholder="Description (optional)"
			class="w-full bg-adjudex-900 border border-adjudex-600 rounded-lg px-3 py-2 mb-3 text-slate-200 placeholder-slate-500"
		/>
		<button class="btn btn-primary" onclick={handleCreate}>Create Portfolio</button>
	</div>
{/if}

{#if error}
	<div class="card mb-6 border-red-500/30 bg-red-500/10">
		<p class="text-red-400">{error}</p>
		<button class="btn btn-primary mt-2 text-sm" onclick={loadPortfolios}>Retry</button>
	</div>
{/if}

{#if loading}
	<p class="text-slate-500">Loading portfolios...</p>
{:else if portfolios.length === 0}
	<div class="card text-center py-12">
		<p class="text-slate-500 text-lg mb-4">No portfolios yet</p>
		<p class="text-slate-600">Create one to start tracking your investments.</p>
	</div>
{:else}
	<div class="grid gap-4">
		{#each portfolios as p (p.ID)}
		<div
			role="button"
			tabindex="0"
			onclick={() => selected?.ID === p.ID ? deselect() : select(p)}
			onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { selected?.ID === p.ID ? deselect() : select(p); e.preventDefault(); } }}
			class="card text-left w-full transition-all duration-200 cursor-pointer"
			class:ring-2={selected?.ID === p.ID}
			class:ring-accent={selected?.ID === p.ID}
		>
				<div class="flex items-center justify-between">
					<div>
						<h2 class="text-lg font-semibold text-slate-100">{p.Name}</h2>
						{#if p.Description}
							<p class="text-sm text-slate-500 mt-1">{p.Description}</p>
						{/if}
						<p class="text-xs text-slate-600 mt-2">
							{p.Positions?.length ?? 0} positions · Created {new Date(p.CreatedAt).toLocaleDateString()}
						</p>
					</div>
					<button
						class="btn btn-danger text-xs"
						onclick:stopPropagation={() => handleDelete(p.ID)}
					>
						Delete
					</button>
				</div>

				{#if selected?.ID === p.ID}
					<div class="mt-4 pt-4 border-t border-adjudex-700">
						{#if holdingsLoading}
							<p class="text-slate-500 text-sm">Loading holdings...</p>
						{:else if holdings.length > 0}
							<table class="w-full text-sm">
								<thead>
									<tr class="text-slate-500 text-left border-b border-adjudex-700">
										<th class="pb-2 font-medium">Symbol</th>
										<th class="pb-2 font-medium text-right">Qty</th>
										<th class="pb-2 font-medium text-right">Entry</th>
										<th class="pb-2 font-medium text-right">Current</th>
										<th class="pb-2 font-medium text-right">PnL</th>
									</tr>
								</thead>
								<tbody>
									{#each holdings as h (h.Position.ID)}
										<tr class="border-b border-adjudex-700/50">
											<td class="py-2 font-mono text-accent-glow">{h.Position.Symbol}</td>
											<td class="py-2 text-right">{h.Position.Quantity}</td>
											<td class="py-2 text-right text-slate-400">{formatMoney(h.Position.EntryPrice)}</td>
											<td class="py-2 text-right">{formatMoney(h.CurrentPrice)}</td>
											<td class="py-2 text-right" class:text-gain={h.PnL >= 0} class:text-loss={h.PnL < 0}>
												{formatPct(h.PnLPercent)}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						{:else}
							<p class="text-slate-500 text-sm">No positions in this portfolio.</p>
						{/if}
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}
