import type { Me, RoleName } from '@rovo/api-client';
import type { AppId } from '@rovo/api-client';

export type { RoleName } from '@rovo/api-client';

export const ROLES_BY_APP: Record<AppId, readonly RoleName[]> = {
  customer: ['CUSTOMER'],
  restaurant: ['RESTAURANT_OWNER', 'RESTAURANT_STAFF'],
  rider: ['RIDER'],
  admin: ['ADMIN_SUPER', 'ADMIN_OPS', 'ADMIN_SUPPORT', 'ADMIN_FINANCE'],
};

export function userRoles(me: Me | null | undefined): RoleName[] {
  return me?.roles.map((r) => r.role) ?? [];
}

export function hasAnyRole(me: Me | null | undefined, roles: readonly RoleName[]): boolean {
  if (!roles.length) return true;
  const mine = new Set(userRoles(me));
  return roles.some((r) => mine.has(r));
}

/** Whether this user may use this app at all (each host admits only its own roles, doc 17 §6). */
export function canUseApp(me: Me | null | undefined, app: AppId): boolean {
  return hasAnyRole(me, ROLES_BY_APP[app]);
}

/** The audience key used for the "wrong app" message when a user opens the wrong host. */
export function appAudienceKey(app: AppId): 'customer' | 'restaurant' | 'rider' | 'admin' {
  return app;
}
