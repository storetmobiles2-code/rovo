import type { RoleName } from '@rovo/auth';

/**
 * Admin route table (doc 07 §15 screen inventory, doc 17 §3.4 routes and default roles).
 * One source for the router (`staticData.roles`), the per-route guard and the sidebar, so the
 * nav never offers a screen the guard would refuse. The server remains the authority.
 */
export interface AdminScreen {
  /** TanStack Router path. */
  path: string;
  /** Key under `admin:screens.*` (title). */
  screen: string;
  /** Key under `admin:nav.*`; only top-level screens appear in the sidebar. */
  nav?: string;
  /** Roles admitted in addition to ADMIN_SUPER (who sees everything). Empty = every admin. */
  roles: readonly RoleName[];
  /** Screen IDs from doc 07 §15, shown on the placeholder for traceability. */
  ids: string;
  testId: string;
}

const OPS = 'ADMIN_OPS';
const SUPPORT = 'ADMIN_SUPPORT';
const FINANCE = 'ADMIN_FINANCE';

export const ADMIN_SCREENS = [
  { path: '/', screen: 'dashboard', nav: 'dashboard', roles: [], ids: 'A-01', testId: 'dashboard-screen' },
  { path: '/live', screen: 'live', nav: 'live', roles: [OPS, SUPPORT], ids: 'A-02', testId: 'live-board-screen' },
  { path: '/orders', screen: 'orders', nav: 'orders', roles: [OPS, SUPPORT], ids: 'A-02', testId: 'orders-screen' },
  { path: '/orders/$orderId', screen: 'orderDetail', roles: [OPS, SUPPORT], ids: 'A-03, A-05', testId: 'order-detail-screen' },
  { path: '/dispatch', screen: 'dispatch', nav: 'dispatch', roles: [OPS], ids: 'A-04', testId: 'dispatch-screen' },
  { path: '/restaurants', screen: 'restaurants', nav: 'restaurants', roles: [OPS, FINANCE], ids: 'A-20', testId: 'restaurants-screen' },
  { path: '/restaurants/$restaurantId', screen: 'restaurantDetail', roles: [OPS, FINANCE], ids: 'A-21 – A-28', testId: 'restaurant-detail-screen' },
  { path: '/riders', screen: 'riders', nav: 'riders', roles: [OPS, FINANCE], ids: 'A-30', testId: 'riders-screen' },
  { path: '/riders/$riderId', screen: 'riderDetail', roles: [OPS, FINANCE], ids: 'A-31 – A-35', testId: 'rider-detail-screen' },
  { path: '/customers', screen: 'customers', nav: 'customers', roles: [SUPPORT], ids: 'A-10', testId: 'customers-screen' },
  { path: '/customers/$customerId', screen: 'customerDetail', roles: [SUPPORT], ids: 'A-11', testId: 'customer-detail-screen' },
  { path: '/coupons', screen: 'coupons', nav: 'coupons', roles: [OPS, FINANCE], ids: 'A-50, A-51', testId: 'coupons-screen' },
  { path: '/approvals', screen: 'approvals', nav: 'approvals', roles: [FINANCE], ids: 'A-60', testId: 'approvals-screen' },
  { path: '/finance/payouts', screen: 'payouts', nav: 'payouts', roles: [FINANCE], ids: 'A-80, A-82, A-83', testId: 'payouts-screen' },
  { path: '/finance/payouts/$batchId', screen: 'payoutDetail', roles: [FINANCE], ids: 'A-81', testId: 'payout-detail-screen' },
  { path: '/zones', screen: 'zones', nav: 'zones', roles: [OPS], ids: 'A-40 – A-42, A-44', testId: 'zones-screen' },
  { path: '/pricing', screen: 'pricing', nav: 'pricing', roles: [OPS, FINANCE], ids: 'A-43', testId: 'pricing-screen' },
  { path: '/support/tickets', screen: 'tickets', nav: 'tickets', roles: [SUPPORT], ids: 'A-70', testId: 'tickets-screen' },
  { path: '/support/tickets/$ticketId', screen: 'ticketDetail', roles: [SUPPORT], ids: 'A-71', testId: 'ticket-detail-screen' },
  { path: '/audit-log', screen: 'auditLog', nav: 'auditLog', roles: ['ADMIN_SUPER'], ids: 'A-95', testId: 'audit-log-screen' },
  { path: '/reports', screen: 'reports', nav: 'reports', roles: [OPS, FINANCE], ids: 'A-90', testId: 'reports-screen' },
] as const satisfies readonly AdminScreen[];

export type AdminPath = (typeof ADMIN_SCREENS)[number]['path'];

/** Every admin role may open a screen with no role list; ADMIN_SUPER may open all of them. */
export function canOpenScreen(roles: readonly RoleName[], mine: readonly RoleName[]): boolean {
  if (mine.includes('ADMIN_SUPER')) return true;
  if (roles.length === 0) return true;
  return roles.some((role) => mine.includes(role));
}
