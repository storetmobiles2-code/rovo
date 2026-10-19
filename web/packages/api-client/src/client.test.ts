import { describe, expect, it, vi } from 'vitest';
import { createApiClient, idempotencyHeaders, unwrap } from './client';
import { ApiError, errorCode } from './errors';
import { createRefresher } from './refresh';

type Handler = (req: Request) => Response | Promise<Response>;

function json(body: unknown, status = 200, type = 'application/json') {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': type } });
}
const problem = (status: number, code: string, extra: object = {}) =>
  json({ type: `/problems/${code}`, title: code, status, code, ...extra }, status, 'application/problem+json');

function mockServer(handler: Handler) {
  const calls: Request[] = [];
  const fetch = vi.fn(async (req: Request) => {
    calls.push(req.clone());
    return handler(req);
  });
  return { fetch, calls };
}

describe('createApiClient', () => {
  it('sends credentials, base URL and rovo headers on every request', async () => {
    const { fetch, calls } = mockServer(() =>
      json({ id: 'u1', kind: 'CUSTOMER', roles: [], preferredLocale: 'en', accessExpiresAt: 'x' }),
    );
    const { api } = createApiClient({ app: 'customer', version: '1.2.3', fetch, getLanguage: () => 'te' });
    const me = await unwrap(api.GET('/me'));
    expect(me.id).toBe('u1');
    const req = calls[0]!;
    expect(new URL(req.url, 'http://x').pathname).toBe('/api/v1/me');
    expect(req.credentials).toBe('include');
    expect(req.headers.get('X-Rovo-Client')).toBe('customer@1.2.3');
    expect(req.headers.get('X-Rovo-App-Version')).toBe('customer/1.2.3');
    expect(req.headers.get('Accept-Language')).toMatch(/^te-IN/);
  });

  it('refreshes once on 401 and retries the original request with its body', async () => {
    let refreshed = false;
    const { fetch, calls } = mockServer(async (req) => {
      const path = new URL(req.url, 'http://x').pathname;
      if (path === '/api/v1/auth/refresh') {
        refreshed = true;
        return json({ accessExpiresAt: 'x' });
      }
      if (!refreshed) return problem(401, 'TOKEN_EXPIRED');
      return json({ id: 'u1', kind: 'CUSTOMER', roles: [], preferredLocale: 'en', accessExpiresAt: 'x', echo: await req.text() });
    });
    const { api } = createApiClient({ app: 'customer', version: '1.0.0', fetch });
    const me = await unwrap(api.PATCH('/me', { body: { fullName: 'Ravi' } }));
    expect(me.id).toBe('u1');
    expect((me as unknown as { echo: string }).echo).toBe('{"fullName":"Ravi"}');
    expect(calls.map((c) => `${c.method} ${new URL(c.url, 'http://x').pathname}`)).toEqual([
      'PATCH /api/v1/me',
      'POST /api/v1/auth/refresh',
      'PATCH /api/v1/me',
    ]);
    expect(calls[1]!.headers.get('X-Rovo-Client')).toBe('customer@1.0.0');
  });

  it('single-flights concurrent refreshes', async () => {
    let refreshed = false;
    let refreshCalls = 0;
    const { fetch } = mockServer(async (req) => {
      const path = new URL(req.url, 'http://x').pathname;
      if (path === '/api/v1/auth/refresh') {
        refreshCalls++;
        await new Promise((r) => setTimeout(r, 10));
        refreshed = true;
        return json({ accessExpiresAt: 'x' });
      }
      return refreshed ? json({ id: 'u1', kind: 'CUSTOMER', roles: [], preferredLocale: 'en', accessExpiresAt: 'x' }) : problem(401, 'TOKEN_EXPIRED');
    });
    const { api } = createApiClient({ app: 'rider', version: '1.0.0', fetch });
    const results = await Promise.all([unwrap(api.GET('/me')), unwrap(api.GET('/me')), unwrap(api.GET('/me'))]);
    expect(results).toHaveLength(3);
    expect(refreshCalls).toBe(1);
  });

  it('reports an expired session when refresh is rejected and surfaces the 401', async () => {
    const onSessionExpired = vi.fn();
    const { fetch } = mockServer((req) =>
      new URL(req.url, 'http://x').pathname === '/api/v1/auth/refresh'
        ? problem(401, 'SESSION_REVOKED')
        : problem(401, 'TOKEN_EXPIRED'),
    );
    const { api } = createApiClient({ app: 'customer', version: '1.0.0', fetch, onSessionExpired });
    await expect(unwrap(api.GET('/me'))).rejects.toMatchObject({ status: 401, code: 'TOKEN_EXPIRED' });
    expect(onSessionExpired).toHaveBeenCalledTimes(1);
  });

  it('treats 409 REFRESH_RACE as success (another tab rotated the token)', async () => {
    let n = 0;
    const { fetch } = mockServer((req) => {
      if (new URL(req.url, 'http://x').pathname === '/api/v1/auth/refresh') return problem(409, 'REFRESH_RACE');
      return ++n === 1 ? problem(401, 'TOKEN_EXPIRED') : json({ id: 'u2', kind: 'RIDER', roles: [], preferredLocale: 'te', accessExpiresAt: 'x' });
    });
    const { api } = createApiClient({ app: 'rider', version: '1.0.0', fetch });
    await expect(unwrap(api.GET('/me'))).resolves.toMatchObject({ id: 'u2' });
  });

  it('never refreshes for auth endpoints', async () => {
    const { fetch, calls } = mockServer(() => problem(401, 'OTP_INVALID'));
    const { api } = createApiClient({ app: 'customer', version: '1.0.0', fetch });
    const err = await unwrap(
      api.POST('/auth/otp/verify', { body: { challengeId: 'c', code: '000000', deviceInstallId: 'd' } }),
    ).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).code).toBe('OTP_INVALID');
    expect(calls).toHaveLength(1);
  });

  it('maps network failures to ApiError NETWORK', async () => {
    const fetch = vi.fn(async () => {
      throw new TypeError('Failed to fetch');
    });
    const { api } = createApiClient({ app: 'customer', version: '1.0.0', fetch });
    const err = (await unwrap(api.GET('/me')).catch((e: unknown) => e)) as ApiError;
    expect(err.code).toBe('NETWORK');
    expect(err.isNetworkError).toBe(true);
  });

  it('builds Idempotency-Key headers', () => {
    expect(idempotencyHeaders('k1')).toEqual({ 'Idempotency-Key': 'k1' });
  });
});

