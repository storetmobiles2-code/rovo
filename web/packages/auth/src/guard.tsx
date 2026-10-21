import { type ReactNode } from 'react';
import type { AppId, RoleName } from '@rovo/api-client';
import { hasAnyRole } from './roles';
import { useSession } from './session';

export type GuardOutcome =
  | { state: 'loading' }
  | { state: 'guest' }
  | { state: 'wrong-role' }
  | { state: 'allowed' };

/** Pure guard decision — reused by TanStack Router `beforeLoad` and by the component below. */
export function evaluateGuard(
  me: Parameters<typeof hasAnyRole>[0],
  status: 'loading' | 'authenticated' | 'guest',
  roles: readonly RoleName[] = [],
): GuardOutcome {
  if (status === 'loading') return { state: 'loading' };
  if (status === 'guest' || !me) return { state: 'guest' };
  if (!hasAnyRole(me, roles)) return { state: 'wrong-role' };
  return { state: 'allowed' };
}

export interface RequireAuthProps {
  /** Allowed roles; empty means any authenticated user of this app. */
  roles?: readonly RoleName[];
  children: ReactNode;
  /** Rendered while the session loads. */
  fallback?: ReactNode;
  /** Rendered for guests (login redirect). */
  onGuest?: ReactNode;
  /** Rendered when the user is signed in but lacks the role (403 / wrong-app). */
  onForbidden?: ReactNode;
}

/**
 * Client-side route guard (UX only — the server stays the authority, doc 17 §3.4). Frontend
 * app teams wire `onGuest`/`onForbidden` to their router (redirect to /login, show 403).
 */
export function RequireAuth({ roles = [], children, fallback = null, onGuest = null, onForbidden = null }: RequireAuthProps) {
  const { me, status } = useSession();
  const outcome = evaluateGuard(me, status, roles);
  switch (outcome.state) {
    case 'loading':
      return <>{fallback}</>;
    case 'guest':
      return <>{onGuest}</>;
    case 'wrong-role':
      return <>{onForbidden}</>;
    case 'allowed':
      return <>{children}</>;
  }
}

/** Convenience: guard against an entire app's role set (the "wrong app" case). */
export function roleSetFor(app: AppId, map: Record<AppId, readonly RoleName[]>): readonly RoleName[] {
  return map[app];
}
