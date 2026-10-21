import { useNavigate } from '@tanstack/react-router';
import { type ReactNode, useEffect } from 'react';
import { RequireAuth } from '@rovo/auth';
import { SkeletonList } from '@rovo/ui';
import { ForbiddenScreen, WrongAppScreen } from '@rovo/shell';

/** Redirects guests to /login (keeping the return path), shows wrong-app for other roles. */
export function Protected({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  return (
    <RequireAuth
      roles={['CUSTOMER']}
      fallback={<ScreenSkeleton />}
      onGuest={<LoginRedirect />}
      onForbidden={<WrongAppScreen audience="customer" />}
    >
      {children}
    </RequireAuth>
  );
  function LoginRedirect() {
    useEffect(() => {
      const redirect = window.location.pathname + window.location.search;
      void navigate({ to: '/login', search: { redirect }, replace: true });
    }, []);
    return <ScreenSkeleton />;
  }
}

export { ForbiddenScreen };

function ScreenSkeleton() {
  return (
    <div className="p-4">
      <SkeletonList rows={6} />
    </div>
  );
}
