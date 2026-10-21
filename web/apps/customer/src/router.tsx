import {
  createRootRoute,
  createRoute,
  createRouter,
  Link,
  Outlet,
} from '@tanstack/react-router';
import { type ReactNode } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useTranslation } from '@rovo/i18n';
import { NotFoundScreen, PlaceholderScreen } from '@rovo/shell';
import { Protected } from './guard';
import { RootLayout } from './layout';
import { LoginScreen } from './login';

const rootRoute = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: NotFound,
});

function NotFound() {
  const navigate = useNavigate();
  return <NotFoundScreen onHome={() => void navigate({ to: '/' })} />;
}

/** A customer screen that needs login (doc 17 §3.1). */
function guarded(title: string, note?: string, testId?: string) {
  return function GuardedScreen() {
    const { t } = useTranslation('customer');
    const navigate = useNavigate();
    return (
      <Protected>
        <PlaceholderScreen title={t(title)} note={note} onBack={() => void navigate({ to: '/' })} testId={testId} />
      </Protected>
    );
  };
}

/** A public screen (home, discovery, restaurant, search, cart, legal). */
function publicScreen(title: string, testId?: string, withNav = true): () => ReactNode {
  return function PublicScreen() {
    const { t } = useTranslation('customer');
    return <PlaceholderScreen title={t(title)} testId={testId} onBack={withNav ? undefined : undefined} />;
  };
}

// Layout route hosts the tabbed, public-shell pages.
const layoutRoute = createRoute({ getParentRoute: () => rootRoute, id: 'layout', component: RootLayout });

const indexRoute = createRoute({ getParentRoute: () => layoutRoute, path: '/', component: publicScreen('screens.home.title', 'home-screen') });
const cityRoute = createRoute({ getParentRoute: () => layoutRoute, path: '/$city', component: publicScreen('screens.city.title', 'city-screen') });
const restaurantRoute = createRoute({ getParentRoute: () => layoutRoute, path: '/$city/r/$restaurantSlug', component: publicScreen('screens.restaurant.title', 'restaurant-screen') });
const searchRoute = createRoute({ getParentRoute: () => layoutRoute, path: '/search', component: publicScreen('screens.search.title', 'search-screen') });
const cartRoute = createRoute({ getParentRoute: () => layoutRoute, path: '/cart', component: publicScreen('screens.cart.title', 'cart-screen') });
const ordersRoute = createRoute({ getParentRoute: () => layoutRoute, path: '/orders', component: guarded('screens.orders.title', undefined, 'orders-screen') });
const accountRoute = createRoute({ getParentRoute: () => layoutRoute, path: '/account', component: guarded('screens.account.title', undefined, 'account-screen') });

// Full-screen routes (no bottom nav).
const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  validateSearch: (s: Record<string, unknown>): { redirect?: string } => ({ redirect: typeof s.redirect === 'string' ? s.redirect : undefined }),
  component: LoginScreen,
});
const checkoutRoute = createRoute({ getParentRoute: () => rootRoute, path: '/checkout', component: guarded('screens.checkout.title', undefined, 'checkout-screen') });
const payRoute = createRoute({ getParentRoute: () => rootRoute, path: '/checkout/pay/$orderId', component: guarded('screens.payment.title', undefined, 'payment-screen') });
const orderDetailRoute = createRoute({ getParentRoute: () => rootRoute, path: '/orders/$orderId', component: guarded('screens.orderTracking.title', undefined, 'order-tracking-screen') });
const orderRateRoute = createRoute({ getParentRoute: () => rootRoute, path: '/orders/$orderId/rate', component: guarded('screens.orderRate.title', undefined, 'order-rate-screen') });
const orderHelpRoute = createRoute({ getParentRoute: () => rootRoute, path: '/orders/$orderId/help', component: guarded('screens.orderHelp.title', undefined, 'order-help-screen') });
const addressesRoute = createRoute({ getParentRoute: () => rootRoute, path: '/account/addresses', component: guarded('screens.addresses.title', undefined, 'addresses-screen') });
const addressNewRoute = createRoute({ getParentRoute: () => rootRoute, path: '/account/addresses/new', component: guarded('screens.addressNew.title', 'Map pin-drop loads as a lazy MapLibre chunk.', 'address-new-screen') });
const addressEditRoute = createRoute({ getParentRoute: () => rootRoute, path: '/account/addresses/$addressId', component: guarded('screens.addressEdit.title', undefined, 'address-edit-screen') });
const privacyRoute = createRoute({ getParentRoute: () => rootRoute, path: '/account/privacy', component: guarded('screens.privacy.title', undefined, 'privacy-screen') });
const legalRoute = createRoute({ getParentRoute: () => rootRoute, path: '/legal/$doc', component: publicScreen('screens.legal.title', 'legal-screen') });

const routeTree = rootRoute.addChildren([
  layoutRoute.addChildren([indexRoute, cityRoute, restaurantRoute, searchRoute, cartRoute, ordersRoute, accountRoute]),
  loginRoute,
  checkoutRoute,
  payRoute,
  orderDetailRoute,
  orderRateRoute,
  orderHelpRoute,
  addressesRoute,
  addressNewRoute,
  addressEditRoute,
  privacyRoute,
  legalRoute,
]);

export const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
  defaultNotFoundComponent: NotFound,
  scrollRestoration: true,
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

export { Link };
