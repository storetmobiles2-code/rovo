/**
 * MSW scaffold. Until handlers are generated from OpenAPI examples (doc 17 §13.5), these
 * hand-written handlers model the auth + session contract of doc 11 §2.2 with an in-memory
 * state object tests can inspect and mutate.
 */
import { http, HttpResponse, type RequestHandler } from 'msw';
import { setupServer, type SetupServer } from 'msw/node';
import { afterAll, afterEach, beforeAll } from 'vitest';
import type { Me } from '@rovo/api-client';

export const API = '*/api/v1';

const at = () => new Date(Date.now() + 10 * 60_000).toISOString();

export const fixtures = {
  customer: (): Me => ({
    id: 'usr_customer_1',
    kind: 'CUSTOMER',
    fullName: 'Lakshmi',
    phoneMasked: '+91 ••••• •3210',
    email: null,
    preferredLocale: 'en',
    roles: [{ role: 'CUSTOMER', cityId: 'city_mbnr', restaurantId: null }],
    activeContext: null,
    consents: {},
    accessExpiresAt: at(),
  }),
  restaurantOwner: (): Me => ({
    id: 'usr_owner_1',
    kind: 'PARTNER',
    fullName: 'Srinivas',
    phoneMasked: '+91 ••••• •1111',
    email: null,
    preferredLocale: 'te',
    roles: [{ role: 'RESTAURANT_OWNER', cityId: 'city_mbnr', restaurantId: 'rst_1' }],
    activeContext: 'restaurant:rst_1',
    consents: {},
    accessExpiresAt: at(),
  }),
  rider: (): Me => ({
    id: 'usr_rider_1',
    kind: 'RIDER',
    fullName: 'Ramesh',
    phoneMasked: '+91 ••••• •2222',
    email: null,
    preferredLocale: 'te',
    roles: [{ role: 'RIDER', cityId: 'city_mbnr', restaurantId: null }],
    activeContext: null,
    consents: {},
    accessExpiresAt: at(),
  }),
  admin: (): Me => ({
    id: 'usr_admin_1',
    kind: 'ADMIN',
    fullName: 'Ops Admin',
    phoneMasked: null,
    email: 'ops@rovo.example',
    preferredLocale: 'en',
    roles: [{ role: 'ADMIN_OPS', cityId: 'city_mbnr', restaurantId: null }],
    activeContext: null,
    consents: {},
    accessExpiresAt: at(),
  }),
};

export interface FakeAuthState {
  /** Current session (null = guest). */
  session: Me | null;
  /** User that a successful OTP/admin login creates. */
  loginAs: Me;
  otp: string;
  totp: string;
  adminEmail: string;
  adminPassword: string;
  /** When false, /auth/refresh answers 401 SESSION_REVOKED. */
  refreshValid: boolean;
  /** When true, the next GET /me answers 401 TOKEN_EXPIRED once (exercises refresh-retry). */
  expireAccessOnce: boolean;
  calls: string[];
}

export function createAuthState(overrides: Partial<FakeAuthState> = {}): FakeAuthState {
  return {
    session: null,
    loginAs: fixtures.customer(),
    otp: '123456',
    totp: '654321',
    adminEmail: 'ops@rovo.example',
    adminPassword: 'correct horse battery staple',
    refreshValid: true,
    expireAccessOnce: false,
    calls: [],
    ...overrides,
  };
}

const problem = (status: number, code: string, extra: Record<string, unknown> = {}) =>
  HttpResponse.json(
    { type: `/problems/${code.toLowerCase().replace(/_/g, '-')}`, title: code, status, code, traceId: 'trace-test', ...extra },
    { status, headers: { 'Content-Type': 'application/problem+json' } },
  );

export function authHandlers(state: FakeAuthState): RequestHandler[] {
  const log = (s: string) => state.calls.push(s);
  return [
    http.post(`${API}/auth/otp/request`, async ({ request }) => {
      log('otp.request');
      const body = (await request.json()) as { phone?: string };
      if (!body.phone || !/^\+91[6-9]\d{9}$/.test(body.phone)) {
        return problem(422, 'VALIDATION_FAILED', { fieldErrors: [{ field: 'phone', code: 'invalidPhone' }] });
      }
      return HttpResponse.json(
        { challengeId: 'chl_1', resendAfterSeconds: 30, expiresAt: new Date(Date.now() + 300_000).toISOString() },
        { status: 202 },
      );
    }),
    http.post(`${API}/auth/otp/verify`, async ({ request }) => {
      log('otp.verify');
      const body = (await request.json()) as { code?: string; deviceInstallId?: string };
      if (!body.deviceInstallId) return problem(422, 'VALIDATION_FAILED');
      if (body.code !== state.otp) return problem(400, 'OTP_INVALID');
      state.session = state.loginAs;
      return HttpResponse.json({
        user: { id: state.loginAs.id, fullName: state.loginAs.fullName ?? null, roles: state.loginAs.roles.map((r) => r.role) },
        isNewUser: false,
        accessExpiresAt: state.loginAs.accessExpiresAt,
        needsAccountConfirmation: false,
      });
    }),
    http.post(`${API}/auth/admin/login`, async ({ request }) => {
      log('admin.login');
      const body = (await request.json()) as { email?: string; password?: string };
      if (body.email !== state.adminEmail || body.password !== state.adminPassword) {
        return problem(401, 'INVALID_CREDENTIALS');
      }
      return HttpResponse.json({ mfaToken: 'mfa_1', mfaMethods: ['TOTP', 'RECOVERY_CODE'] });
    }),
    http.post(`${API}/auth/admin/mfa`, async ({ request }) => {
      log('admin.mfa');
      const body = (await request.json()) as { mfaToken?: string; totp?: string };
      if (body.mfaToken !== 'mfa_1' || body.totp !== state.totp) return problem(401, 'MFA_INVALID');
      state.session = state.loginAs;
      return HttpResponse.json({ accessExpiresAt: state.loginAs.accessExpiresAt });
    }),
    http.post(`${API}/auth/refresh`, () => {
      log('refresh');
      if (!state.session || !state.refreshValid) return problem(401, 'SESSION_REVOKED');
      return HttpResponse.json({ accessExpiresAt: at() });
    }),
    http.post(`${API}/auth/logout`, () => {
      log('logout');
      state.session = null;
      return new HttpResponse(null, { status: 204 });
    }),
    http.get(`${API}/me`, () => {
      log('me');
      if (state.expireAccessOnce) {
        state.expireAccessOnce = false;
        return problem(401, 'TOKEN_EXPIRED');
      }
      if (!state.session) return problem(401, 'UNAUTHENTICATED');
      return HttpResponse.json(state.session);
    }),
    http.patch(`${API}/me`, async ({ request }) => {
      log('me.patch');
      if (!state.session) return problem(401, 'UNAUTHENTICATED');
      const body = (await request.json()) as Partial<Me>;
      state.session = { ...state.session, ...body };
      return HttpResponse.json(state.session);
    }),
  ];
}

/** Starts an MSW server for the current test file (unhandled requests fail the test). */
export function setupMsw(...handlers: RequestHandler[]): SetupServer {
  const server = setupServer(...handlers);
  beforeAll(() => server.listen({ onUnhandledFrame: 'error' }));
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());
  return server;
}

export { http, HttpResponse };
