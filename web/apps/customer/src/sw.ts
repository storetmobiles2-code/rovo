/// <reference lib="webworker" />
/**
 * Customer service worker (vite-plugin-pwa injectManifest). Doc 18 §3:
 * - precache the app shell; SPA navigations fall back to index.html (except /api, /media, /r)
 * - public catalog GETs: StaleWhileRevalidate (read-only offline browsing)
 * - media images: CacheFirst
 * - auth / payments / orders POST / stream / me: NEVER cached (enforced here and tested)
 */
import { cacheNames, clientsClaim } from 'workbox-core';
import { cleanupOutdatedCaches, precacheAndRoute } from 'workbox-precaching';
import { NavigationRoute, registerRoute } from 'workbox-routing';
import { CacheFirst, NetworkOnly, StaleWhileRevalidate } from 'workbox-strategies';
import { ExpirationPlugin } from 'workbox-expiration';
import { CacheableResponsePlugin } from 'workbox-cacheable-response';

declare const self: ServiceWorkerGlobalScope & { __WB_MANIFEST: Array<{ url: string; revision: string | null }> };

/** Paths the SW must never serve from cache (auth, payments, the stream, personal data). */
export const NEVER_CACHE = [/\/api\/v1\/auth\//, /\/api\/v1\/payments\//, /\/api\/v1\/stream/, /\/api\/v1\/me(\/|$)/];
const PUBLIC_CATALOG = /\/api\/v1\/public\//;
const MEDIA = /\/media\//;

cleanupOutdatedCaches();
precacheAndRoute(self.__WB_MANIFEST);

// SPA navigation fallback — but never rewrite API, media or the Go share page (/r/*).
registerRoute(
  new NavigationRoute(
    async ({ request }) => {
      const cached = await caches.match('/index.html', { cacheName: cacheNames.precache });
      return cached ?? fetch(request);
    },
    { denylist: [/^\/api\//, /^\/media\//, /^\/\.well-known\//, /^\/r\//] },
  ),
);

// Never-cache routes first, so they always win.
for (const pattern of NEVER_CACHE) registerRoute(pattern, new NetworkOnly(), 'GET');
registerRoute(({ request }) => request.method !== 'GET', new NetworkOnly());

registerRoute(
  PUBLIC_CATALOG,
  new StaleWhileRevalidate({
    cacheName: 'catalog',
    plugins: [
      new CacheableResponsePlugin({ statuses: [0, 200] }),
      new ExpirationPlugin({ maxEntries: 60, maxAgeSeconds: 24 * 60 * 60 }),
    ],
  }),
  'GET',
);

registerRoute(
  MEDIA,
  new CacheFirst({
    cacheName: 'images',
    plugins: [
      new CacheableResponsePlugin({ statuses: [0, 200] }),
      new ExpirationPlugin({ maxEntries: 200, maxAgeSeconds: 7 * 24 * 60 * 60, purgeOnQuotaError: true }),
    ],
  }),
  'GET',
);

self.addEventListener('message', (event) => {
  if ((event.data as { type?: string } | undefined)?.type === 'SKIP_WAITING') void self.skipWaiting();
});

clientsClaim();
