import { useLocation, useNavigate } from '@tanstack/react-router';
import { type ReactNode, useEffect, useState } from 'react';
import { RequireAuth } from '@rovo/auth';
import { SkeletonList } from '@rovo/ui';
import { WrongAppScreen } from '@rovo/shell';

function Loading() {
  return (
    <div className="p-4">
      <SkeletonList rows={6} />
    </div>
  );
}

function LoginRedirect() {
  const navigate = useNavigate();
  const { href } = useLocation();
  // Captured once: the page the user asked for, before this redirect changes the location.
  const [redirect] = useState(href);
  useEffect(() => {
    void navigate({ to: '/login', search: { redirect }, replace: true });
  }, [navigate, redirect]);
  return <Loading />;
}

/** Rider app admits delivery partners only (doc 17 §6). */
export function Protected({ children }: { children: ReactNode }) {
  return (
    <RequireAuth roles={['RIDER']} fallback={<Loading />} onGuest={<LoginRedirect />} onForbidden={<WrongAppScreen audience="rider" />}>
      {children}
    </RequireAuth>
  );
}
