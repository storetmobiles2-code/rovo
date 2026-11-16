import { createRootRoute, createRoute, createRouter, Outlet, type RouterHistory, useNavigate } from '@tanstack/react-router';
import type { RoleName } from '@rovo/auth';
import { NotFoundScreen } from '@rovo/shell';
import { RootLayout } from './layout';
import { LoginScreen } from './login';
import { AdminScreenSkeleton } from './placeholder';
import { ADMIN_SCREENS, type AdminScreen } from './screens';

function NotFound() {
  const navigate = useNavigate();
  return <NotFoundScreen onHome={() => void navigate({ to: '/' })} />;
}

const rootRoute = createRootRoute({ component: () => <Outlet />, notFoundComponent: NotFound });

// Everything under the layout route needs an admin session (guard in RootLayout).
const layoutRoute = createRoute({ getParentRoute: () => rootRoute, id: 'console', component: RootLayout });

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  validateSearch: (s: Record<string, unknown>): { redirect?: string } => (typeof s.redirect === 'string' ? { redirect: s.redirect } : {}),
  component: LoginScreen,
});

function screenRoute<const S extends AdminScreen>(screen: S) {
  return createRoute({
    getParentRoute: () => layoutRoute,
    path: screen.path as S['path'],
    staticData: { roles: screen.roles },
    component: function Screen() {
      return <AdminScreenSkeleton screen={screen} />;
    },
  });
}

type ScreenRoutes = { [K in keyof typeof ADMIN_SCREENS]: ReturnType<typeof screenRoute<(typeof ADMIN_SCREENS)[K]>> };
const screenRoutes = ADMIN_SCREENS.map((screen) => screenRoute(screen)) as unknown as ScreenRoutes;

const routeTree = rootRoute.addChildren([layoutRoute.addChildren(screenRoutes), loginRoute]);

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
