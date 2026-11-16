import { useNavigate, useSearch } from '@tanstack/react-router';
import { useEffect } from 'react';
import { canUseApp, useSession } from '@rovo/auth';
import { AdminLoginScreen } from '@rovo/shell';
import { safeRedirect } from '@rovo/utils';

/** Admin email + password → TOTP login (A-00). */
export function LoginScreen() {
  const navigate = useNavigate();
  const { redirect } = useSearch({ strict: false }) as { redirect?: string };
  const { me, status } = useSession();
  const signedIn = status === 'authenticated' && canUseApp(me, 'admin');
  const target = safeRedirect(redirect);

  // Already signed in (or just finished TOTP): leave the login page.
  useEffect(() => {
    if (signedIn) void navigate({ to: target, replace: true });
  }, [signedIn, target, navigate]);

  return <AdminLoginScreen onSuccess={() => void navigate({ to: target, replace: true })} />;
}
