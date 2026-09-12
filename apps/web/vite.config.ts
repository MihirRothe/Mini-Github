import { defineConfig, Plugin } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import fs from 'fs';

// Custom plugin to reliably resolve local relative imports when workspace path contains '#'
function hashPathResolverPlugin(): Plugin {
  return {
    name: 'hash-path-resolver',
    enforce: 'pre',
    resolveId(source, importer) {
      if (source.startsWith('.') && importer) {
        const dir = path.dirname(importer);
        const candidateBase = path.resolve(dir, source);
        const candidates = [
          candidateBase + '.tsx',
          candidateBase + '.ts',
          candidateBase + '.jsx',
          candidateBase + '.js',
          path.join(candidateBase, 'index.tsx'),
          path.join(candidateBase, 'index.ts'),
        ];
        for (const c of candidates) {
          if (fs.existsSync(c)) {
            return c;
          }
        }
      }
      return null;
    },
  };
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [hashPathResolverPlugin(), react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/healthz': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/readyz': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
