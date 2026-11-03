import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// SvelteKit 3 no longer reads svelte.config.js — the project configuration is passed
// to the sveltekit() plugin, and options that lived under `kit.*` are now top-level
// plugin options.
export default defineConfig({
  plugins: [
    tailwindcss(),
    sveltekit({
      adapter: adapter({
        pages: './static',
        assets: './static',
        fallback: 'index.html',
        precompress: false,
        strict: true,
      }),
    }),
  ],
});
