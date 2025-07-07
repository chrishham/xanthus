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
		target: 'es2020',
		minify: 'terser',
		sourcemap: false,
		rollupOptions: {
			output: {
				manualChunks(id) {
					// Vendor libraries
					if (id.includes('sweetalert2')) return 'vendor-sweetalert';
					if (id.includes('@xterm') || id.includes('xterm')) return 'vendor-terminal';
					
					// Core services
					if (id.includes('src/lib/services/api.ts') || 
						id.includes('src/lib/stores/ui.ts') || 
						id.includes('src/lib/stores/auth.ts')) return 'core';
					
					// Feature-specific chunks
					if (id.includes('src/lib/stores/dns.ts') || 
						id.includes('src/lib/services/dns.ts')) return 'dns';
					
					if (id.includes('src/lib/stores/vps.ts') || 
						id.includes('src/lib/services/terminal.ts')) return 'vps';
					
					if (id.includes('src/lib/stores/applications.ts') || 
						id.includes('src/lib/services/autoRefresh.ts')) return 'applications';
					
					// Default vendor chunk for other node_modules
					if (id.includes('node_modules')) return 'vendor';
				}
			}
		}
	},
	optimizeDeps: {
		include: ['sweetalert2', '@xterm/xterm']
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});