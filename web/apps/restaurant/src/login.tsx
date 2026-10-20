import { useNavigate, useSearch } from '@tanstack/react-router';
import { PhoneOtpLoginScreen } from '@rovo/shell';

export function LoginScreen() {
  const navigate = useNavigate();
  const { redirect } = useSearch({ strict: false }) as { redirect?: string };
  return <PhoneOtpLoginScreen onSuccess={() => void navigate({ to: redirect ?? '/inbox', replace: true })} />;
}
