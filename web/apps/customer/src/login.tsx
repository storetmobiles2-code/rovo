import { useNavigate, useSearch } from '@tanstack/react-router';
import { PhoneOtpLoginScreen } from '@rovo/shell';
import { safeRedirect } from '@rovo/utils';

/** Customer phone + OTP login; returns to the `redirect` route on success. */
export function LoginScreen() {
  const navigate = useNavigate();
  const { redirect } = useSearch({ strict: false }) as { redirect?: string };
  return <PhoneOtpLoginScreen onSuccess={() => void navigate({ to: safeRedirect(redirect), replace: true })} />;
}
