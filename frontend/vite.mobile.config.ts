import { defineConfig } from 'vite';
import { svelte, vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
	root: 'mobile',
	base: './',
	plugins: [svelte({ configFile: false, preprocess: vitePreprocess() })],
	build: { outDir: 'dist', emptyOutDir: true }
});
