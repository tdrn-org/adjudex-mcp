import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			pages: '../internal/web/static',
			assets: '../internal/web/static',
			fallback: 'index.html'
		})
	}
};

export default config;
