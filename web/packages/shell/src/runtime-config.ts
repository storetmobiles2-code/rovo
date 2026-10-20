/**
 * Runtime config injected by the deploy step as
 * `<script id="rovo-config" type="application/json">{…}</script>` (doc 17 §15.2) — "build
 * once, deploy many". Non-secret only. Missing keys fall back to dev defaults.
 */
export interface RovoRuntimeConfig {
  apiBase: string;
  env: 'local' | 'dev' | 'staging' | 'production';
  mediaBase: string;
  mapStyleUrl?: string;
  faroCollectorPath?: string;
  faroAppName?: string;
  umamiWebsiteId?: string;
  vapidPublicKey?: string;
  razorpayKeyId?: string;
  supportPhone?: string;
  cityDefault?: string;
}

const DEFAULTS: RovoRuntimeConfig = {
  apiBase: '/api/v1',
  env: 'local',
  mediaBase: '/media',
  cityDefault: 'mahabubnagar',
};

let cached: RovoRuntimeConfig | null = null;

export function readRuntimeConfig(): RovoRuntimeConfig {
  if (cached) return cached;
  let parsed: Partial<RovoRuntimeConfig> = {};
  try {
    const el = typeof document !== 'undefined' ? document.getElementById('rovo-config') : null;
    if (el?.textContent) parsed = JSON.parse(el.textContent) as Partial<RovoRuntimeConfig>;
  } catch {
    /* malformed or absent — use defaults (dev) */
  }
  cached = { ...DEFAULTS, ...parsed };
  return cached;
}

/** Test seam. */
export function __setRuntimeConfig(config: RovoRuntimeConfig | null): void {
  cached = config;
}
