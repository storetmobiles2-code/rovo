import { createRootRoute, createRoute, createRouter, Outlet, type RouterHistory } from '@tanstack/react-router';
import { createElement } from 'react';
import type { RoleName } from '@rovo/auth';
import { RootLayout } from './layout';
import { LoginScreen } from './login';
import { NotFound } from './not-found';
import { AdminScreenSkeleton } from './placeholder';
import { ADMIN_SCREENS, type AdminPath, type AdminScreen } from './screens';

const rootRoute = createRootRoute({ component: Outlet, notFoundComponent: NotFound });

// Everything under the layout route needs an admin session (guard in RootLayout).
const layoutRoute = createRoute({ getParentRoute: () => rootRoute, id: 'console', component: RootLayout });

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  validateSearch: (s: Record<string, unknown>): { redirect?: string } => (typeof s.redirect === 'string' ? { redirect: s.redirect } : {}),
  component: LoginScreen,
});

/** Typed by its literal path so `<Link to>` / `navigate` stay checked; data comes from the table. */
function screenRoute<const P extends AdminPath>(path: P) {
  const screen: AdminScreen = ADMIN_SCREENS.find((s) => s.path === path)!;
  return createRoute({
    getParentRoute: () => layoutRoute,
    path,
    staticData: { roles: screen.roles },
    component: () => createElement(AdminScreenSkeleton, { screen }),
  });
}

// One entry per ADMIN_SCREENS row (app.test.tsx fails if a row has no route).
const routeTree = rootRoute.addChildren([
  layoutRoute.addChildren([
    screenRoute('/'),
    screenRoute('/live'),
    screenRoute('/orders'),
    screenRoute('/orders/$orderId'),
    screenRoute('/dispatch'),
    screenRoute('/restaurants'),
    screenRoute('/restaurants/$restaurantId'),
    screenRoute('/riders'),
    screenRoute('/riders/$riderId'),
    screenRoute('/customers'),
    screenRoute('/customers/$customerId'),
    screenRoute('/coupons'),
    screenRoute('/approvals'),
    screenRoute('/finance/payouts'),
    screenRoute('/finance/payouts/$batchId'),
    screenRoute('/zones'),
    screenRoute('/pricing'),
    screenRoute('/support/tickets'),
    screenRoute('/support/tickets/$ticketId'),
    screenRoute('/audit-log'),
    screenRoute('/reports'),
  ]),
  loginRoute,
]);

/** `history` is injectable so tests can drive the console with a memory history. */
export function createAdminRouter(history?: RouterHistory) {
  return createRouter({ routeTree, history, defaultPreload: 'intent', defaultNotFoundComponent: NotFound, scrollRestoration: true });
}

export const router = createAdminRouter();

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
  interface StaticDataRouteOption {
    /** Roles admitted besides ADMIN_SUPER; empty or absent = every admin (doc 17 §3.4). */
    roles?: readonly RoleName[];
  }
}
