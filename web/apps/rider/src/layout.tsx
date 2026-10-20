import { Link, Outlet, useLocation } from '@tanstack/react-router';
import { Banknote, History, House, User, Wallet } from 'lucide-react';
import { BottomNav, type BottomNavItem, OfflineBanner, useOnline } from '@rovo/ui';
import { useTranslation } from '@rovo/i18n';

const TABS = [
  { key: 'home', to: '/', icon: House, labelKey: 'nav.home' },
  { key: 'earnings', to: '/earnings', icon: Wallet, labelKey: 'nav.earnings' },
  { key: 'cash', to: '/cash', icon: Banknote, labelKey: 'nav.cash' },
  { key: 'history', to: '/history', icon: History, labelKey: 'nav.history' },
  { key: 'profile', to: '/profile', icon: User, labelKey: 'nav.profile' },
] as const;

export function RootLayout() {
  const online = useOnline();
  const { t } = useTranslation('rider');
  const location = useLocation();
  const active = (to: string) => (to === '/' ? location.pathname === '/' : location.pathname.startsWith(to));
  const items: BottomNavItem[] = TABS.map((tab) => ({ key: tab.key, label: t(tab.labelKey), icon: tab.icon, active: active(tab.to) }));
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
            <Link to={tab.to} aria-current={item.active ? 'page' : undefined} data-testid={`nav-${tab.key}`} className={`flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 py-1 ${item.active ? 'text-primary' : 'text-fg-muted'}`}>
              {content}
            </Link>
          );
        }}
      />
    </div>
  );
}