describe('ApiError', () => {
  it('parses problem+json with field errors, trace id and Retry-After', async () => {
    const res = new Response(
      JSON.stringify({
        type: '/problems/validation',
        title: 'Validation failed',
        status: 422,
        code: 'VALIDATION_FAILED',
        traceId: 't-1',
        fieldErrors: [{ field: 'landmark', code: 'required' }],
      }),
      { status: 422, headers: { 'Content-Type': 'application/problem+json', 'Retry-After': '30' } },
    );
    const err = await ApiError.read(res);
    expect(err).toMatchObject({ status: 422, code: 'VALIDATION_FAILED', traceId: 't-1', retryAfterSeconds: 30 });
    expect(err.fieldErrors).toEqual([{ field: 'landmark', code: 'required' }]);
  });

  it('keeps extension members such as the fresh quote on 409 QUOTE_CHANGED', () => {
    const err = ApiError.fromResponse(new Response(null, { status: 409 }), {
      code: 'QUOTE_CHANGED',
      quote: { quoteId: 'q2' },
    });
    expect(err.problem.quote).toEqual({ quoteId: 'q2' });
  });

  it('falls back to status-derived codes for non-problem bodies', () => {
    expect(ApiError.fromResponse(new Response('<html>', { status: 502 }), '<html>').code).toBe('INTERNAL');
    expect(ApiError.fromResponse(new Response('', { status: 429 })).code).toBe('RATE_LIMITED');
    expect(ApiError.fromResponse(new Response('', { status: 404 })).code).toBe('NOT_FOUND');
    expect(errorCode(new Error('x'))).toBe('UNKNOWN');
  });
});

describe('createRefresher', () => {
  it('uses the Web Locks API when available', async () => {
    const names: string[] = [];
    const locks = {
      request<T>(name: string, cb: () => Promise<T>): Promise<T> {
        names.push(name);
        return cb();
      },
    };
    const r = createRefresher({
      url: 'http://localhost/api/v1/auth/refresh',
      fetch: async () => json({}),
      headers: () => ({}),
      locks,
    });
    await expect(r.refresh()).resolves.toBe('ok');
    expect(names).toEqual(['rovo-refresh']);
  });

  it('reports 5xx and network errors as unavailable (not a logout)', async () => {
    const r = createRefresher({ url: 'http://localhost/r', fetch: async () => new Response('', { status: 503 }), headers: () => ({}), locks: null });
    await expect(r.refresh()).resolves.toBe('unavailable');
  });
});
