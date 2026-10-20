import { createRootRoute, createRoute, createRouter, Link, Outlet, useNavigate } from '@tanstack/react-router';
import { useTranslation } from '@rovo/i18n';
import { NotFoundScreen, PlaceholderScreen } from '@rovo/shell';
import { Protected } from './guard';
import { RootLayout } from './layout';
import { LoginScreen } from './login';

const rootRoute = createRootRoute({ component: () => <Outlet />, notFoundComponent: NotFound });

function NotFound() {
  const navigate = useNavigate();
  return <NotFoundScreen onHome={() => void navigate({ to: '/' })} />;
}

function makeScreen(title: string, note?: string, testId?: string, withBack = true) {
  return function Screen() {
    const { t } = useTranslation('rider');
    const navigate = useNavigate();
    return (
      <Protected>
        <PlaceholderScreen title={t(title)} note={note} testId={testId} onBack={withBack ? () => void navigate({ to: '/' }) : undefined} />
      </Protected>
    );
  };
}

const layoutRoute = createRoute({ getParentRoute: () => rootRoute, id: 'layout', component: RootLayout });
const tab = <const P extends string>(path: P, title: string, testId: string) =>
  createRoute({ getParentRoute: () => layoutRoute, path, component: makeScreen(title, undefined, testId, false) });
const full = <const P extends string>(path: P, title: string, testId: string, note?: string) =>
  createRoute({ getParentRoute: () => rootRoute, path, component: makeScreen(title, note, testId, true) });

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  validateSearch: (s: Record<string, unknown>): { redirect?: string } => ({ redirect: typeof s.redirect === 'string' ? s.redirect : undefined }),
  component: LoginScreen,
});

const routeTree = rootRoute.addChildren([
  layoutRoute.addChildren([
    tab('/', 'screens.home.title', 'home-screen'),
    tab('/earnings', 'screens.earnings.title', 'earnings-screen'),
    tab('/cash', 'screens.cash.title', 'cash-screen'),
    tab('/history', 'screens.history.title', 'history-screen'),
    tab('/profile', 'screens.profile.title', 'profile-screen'),
  ]),
  loginRoute,
  full('/onboarding', 'screens.onboarding.title', 'onboarding-screen', 'Document status, permissions checklist (location, notifications) and install prompt.'),
  full('/offers/$offerId', 'screens.offer.title', 'offer-screen', 'Offer card with a 45 s countdown, accept/decline, distance and pay estimate.'),
  full('/deliveries/$deliveryId', 'screens.delivery.title', 'delivery-screen', 'Step flow: to restaurant → picked up → at drop → delivered (COD / delivery code per R39).'),
  full('/earnings/$weekId', 'screens.earnings.title', 'earnings-week-screen'),
  full('/settings', 'screens.profile.title', 'settings-screen'),
  full('/help', 'screens.help.title', 'help-screen'),
]);

export const router = createRouter({ routeTree, defaultPreload: 'intent', defaultNotFoundComponent: NotFound, scrollRestoration: true });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

export { Link };
