import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createMemoryHistory, RouterProvider } from '@tanstack/react-router';
import { beforeEach, describe, expect, it } from 'vitest';
import { createApiClient } from '@rovo/api-client';
import { ApiClientProvider } from '@rovo/api-client/react';
import { SessionProvider } from '@rovo/auth';
import { expectNoAxeViolations, render, screen, waitFor } from '@rovo/testing';
import { authHandlers, createAuthState, type FakeAuthState, fixtures, setupMsw } from '@rovo/testing/msw';
import { createAppRouter } from './router';

let state: FakeAuthState;
const server = setupMsw();

beforeEach(() => {
  state = createAuthState({ loginAs: fixtures.restaurantOwner() });
  server.use(...authHandlers(state));
});

function renderAt(path: string) {
  const router = createAppRouter(createMemoryHistory({ initialEntries: [path] }));
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const client = createApiClient({ app: 'restaurant', version: '0.0.0-test' });
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

describe('restaurant app shell', () => {
  it('opens the order inbox with the tab bar for an owner and passes axe', async () => {
    state.session = fixtures.restaurantOwner();
    renderAt('/inbox');
    expect(await screen.findByTestId('inbox-screen')).toBeInTheDocument();
    expect(screen.getByTestId('nav-inbox')).toHaveAttribute('aria-current', 'page');
    for (const tab of ['menu', 'hours', 'analytics', 'payouts']) expect(screen.getByTestId(`nav-${tab}`)).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('redirects the host root to the inbox', async () => {
    state.session = fixtures.restaurantOwner();
    const router = renderAt('/');
    await waitFor(() => expect(router.state.location.pathname).toBe('/inbox'));
    expect(await screen.findByTestId('inbox-screen')).toBeInTheDocument();
  });

  it('shows the phone login screen and passes axe', async () => {
    renderAt('/login');
    expect(await screen.findByTestId('login-phone-form')).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('sends a guest who opens a protected screen to /login with a return path', async () => {
    const router = renderAt('/menu');
    expect(await screen.findByTestId('login-phone-form')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/login');
    expect(router.state.location.search).toEqual({ redirect: '/menu' });
  });

  it('shows "wrong app" to an account with another role', async () => {
    state.session = fixtures.rider();
    renderAt('/menu');
    expect(await screen.findByTestId('page-wrong-app')).toBeInTheDocument();
  });

  it('shows 404 for an unknown path', async () => {
    renderAt('/no/such/page/here');
    expect(await screen.findByTestId('page-not-found')).toBeInTheDocument();
    await expectNoAxeViolations();
  });
});
