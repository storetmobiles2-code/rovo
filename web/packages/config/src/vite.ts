import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { PluginOption, UserConfig } from 'vite';
import { VitePWA } from 'vite-plugin-pwa';
import { appThemeColor } from './tokens';

export type RovoAppName = 'customer' | 'restaurant' | 'rider' | 'admin';

export interface RovoPwaOptions {
  name: string;
  shortName: string;
  description: string;
  orientation: 'portrait' | 'any';
  /** Icon background (brand README: white for customer/rider, ink for restaurant). */
  backgroundColor: string;
}

export interface RovoAppOptions {
  app: RovoAppName;
  /** Absolute app directory (pass `import.meta.dirname`). */
  root: string;
  /** Dev server port: customer 5173, restaurant 5174, rider 5175, admin 5176. */
  port: number;
  /** PWA manifest + service worker; `false` for admin (no service worker). */
  pwa: RovoPwaOptions | false;
  /** Local API origin for the dev proxy. */
  apiTarget?: string;
}

/**
 * Shared Vite config for the four rovo apps.
 *
 * - `/api` is proxied to the local API **without** rewriting `Host`
 *   (`changeOrigin: false`), so the backend derives the audience from
 *   `localhost:517x` exactly as it derives it from `app.<domain>` in production.
 * - Build target ES2020 (doc 17 §12), hidden source maps (uploaded, never served).
 */
export function defineRovoApp(opts: RovoAppOptions): UserConfig {
  const pkg = JSON.parse(readFileSync(join(opts.root, 'package.json'), 'utf8')) as {
    version: string;
  };
  const apiTarget = opts.apiTarget ?? process.env.ROVO_API_URL ?? 'http://localhost:8080';
  const proxy = {
    '/api': { target: apiTarget, changeOrigin: false, ws: false, xfwd: true },
  };

  const plugins: PluginOption[] = [react(), tailwindcss()];
  if (opts.pwa) {
    plugins.push(
      VitePWA({
        strategies: 'injectManifest',
        srcDir: 'src',
        filename: 'sw.ts',
        registerType: 'prompt',
        injectRegister: false,
        includeAssets: ['favicon.ico', 'favicon-32.png', 'apple-touch-icon.png', 'logo-*.webp'],
        manifest: {
          id: `/?app=${opts.app}`,
          name: opts.pwa.name,
          short_name: opts.pwa.shortName,
          description: opts.pwa.description,
          start_url: '/?source=pwa',
          scope: '/',
          display: 'standalone',
          orientation: opts.pwa.orientation,
          lang: 'en-IN',
          dir: 'ltr',
          theme_color: appThemeColor,
          background_color: opts.pwa.backgroundColor,
          categories: ['food', opts.app === 'customer' ? 'shopping' : 'business'],
          prefer_related_applications: false,
          icons: [
            { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
            { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
            {
              src: '/icons/maskable-512.png',
              sizes: '512x512',
              type: 'image/png',
              purpose: 'maskable',
            },
          ],
        },
        injectManifest: {
          // App shell only. Fonts and media are runtime-cached by sw.ts.
          globPatterns: ['**/*.{js,css,html}', 'icons/icon-192.png', 'logo-*.webp', 'favicon-32.png'],
          globIgnores: ['**/*.map'],
          maximumFileSizeToCacheInBytes: 600 * 1024,
        },
        devOptions: { enabled: false },
      }),
    );
  }

  return {
    root: opts.root,
    plugins,
    define: {
      'import.meta.env.VITE_APP_NAME': JSON.stringify(opts.app),
      'import.meta.env.VITE_APP_VERSION': JSON.stringify(pkg.version),
    },
    server: { port: opts.port, strictPort: true, host: 'localhost', proxy },
    preview: { port: opts.port, strictPort: true, host: 'localhost', proxy },
    build: {
      target: 'es2020',
      sourcemap: 'hidden',
      assetsInlineLimit: 2048,
      chunkSizeWarningLimit: 400,
    },
  };
}
