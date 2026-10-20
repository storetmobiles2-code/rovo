import { useNavigate } from '@tanstack/react-router';
import { type ReactNode, useEffect } from 'react';
import { RequireAuth } from '@rovo/auth';
import { SkeletonList } from '@rovo/ui';
import { WrongAppScreen } from '@rovo/shell';

/** Restaurant app admits owners and staff (doc 17 §6). */
export function Protected({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  function LoginRedirect() {
    useEffect(() => {
      const redirect = window.location.pathname + window.location.search;
      void navigate({ to: '/login', search: { redirect }, replace: true });
    }, []);
    return <Loading />;
  }
  return (
    <RequireAuth
      roles={['RESTAURANT_OWNER', 'RESTAURANT_STAFF']}
      fallback={<Loading />}
      onGuest={<LoginRedirect />}
      onForbidden={<WrongAppScreen audience="restaurant" />}
    >
      {children}
    </RequireAuth>
  );
}

function Loading() {
  return (
    <div className="p-4">
      <SkeletonList rows={6} />
    </div>
  );
}
