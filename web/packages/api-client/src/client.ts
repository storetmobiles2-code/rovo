import createClient, { type Client, type Middleware } from 'openapi-fetch';
import { ApiError } from './errors';
import { createAuthFetch, createRefresher, type Refresher } from './refresh';
import type { paths } from './schema';

export type AppId = 'customer' | 'restaurant' | 'rider' | 'admin';

export const API_BASE_URL = '/api/v1';

export interface ApiClientOptions {
  app: AppId;
  /** App semver (package.json), sent as `X-Rovo-Client: <app>@<version>`. */
  version: string;
  baseUrl?: string;
  /** Override fetch (tests, React Native). */
  fetch?: (req: Request) => Promise<Response>;
  /** Current UI language for `Accept-Language` (te → `te-IN, en;q=0.8`). */
  getLanguage?: () => string;
  /** Fired when the refresh token is rejected (session revoked/expired). */
  onSessionExpired?: () => void;
}

export type RovoClient = Client<paths>;

export interface ApiClient {
  /** Typed openapi-fetch client: `api.GET('/me')`, `api.POST('/auth/otp/request', { body })`. */
  api: RovoClient;
  /** Same auth-aware fetch for non-OpenAPI calls (e.g. CSV downloads). */
  fetch: (req: Request) => Promise<Response>;
  refresher: Refresher;
  app: AppId;
  version: string;
  baseUrl: string;
  /** Replaces the session-expired callback (the auth package installs its own). */
  setOnSessionExpired(cb: (() => void) | undefined): void;
  /** Reports an expired session (e.g. from the SSE stream) to the installed callback. */
  expireSession(): void;
  /** Standard headers every request carries. */
  headers(): Record<string, string>;
}

export function acceptLanguage(lang: string | undefined): string {
  return lang === 'te' ? 'te-IN, te;q=0.9, en;q=0.8' : 'en-IN, en;q=0.9';
}

/** Resolves `/api/v1` against the page origin (Request() needs absolute URLs outside browsers). */
function absolute(url: string): string {
  if (/^https?:/.test(url) || typeof location === 'undefined' || !location.origin) return url;
  return new URL(url, location.origin).toString().replace(/\/$/, '');
}

export function createApiClient(opts: ApiClientOptions): ApiClient {
  const baseUrl = absolute(opts.baseUrl ?? API_BASE_URL);
  const rawFetch = opts.fetch ?? ((req: Request) => globalThis.fetch(req));
  let onExpired = opts.onSessionExpired;

  const headers = (): Record<string, string> => ({
    // CSRF layer (doc 12 AUTH-D06) + telemetry/min-version gate (doc 11 §1).
    'X-Rovo-Client': `${opts.app}@${opts.version}`,
    'X-Rovo-App-Version': `${opts.app}/${opts.version}`,
    'Accept-Language': acceptLanguage(opts.getLanguage?.()),
  });

  const refresher = createRefresher({ url: `${baseUrl}/auth/refresh`, fetch: rawFetch, headers });
  const authFetch = createAuthFetch({
    fetch: rawFetch,
    refresher,
    onSessionExpired: () => onExpired?.(),
  });

  const headerMiddleware: Middleware = {
    onRequest({ request }) {
      for (const [k, v] of Object.entries(headers())) request.headers.set(k, v);
      if (!request.headers.has('Accept')) {
        request.headers.set('Accept', 'application/json, application/problem+json');
      }
      return request;
    },
  };

  const api = createClient<paths>({ baseUrl, fetch: authFetch, credentials: 'include' });
  api.use(headerMiddleware);

  return {
    api,
    fetch: authFetch,
    refresher,
    app: opts.app,
    version: opts.version,
    baseUrl,
    headers,
    setOnSessionExpired(cb) {
      onExpired = cb;
    },
    expireSession() {
      onExpired?.();
    },
  };
}

interface FetchResultLike<T> {
  data?: T;
  error?: unknown;
  response: Response;
}

/**
 * Turns an openapi-fetch result into data-or-throw for TanStack Query:
 * `queryFn: () => unwrap(api.GET('/me'))`. Network failures become `ApiError` code NETWORK.
 */
export async function unwrap<T>(promise: Promise<FetchResultLike<T>>): Promise<T> {
  let result: FetchResultLike<T>;
  try {
    result = await promise;
  } catch (e) {
    if (e instanceof ApiError) throw e;
    throw ApiError.network(e);
  }
  if (result.error !== undefined || !result.response.ok) {
    throw ApiError.fromResponse(result.response, result.error);
  }
  return result.data as T;
}

/** `Idempotency-Key` header for creating POSTs (doc 11; one key per user intent, reused on retry). */
export function idempotencyHeaders(key: string): { 'Idempotency-Key': string } {
  return { 'Idempotency-Key': key };
}
