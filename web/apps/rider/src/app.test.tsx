import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createMemoryHistory, RouterProvider } from '@tanstack/react-router';
import { beforeEach, describe, expect, it } from 'vitest';
import { createApiClient } from '@rovo/api-client';
import { ApiClientProvider } from '@rovo/api-client/react';
import { SessionProvider } from '@rovo/auth';
import { expectNoAxeViolations, render, screen } from '@rovo/testing';
import { authHandlers, createAuthState, type FakeAuthState, fixtures, setupMsw } from '@rovo/testing/msw';
import { createAppRouter } from './router';

let state: FakeAuthState;
const server = setupMsw();

beforeEach(() => {
  state = createAuthState({ loginAs: fixtures.rider() });
  server.use(...authHandlers(state));
});

function renderAt(path: string) {
  const router = createAppRouter(createMemoryHistory({ initialEntries: [path] }));
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const client = createApiClient({ app: 'rider', version: '0.0.0-test' });
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

describe('rider app shell', () => {
  it('opens home with the tab bar for a delivery partner and passes axe', async () => {
    state.session = fixtures.rider();
    renderAt('/');
    expect(await screen.findByTestId('home-screen')).toBeInTheDocument();
    expect(screen.getByTestId('nav-home')).toHaveAttribute('aria-current', 'page');
    for (const tab of ['earnings', 'cash', 'history', 'profile']) expect(screen.getByTestId(`nav-${tab}`)).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('shows the phone login screen and passes axe', async () => {
    renderAt('/login');
    expect(await screen.findByTestId('login-phone-form')).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('sends a guest who opens a protected screen to /login with a return path', async () => {
    const router = renderAt('/earnings');
    expect(await screen.findByTestId('login-phone-form')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/login');
    expect(router.state.location.search).toEqual({ redirect: '/earnings' });
  });

  it('shows "wrong app" to an account with another role', async () => {
    state.session = fixtures.customer();
    renderAt('/earnings');
    expect(await screen.findByTestId('page-wrong-app')).toBeInTheDocument();
  });

  it('shows 404 for an unknown path', async () => {
    renderAt('/no/such/page/here');
    expect(await screen.findByTestId('page-not-found')).toBeInTheDocument();
    await expectNoAxeViolations();
  });
});
