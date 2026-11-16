import { createRootRoute, createRoute, createRouter, Link, Outlet, type RouterHistory, redirect, useNavigate } from '@tanstack/react-router';
import { useTranslation } from '@rovo/i18n';
import { NotFoundScreen, PlaceholderScreen } from '@rovo/shell';
import { Protected } from './guard';
import { RootLayout } from './layout';
import { LoginScreen } from './login';

const rootRoute = createRootRoute({ component: () => <Outlet />, notFoundComponent: NotFound });

function NotFound() {
  const navigate = useNavigate();
  return <NotFoundScreen onHome={() => void navigate({ to: '/inbox' })} />;
}

function makeScreen(title: string, note?: string, testId?: string, withBack = false) {
  return function Screen() {
    const { t } = useTranslation('restaurant');
    const navigate = useNavigate();
    return (
      <Protected>
        <PlaceholderScreen title={t(title)} note={note} testId={testId} onBack={withBack ? () => void navigate({ to: '/inbox' }) : undefined} />
      </Protected>
    );
  };
}

const layoutRoute = createRoute({ getParentRoute: () => rootRoute, id: 'layout', component: RootLayout });
const tab = <const P extends string>(path: P, title: string, testId: string) =>
  createRoute({ getParentRoute: () => layoutRoute, path, component: makeScreen(title, undefined, testId) });

const inbox = tab('/inbox', 'screens.inbox.title', 'inbox-screen');
const menu = tab('/menu', 'screens.menu.title', 'menu-screen');
const hours = tab('/hours', 'screens.hours.title', 'hours-screen');
const analytics = tab('/analytics', 'screens.analytics.title', 'analytics-screen');
const payouts = tab('/payouts', 'screens.payouts.title', 'payouts-screen');

const full = <const P extends string>(path: P, title: string, testId: string, note?: string) =>
  createRoute({ getParentRoute: () => rootRoute, path, component: makeScreen(title, note, testId, true) });

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  validateSearch: (s: Record<string, unknown>): { redirect?: string } => ({ redirect: typeof s.redirect === 'string' ? s.redirect : undefined }),
  component: LoginScreen,
});

// The host root opens the order inbox (the app's home tab).
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/inbox', replace: true });
  },
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  layoutRoute.addChildren([inbox, menu, hours, analytics, payouts]),
  loginRoute,
  full('/outlets', 'screens.outlets.title', 'outlets-screen'),
  full('/start', 'screens.start.title', 'start-shift-screen', 'One tap unlocks audio, takes the wake lock, checks notification permission and connects SSE (doc 18 §7).'),
  full('/orders', 'screens.orders.title', 'orders-history-screen'),
  full('/orders/$orderId', 'screens.orderDetail.title', 'order-detail-screen'),
  full('/availability', 'screens.availability.title', 'availability-screen'),
  full('/menu/categories', 'screens.menu.title', 'menu-categories-screen'),
  full('/menu/items/new', 'screens.menuItem.title', 'menu-item-new-screen'),
  full('/menu/items/$itemId', 'screens.menuItem.title', 'menu-item-screen'),
  full('/devices', 'screens.devices.title', 'devices-screen'),
  full('/payouts/$payoutId', 'screens.payoutDetail.title', 'payout-detail-screen'),
  full('/profile', 'screens.payouts.title', 'profile-screen'),
  full('/settings/alerts', 'screens.alerts.title', 'alerts-screen'),
  full('/help', 'screens.help.title', 'help-screen'),
]);

/** `history` is injectable so tests can drive the app with a memory history. */
export function createAppRouter(history?: RouterHistory) {
  return createRouter({ routeTree, history, defaultPreload: 'intent', defaultNotFoundComponent: NotFound, scrollRestoration: true });
}

export const router = createAppRouter();

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

export { Link };
