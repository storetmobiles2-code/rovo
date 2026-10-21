/** Outcome of `POST /auth/refresh`. */
export type RefreshResult = 'ok' | 'expired' | 'unavailable';

export interface Refresher {
  /** Single-flight: concurrent callers share one in-flight refresh. */
  refresh(): Promise<RefreshResult>;
}

interface LockManagerLike {
  request<T>(name: string, cb: () => Promise<T>): Promise<T>;
}

export interface RefresherOptions {
  url: string;
  fetch: (req: Request) => Promise<Response>;
  headers: () => Record<string, string>;
  /** Cross-tab lock (Web Locks API); defaults to `navigator.locks` when available. */
  locks?: LockManagerLike | null;
}

export const REFRESH_LOCK = 'rovo-refresh';

export function createRefresher(opts: RefresherOptions): Refresher {
  let inflight: Promise<RefreshResult> | null = null;
  const locks =
    opts.locks !== undefined
      ? opts.locks
      : typeof navigator !== 'undefined' && 'locks' in navigator
        ? (navigator.locks as unknown as LockManagerLike)
        : null;

  async function run(): Promise<RefreshResult> {
    try {
      const res = await opts.fetch(
        new Request(opts.url, {
          method: 'POST',
          credentials: 'include',
          headers: { ...opts.headers(), 'Content-Type': 'application/json' },
        }),
      );
      // 409 REFRESH_RACE: another tab already rotated; the cookie jar holds the new token.
      if (res.ok || res.status === 409) return 'ok';
      if (res.status === 401 || res.status === 403) return 'expired';
      return 'unavailable';
    } catch {
      return 'unavailable';
    }
  }

  return {
    refresh() {
      if (!inflight) {
        const p = locks ? locks.request(REFRESH_LOCK, run) : run();
        inflight = p.finally(() => {
          inflight = null;
        });
      }
      return inflight;
    },
  };
}

/** Requests to these paths never trigger a refresh-and-retry. */
const NO_REFRESH = [/\/auth\/refresh$/, /\/auth\/otp\//, /\/auth\/admin\//, /\/auth\/logout$/];

export function isRefreshExempt(url: string): boolean {
  const path = url.split('?')[0] ?? url;
  return NO_REFRESH.some((re) => re.test(path));
}

export interface AuthFetchOptions {
  fetch: (req: Request) => Promise<Response>;
  refresher: Refresher;
  /** Called when the refresh token is rejected. The auth package decides what to do. */
  onSessionExpired?: () => void;
}

/**
 * Wraps fetch: on 401 it refreshes once (single-flight) and retries the original request.
 * If refresh says the session is gone, `onSessionExpired` fires and the 401 is returned.
 */
export function createAuthFetch(opts: AuthFetchOptions): (req: Request) => Promise<Response> {
  return async (request: Request) => {
    const retry = request.clone();
    const response = await opts.fetch(request);
    if (response.status !== 401 || isRefreshExempt(request.url)) return response;
    const result = await opts.refresher.refresh();
    if (result === 'ok') return opts.fetch(retry);
    if (result === 'expired') opts.onSessionExpired?.();
    return response;
  };
}
