import { defineConfig } from 'vite';

export default defineConfig({
  build: { outDir: '../internal/server/web/dist', emptyOutDir: true },
  server: { proxy: { '/api': 'http://127.0.0.1:8080', '/healthz': 'http://127.0.0.1:8080' } },
});
