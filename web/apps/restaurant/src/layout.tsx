import { Link, Outlet, useLocation } from '@tanstack/react-router';
import { ChartColumn, Clock, Inbox, UtensilsCrossed, Wallet } from 'lucide-react';
import { BottomNav, type BottomNavItem, OfflineBanner, useOnline } from '@rovo/ui';
import { useTranslation } from '@rovo/i18n';

const TABS = [
  { key: 'inbox', to: '/inbox', icon: Inbox, labelKey: 'nav.inbox' },
  { key: 'menu', to: '/menu', icon: UtensilsCrossed, labelKey: 'nav.menu' },
  { key: 'hours', to: '/hours', icon: Clock, labelKey: 'nav.hours' },
  { key: 'analytics', to: '/analytics', icon: ChartColumn, labelKey: 'nav.analytics' },
  { key: 'payouts', to: '/payouts', icon: Wallet, labelKey: 'nav.payouts' },
] as const;

export function RootLayout() {
  const online = useOnline();
  const { t } = useTranslation('restaurant');
  const location = useLocation();
  const active = (to: string) => location.pathname.startsWith(to);
  const items: BottomNavItem[] = TABS.map((tab) => ({ key: tab.key, label: t(tab.labelKey), icon: tab.icon, active: active(tab.to) }));
  return (
    <div className="flex min-h-dvh flex-col bg-bg">
      <OfflineBanner online={online} />
      <div className="flex-1">
        <Outlet />
      </div>
      <BottomNav
        items={items}
        renderItem={(item, content) => {
          const tab = TABS.find((x) => x.key === item.key)!;
          return (
            <Link to={tab.to} aria-current={item.active ? 'page' : undefined} data-testid={`nav-${tab.key}`} className={`flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 py-1 ${item.active ? 'text-primary' : 'text-fg-muted'}`}>
              {content}
            </Link>
          );
        }}
      />
    </div>
  );
}
