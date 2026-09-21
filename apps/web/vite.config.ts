import { defineConfig, Plugin } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import fs from 'fs';

// Custom plugin to reliably resolve local imports when workspace path contains '#'
function hashPathResolverPlugin(): Plugin {
  return {
    name: 'hash-path-resolver',
    enforce: 'pre',
    resolveId(source, importer) {
      // 1. Root-absolute paths (e.g. /src/main.tsx, /src/index.css)
      if (source.startsWith('/') && !source.startsWith('/@') && !source.startsWith('/node_modules')) {
        const cleanSource = source.split('?')[0];
        const candidateBase = path.join(__dirname, cleanSource);
        const candidates = [
          candidateBase,
          candidateBase + '.tsx',
          candidateBase + '.ts',
          candidateBase + '.jsx',
          candidateBase + '.js',
          path.join(candidateBase, 'index.tsx'),
          path.join(candidateBase, 'index.ts'),
        ];
        for (const c of candidates) {
          if (fs.existsSync(c)) {
            return c.replace(/\\/g, '/');
          }
        }
      }

      // 2. Relative paths (e.g. ./App, ../components/...)
      if (source.startsWith('.') && importer) {
        const cleanSource = source.split('?')[0];
        const dir = path.dirname(importer.split('?')[0]);
        const candidateBase = path.resolve(dir, cleanSource);
        const candidates = [
          candidateBase,
          candidateBase + '.tsx',
          candidateBase + '.ts',
          candidateBase + '.jsx',
          candidateBase + '.js',
          path.join(candidateBase, 'index.tsx'),
          path.join(candidateBase, 'index.ts'),
        ];
        for (const c of candidates) {
          if (fs.existsSync(c)) {
            return c.replace(/\\/g, '/');
          }
        }
      }
      return null;
    },
    load(id) {
      const cleanId = id.split('?')[0];
      const normalized = path.normalize(cleanId);
      if (fs.existsSync(normalized) && fs.statSync(normalized).isFile()) {
        return fs.readFileSync(normalized, 'utf-8');
      }
      return null;
    },
    configureServer(server) {
      server.middlewares.stack.unshift({
        route: '',
        handle: (req: any, res: any, next: any) => {
          if (req.url && (req.url.startsWith('/node_modules/') || req.url.includes('/node_modules/'))) {
            const rawPath = req.url.split('?')[0];
            const nodeModulesIdx = rawPath.indexOf('/node_modules/');
            const relPath = rawPath.slice(nodeModulesIdx);
            const filePath = path.join(__dirname, relPath);
            if (fs.existsSync(filePath) && fs.statSync(filePath).isFile()) {
              const ext = path.extname(filePath);
              const contentType =
                ext === '.js' || ext === '.mjs'
                  ? 'text/javascript'
                  : ext === '.css'
                  ? 'text/css'
                  : ext === '.json' || ext === '.map'
                  ? 'application/json'
                  : 'application/octet-stream';
              res.setHeader('Content-Type', contentType);
              res.setHeader('Cache-Control', 'no-cache');
              fs.createReadStream(filePath).pipe(res);
              return;
            }
          }
          next();
        },
      });
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
  preview: {
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
