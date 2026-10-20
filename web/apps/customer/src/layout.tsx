import { Link, Outlet, useLocation } from '@tanstack/react-router';
import { House, Receipt, Search, ShoppingBag, User } from 'lucide-react';
import { BottomNav, type BottomNavItem, OfflineBanner, useOnline } from '@rovo/ui';
import { useTranslation } from '@rovo/i18n';

const TABS = [
  { key: 'home', to: '/', icon: House, labelKey: 'nav.home' },
  { key: 'search', to: '/search', icon: Search, labelKey: 'nav.search' },
  { key: 'cart', to: '/cart', icon: ShoppingBag, labelKey: 'nav.cart' },
  { key: 'orders', to: '/orders', icon: Receipt, labelKey: 'nav.orders' },
  { key: 'account', to: '/account', icon: User, labelKey: 'nav.account' },
] as const;

/** Root layout: offline banner, routed content and the bottom tab bar. */
export function RootLayout() {
  const online = useOnline();
  const { t } = useTranslation('customer');
  const location = useLocation();
  const active = (to: string) => (to === '/' ? location.pathname === '/' : location.pathname.startsWith(to));

  const items: BottomNavItem[] = TABS.map((tab) => ({
    key: tab.key,
    label: t(tab.labelKey),
    icon: tab.icon,
    active: active(tab.to),
  }));

  return (
    <div className="mx-auto flex min-h-dvh w-full max-w-screen-sm flex-col bg-bg">
      <OfflineBanner online={online} />
      <div className="flex-1">
        <Outlet />
      </div>
      <BottomNav
        items={items}
        renderItem={(item, content) => {
          const tab = TABS.find((x) => x.key === item.key)!;
          return (
            <Link
              to={tab.to}
              aria-current={item.active ? 'page' : undefined}
              data-testid={`nav-${tab.key}`}
              className={`flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 py-1 ${item.active ? 'text-primary' : 'text-fg-muted'}`}
            >
              {content}
            </Link>
          );
        }}
      />
    </div>
  );
}
