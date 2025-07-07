import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': 'http://localhost:8081',
			'/static': 'http://localhost:8081'
		}
	},
	build: {
		target: 'es2020'
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});