<script lang="ts">
  import { page } from '$app/stores';
  import '../app.css';

  const tabs = [
    { href: '/', label: 'Portfolios' },
  ];

  let { children } = $props();
</script>

<div class="min-h-screen bg-slate-950">
  <!-- Header -->
  <header class="border-b border-border bg-surface/50 backdrop-blur-sm sticky top-0 z-50">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-16">
        <a href="/" class="flex items-center gap-3 no-underline">
          <span class="text-2xl">📊</span>
          <span class="text-xl font-bold text-white tracking-tight">adjudex</span>
        </a>

        <nav class="flex gap-1">
          {#each tabs as tab}
            <a
              href={tab.href}
              class="px-4 py-2 rounded-lg text-sm font-medium transition-colors duration-150
                {$page.url.pathname === tab.href
                  ? 'bg-accent/15 text-accent-glow'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-surface-alt'}"
            >
              {tab.label}
            </a>
          {/each}
        </nav>
      </div>
    </div>
  </header>

  <!-- Content -->
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    {@render children()}
  </main>

  <!-- Footer -->
  <footer class="border-t border-border mt-auto">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
      <p class="text-xs text-slate-500 text-center">
        adjudex · Agent juDy eXchange · <span class="text-accent-glow">♥</span>
      </p>
    </div>
  </footer>
</div>
