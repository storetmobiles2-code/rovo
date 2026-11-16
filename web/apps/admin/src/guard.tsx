import { useLocation, useNavigate } from '@tanstack/react-router';
import { type ReactNode, useEffect, useState } from 'react';
import { RequireAuth, ROLES_BY_APP, type RoleName, userRoles, useSession } from '@rovo/auth';
import { ForbiddenScreen, WrongAppScreen } from '@rovo/shell';
import { SkeletonList } from '@rovo/ui';
import { canOpenScreen } from './screens';

function Loading() {
  return (
    <div className="p-6">
      <SkeletonList rows={6} />
    </div>
  );
}

function LoginRedirect() {
  const navigate = useNavigate();
  const { href } = useLocation();
  // Captured once: the page the admin asked for, before this redirect changes the location.
  const [redirect] = useState(href);
  useEffect(() => {
    void navigate({ to: '/login', search: redirect === '/' ? {} : { redirect }, replace: true });
  }, [navigate, redirect]);
  return <Loading />;
}

/** Admits only admin identities (doc 17 §6); guests go to /login, other roles see "wrong app". */
export function Protected({ children }: { children: ReactNode }) {
  return (
    <RequireAuth roles={ROLES_BY_APP.admin} fallback={<Loading />} onGuest={<LoginRedirect />} onForbidden={<WrongAppScreen audience="admin" />}>
      {children}
    </RequireAuth>
  );
}

/** Per-screen role check inside the console (UX only; the API enforces RBAC). */
export function RequireScreenRole({ roles, children }: { roles: readonly RoleName[]; children: ReactNode }) {
  const { me } = useSession();
  if (!canOpenScreen(roles, userRoles(me))) return <ForbiddenScreen />;
  return <>{children}</>;
}
