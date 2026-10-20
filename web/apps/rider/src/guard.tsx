import { useNavigate } from '@tanstack/react-router';
import { type ReactNode, useEffect } from 'react';
import { RequireAuth } from '@rovo/auth';
import { SkeletonList } from '@rovo/ui';
import { WrongAppScreen } from '@rovo/shell';

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
    <RequireAuth roles={['RIDER']} fallback={<Loading />} onGuest={<LoginRedirect />} onForbidden={<WrongAppScreen audience="rider" />}>
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
