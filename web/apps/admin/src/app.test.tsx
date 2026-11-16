import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createMemoryHistory, RouterProvider } from '@tanstack/react-router';
import { beforeEach, describe, expect, it } from 'vitest';
import { createApiClient } from '@rovo/api-client';
import { ApiClientProvider } from '@rovo/api-client/react';
import { SessionProvider } from '@rovo/auth';
import { expectNoAxeViolations, render, screen, userEvent, waitFor } from '@rovo/testing';
import { authHandlers, createAuthState, type FakeAuthState, fixtures, setupMsw } from '@rovo/testing/msw';
import { createAdminRouter } from './router';
import { ADMIN_SCREENS, canOpenScreen } from './screens';

let state: FakeAuthState;
const server = setupMsw();

beforeEach(() => {
  state = createAuthState({ loginAs: fixtures.admin() });
  server.use(...authHandlers(state));
});

function renderAt(path: string) {
  const router = createAdminRouter(createMemoryHistory({ initialEntries: [path] }));
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const client = createApiClient({ app: 'admin', version: '0.0.0-test' });
  render(
    <QueryClientProvider client={queryClient}>
      <ApiClientProvider client={client}>
        <SessionProvider>
          <RouterProvider router={router} />
        </SessionProvider>
      </ApiClientProvider>
    </QueryClientProvider>,
  );
  return router;
}

describe('admin console shell', () => {
  it('sends a guest to /login and keeps the page they asked for', async () => {
    const router = renderAt('/coupons');
    expect(await screen.findByTestId('login-credentials-form')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/login');
    expect(router.state.location.search).toEqual({ redirect: '/coupons' });
    await expectNoAxeViolations();
  });

  it('logs in with email + password + TOTP and lands on the requested screen', async () => {
    const router = renderAt('/login?redirect=%2Fcoupons');
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText(/work email/i), state.adminEmail);
    await user.type(screen.getByLabelText(/password/i), state.adminPassword);
    await user.click(screen.getByTestId('login-submit-credentials'));
    await screen.findByTestId('login-totp-form');
    await user.type(screen.getByLabelText(/authenticator code/i), state.totp);
    expect(await screen.findByTestId('coupons-screen')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/coupons');
  });

  it('shows the dashboard with a role-filtered sidebar for an ops admin, and passes axe', async () => {
    state.session = fixtures.admin();
    renderAt('/');
    expect(await screen.findByTestId('dashboard-screen')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'Today at a glance' })).toBeInTheDocument();
    expect(screen.getByTestId('nav-dashboard')).toHaveAttribute('aria-current', 'page');
    expect(screen.getByTestId('nav-orders')).toBeInTheDocument();
    expect(screen.getByTestId('nav-zones')).toBeInTheDocument();
    // ADMIN_OPS does not get finance, support or super-only screens.
    expect(screen.queryByTestId('nav-approvals')).not.toBeInTheDocument();
    expect(screen.queryByTestId('nav-payouts')).not.toBeInTheDocument();
    expect(screen.queryByTestId('nav-tickets')).not.toBeInTheDocument();
    expect(screen.queryByTestId('nav-auditLog')).not.toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('refuses a screen the role may not open, even by direct URL', async () => {
    state.session = fixtures.admin();
    renderAt('/audit-log');
    expect(await screen.findByTestId('page-forbidden')).toBeInTheDocument();
    expect(screen.queryByTestId('audit-log-screen')).not.toBeInTheDocument();
  });

  it('lets a super admin open every screen', async () => {
    state.session = { ...fixtures.admin(), roles: [{ role: 'ADMIN_SUPER', cityId: null, restaurantId: null }] };
    const router = renderAt('/');
    await screen.findByTestId('dashboard-screen');
    for (const s of ADMIN_SCREENS) {
      await router.navigate({ to: s.path.replace(/\$\w+/, 'x1') });
      expect(await screen.findByTestId(s.testId)).toBeInTheDocument();
    }
    expect(screen.getAllByRole('link').length).toBeGreaterThanOrEqual(ADMIN_SCREENS.filter((s) => 'nav' in s).length);
  });

  it('shows "wrong app" to a non-admin account', async () => {
    state.session = fixtures.rider();
    renderAt('/');
    expect(await screen.findByTestId('page-wrong-app')).toBeInTheDocument();
  });

  it('logs out and returns to the login page', async () => {
    state.session = fixtures.admin();
    const router = renderAt('/');
    await userEvent.click(await screen.findByTestId('logout-button'));
    await waitFor(() => expect(router.state.location.pathname).toBe('/login'));
    expect(state.session).toBeNull();
  });

  it('shows 404 for an unknown path', async () => {
    renderAt('/nope');
    expect(await screen.findByTestId('page-not-found')).toBeInTheDocument();
  });
});

describe('admin route helpers', () => {
  it('canOpenScreen: open screens, role match, super override', () => {
    expect(canOpenScreen([], ['ADMIN_SUPPORT'])).toBe(true);
    expect(canOpenScreen(['ADMIN_FINANCE'], ['ADMIN_OPS'])).toBe(false);
    expect(canOpenScreen(['ADMIN_FINANCE'], ['ADMIN_OPS', 'ADMIN_FINANCE'])).toBe(true);
    expect(canOpenScreen(['ADMIN_FINANCE'], ['ADMIN_SUPER'])).toBe(true);
  });
});
