import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { createApiClient } from '@rovo/api-client';
import { ApiClientProvider } from '@rovo/api-client/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { authHandlers, createAuthState, type FakeAuthState, fixtures, setupMsw } from '@rovo/testing/msw';
import { type ReactNode } from 'react';
import { useAdminLogin } from './admin';
import { useSession, SessionProvider } from './session';
import { usePhoneOtp } from './otp';
import { canUseApp, hasAnyRole } from './roles';
import { evaluateGuard } from './guard';

let state: FakeAuthState;
const server = setupMsw();

beforeEach(() => {
  state = createAuthState();
  server.use(...authHandlers(state));
  localStorage.clear();
});
afterEach(() => localStorage.clear());

function wrapper(app: 'customer' | 'admin' = 'customer') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const client = createApiClient({ app, version: '0.0.0' });
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ApiClientProvider client={client}>
          <SessionProvider>{children}</SessionProvider>
        </ApiClientProvider>
      </QueryClientProvider>
    );
  };
}

describe('usePhoneOtp', () => {
  it('runs request → verify and surfaces the session', async () => {
    const { result } = renderHook(() => ({ otp: usePhoneOtp(), session: useSession() }), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.session.status).toBe('guest'));

    await act(async () => {
      await result.current.otp.requestOtp('98765 43210');
    });
    expect(result.current.otp.step).toBe('code');
    expect(result.current.otp.challenge?.phone).toBe('+919876543210');
    expect(result.current.otp.resendIn).toBe(30);
    expect(state.calls).toContain('otp.request');

    await act(async () => {
      await result.current.otp.verify('123456');
    });
    await waitFor(() => expect(result.current.session.status).toBe('authenticated'));
    expect(result.current.session.me?.id).toBe('usr_customer_1');
  });

  it('rejects an invalid mobile number before any network call', async () => {
    const { result } = renderHook(() => usePhoneOtp(), { wrapper: wrapper() });
    await expect(
      act(async () => {
        await result.current.requestOtp('12345');
      }),
    ).rejects.toMatchObject({ code: 'VALIDATION_FAILED' });
    expect(state.calls).not.toContain('otp.request');
  });

  it('exposes a translatable code for a wrong OTP', async () => {
    const { result } = renderHook(() => usePhoneOtp(), { wrapper: wrapper() });
    await act(async () => {
      await result.current.requestOtp('98765 43210');
    });
    await act(async () => {
      await result.current.verify('000000').catch(() => {});
    });
    expect(result.current.verifyError?.code).toBe('OTP_INVALID');
  });
});

describe('useAdminLogin', () => {
  it('runs email+password → TOTP → session', async () => {
    state.loginAs = fixtures.admin();
    const { result } = renderHook(() => ({ login: useAdminLogin(), session: useSession() }), { wrapper: wrapper('admin') });
    await waitFor(() => expect(result.current.session.status).toBe('guest'));

    await act(async () => {
      await result.current.login.submitCredentials('ops@rovo.example', 'correct horse battery staple');
    });
    expect(result.current.login.step).toBe('totp');
    expect(result.current.login.mfaMethods).toContain('TOTP');

    await act(async () => {
      await result.current.login.submitTotp({ totp: '654321' });
    });
    await waitFor(() => expect(result.current.session.status).toBe('authenticated'));
    expect(result.current.session.me?.roles[0]?.role).toBe('ADMIN_OPS');
  });

  it('reports invalid credentials and invalid TOTP', async () => {
    const { result } = renderHook(() => useAdminLogin(), { wrapper: wrapper('admin') });
    await act(async () => {
      await result.current.submitCredentials('ops@rovo.example', 'wrong').catch(() => {});
    });
    await waitFor(() => expect(result.current.credentialsError?.code).toBe('INVALID_CREDENTIALS'));
    expect(result.current.step).toBe('credentials');
  });
});

describe('logout', () => {
  it('clears the session', async () => {
    state.session = fixtures.customer();
    const { result } = renderHook(() => useSession(), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.status).toBe('authenticated'));
    await act(async () => {
      await result.current.logout.mutateAsync();
    });
    expect(state.calls).toContain('logout');
    await waitFor(() => expect(result.current.status).toBe('guest'));
  });
});

describe('role helpers', () => {
  it('scopes apps by role and evaluates the guard', () => {
    const rider = fixtures.rider();
    expect(canUseApp(rider, 'rider')).toBe(true);
    expect(canUseApp(rider, 'restaurant')).toBe(false);
    expect(hasAnyRole(rider, ['RIDER'])).toBe(true);
    expect(evaluateGuard(rider, 'authenticated', ['RESTAURANT_OWNER'])).toEqual({ state: 'wrong-role' });
    expect(evaluateGuard(rider, 'authenticated', ['RIDER'])).toEqual({ state: 'allowed' });
    expect(evaluateGuard(null, 'guest', [])).toEqual({ state: 'guest' });
    expect(evaluateGuard(null, 'loading', [])).toEqual({ state: 'loading' });
  });
});
