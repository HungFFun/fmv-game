import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Proxy /api + /media → Go backend (cùng origin → cookie hoạt động, không cần CORS).
// Toàn bộ src là .js — cho @vitejs/plugin-react (babel) transform JSX trong .js
// (giống convention Tevi index.js). esbuildOptions xử lý JSX trong .js khi prebundle deps.
export default defineConfig({
  plugins: [react()],
  // Build production: plugin-react để esbuild lo JSX → bảo esbuild coi src/*.js là jsx.
  esbuild: { loader: 'jsx', include: /src\/.*\.js$/, exclude: [] },
  optimizeDeps: { esbuildOptions: { loader: { '.js': 'jsx' } } },
  resolve: {
    alias: {
      '@app': fileURLToPath(new URL('./src', import.meta.url)),
      '@models': fileURLToPath(new URL('./src/models', import.meta.url)),
      '@containers': fileURLToPath(new URL('./src/containers', import.meta.url)),
      '@components': fileURLToPath(new URL('./src/components', import.meta.url)),
      '@contexts': fileURLToPath(new URL('./src/contexts', import.meta.url)),
      '@providers': fileURLToPath(new URL('./src/providers', import.meta.url)),
      '@hooks': fileURLToPath(new URL('./src/hooks', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/media': 'http://localhost:8080',
    },
  },
});
