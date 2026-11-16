import { Link, Outlet, useLocation, useNavigate } from '@tanstack/react-router';
import {
  BadgeIndianRupee,
  Bike,
  ChartColumn,
  ClipboardCheck,
  FileClock,
  Headset,
  LayoutDashboard,
  LogOut,
  type LucideIcon,
  MapPinned,
  Radio,
  Receipt,
  Route,
  Store,
  Ticket,
  Users,
  Wallet,
} from 'lucide-react';
import { userRoles, useSession } from '@rovo/auth';
import { useTranslation } from '@rovo/i18n';
import { BrandLogo } from '@rovo/shell';
import { Badge, Button, OfflineBanner, useOnline } from '@rovo/ui';
import { Protected } from './guard';
import { ADMIN_SCREENS, canOpenScreen } from './screens';

const ICONS: Record<string, LucideIcon> = {
  dashboard: LayoutDashboard,
  live: Radio,
  orders: Receipt,
  dispatch: Route,
  restaurants: Store,
  riders: Bike,
  customers: Users,
  coupons: Ticket,
  approvals: ClipboardCheck,
  payouts: Wallet,
  zones: MapPinned,
  pricing: BadgeIndianRupee,
  tickets: Headset,
  auditLog: FileClock,
  reports: ChartColumn,
};

type NavScreen = Extract<(typeof ADMIN_SCREENS)[number], { nav: string }>;
const NAV = ADMIN_SCREENS.filter((s): s is NavScreen => 'nav' in s);

function isActive(path: string, pathname: string): boolean {
  return path === '/' ? pathname === '/' : pathname === path || pathname.startsWith(`${path}/`);
}

/**
 * Console frame: sidebar (a scrollable strip below 1024 px, so the live board stays usable on
 * a phone — doc 07 §1), top bar with the signed-in admin, and the routed screen.
 */
function ConsoleFrame() {
  const { t } = useTranslation(['admin', 'common']);
  const { me, logout } = useSession();
  const online = useOnline();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const mine = userRoles(me);
  const items = NAV.filter((s) => canOpenScreen(s.roles, mine));

  function signOut() {
    logout.mutate(undefined, { onSettled: () => void navigate({ to: '/login', replace: true }) });
  }

  return (
    <div className="flex min-h-dvh flex-col bg-bg lg:flex-row" data-testid="admin-shell">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:start-2 focus:top-2 focus:z-50 focus:rounded-md focus:bg-surface focus:px-3 focus:py-2 focus:text-link"
      >
        {t('common:actions.skipToContent')}
      </a>
      <aside className="border-b border-border-subtle bg-surface lg:sticky lg:top-0 lg:h-dvh lg:w-60 lg:shrink-0 lg:overflow-y-auto lg:border-b-0 lg:border-e">
        {/* White wordmark on brand ink (docs/brand: admin uses the monochrome mark on ink-900). */}
        <div className="mb-2 flex items-center gap-3 bg-ink-900 px-4 py-3">
          <BrandLogo height={28} />
          <span className="text-sm font-semibold text-white">{t('admin:shell.product')}</span>
        </div>
        <nav aria-label={t('common:nav.label')}>
          <ul className="flex gap-1 overflow-x-auto px-2 pb-2 lg:flex-col lg:overflow-visible">
            {items.map((item) => {
              const Icon = ICONS[item.nav] ?? LayoutDashboard;
              const active = isActive(item.path, pathname);
              return (
                <li key={item.path} className="shrink-0">
                  <Link
                    to={item.path}
                    aria-current={active ? 'page' : undefined}
                    data-testid={`nav-${item.nav}`}
                    className={`flex min-h-11 items-center gap-3 rounded-md px-3 py-2 text-sm font-medium whitespace-nowrap ${
                      active ? 'bg-flame-50 text-flame-700' : 'text-fg hover:bg-surface-muted'
                    }`}
                  >
                    <Icon aria-hidden className="size-5 shrink-0" />
                    {t(`admin:nav.${item.nav}`)}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <OfflineBanner online={online} />
        <header className="flex min-h-14 flex-wrap items-center justify-end gap-3 border-b border-border-subtle bg-surface px-4 py-2">
          <div className="flex min-w-0 items-center gap-2 text-sm" data-testid="admin-user">
            <span className="truncate font-medium text-fg">{me?.fullName ?? me?.email}</span>
            {mine.map((role) => (
              <Badge key={role} tone="neutral">
                {t(`admin:roles.${role}`, { defaultValue: role })}
              </Badge>
            ))}
          </div>
          <Button variant="secondary" size="sm" onClick={signOut} loading={logout.isPending} leadingIcon={<LogOut aria-hidden className="size-4" />} data-testid="logout-button">
            {t('common:actions.logout')}
          </Button>
        </header>
        <main id="main" tabIndex={-1} className="flex-1 p-4 outline-none lg:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

/** Layout route component: everything inside the console needs an admin session. */
export function RootLayout() {
  return (
    <Protected>
      <ConsoleFrame />
    </Protected>
  );
}
