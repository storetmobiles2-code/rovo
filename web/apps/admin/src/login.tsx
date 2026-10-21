import { useNavigate, useSearch } from '@tanstack/react-router';
import { AdminLoginScreen } from '@rovo/shell';

/** Admin email + password → TOTP login. */
export function LoginScreen() {
  const navigate = useNavigate();
  const { redirect } = useSearch({ strict: false }) as { redirect?: string };
  return <AdminLoginScreen onSuccess={() => void navigate({ to: redirect ?? '/', replace: true })} />;
}
