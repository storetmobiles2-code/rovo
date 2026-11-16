import { type FormEvent, useState } from 'react';
import { useAdminLogin } from '@rovo/auth';
import { useTranslation } from '@rovo/i18n';
import { Button, Input, OtpInput, useAutoFocus } from '@rovo/ui';
import { BrandLogo } from './BrandLogo';

export interface AdminLoginScreenProps {
  onSuccess: () => void;
}

/** Admin login: email + password → mandatory TOTP step (doc 12). */
export function AdminLoginScreen({ onSuccess }: AdminLoginScreenProps) {
  const { t } = useTranslation(['auth', 'errors']);
  const login = useAdminLogin();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [totp, setTotp] = useState('');
  const focusEmail = useAutoFocus<HTMLInputElement>();
  const focusTotp = useAutoFocus<HTMLInputElement>();

  async function submitCredentials(e: FormEvent) {
    e.preventDefault();
    await login.submitCredentials(email, password).catch(() => {});
  }
  // `code` comes from OtpInput.onComplete: the `totp` state has not re-rendered yet when the
  // last digit lands, so auto-submit must use the value it was handed.
  async function verifyTotp(code: string) {
    const ok = await login.submitTotp({ totp: code }).catch(() => false);
    if (ok) onSuccess();
  }
  function submitTotp(e: FormEvent) {
    e.preventDefault();
    void verifyTotp(totp);
  }

  return (
    <main className="mx-auto flex min-h-dvh w-full max-w-sm flex-col justify-center gap-6 p-6">
      {/* The admin header logo is the white wordmark, so it sits on brand ink (docs/brand). */}
      <div className="mx-auto rounded-lg bg-ink-900 px-5 py-3">
        <BrandLogo height={36} />
      </div>
      {login.step === 'credentials' ? (
        <form onSubmit={submitCredentials} className="flex flex-col gap-4" data-testid="login-credentials-form" noValidate>
          <h1 className="text-xl font-semibold text-fg">{t('auth:admin.title')}</h1>
          <Input
            label={t('auth:admin.email')}
            type="email"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            ref={focusEmail}
          />
          <Input
            label={t('auth:admin.password')}
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={login.credentialsError ? t([`errors:${login.credentialsError.code}`, 'errors:INVALID_CREDENTIALS']) : undefined}
          />
          <Button type="submit" size="lg" fullWidth loading={login.submitting} data-testid="login-submit-credentials">
            {t('auth:admin.submit')}
          </Button>
        </form>
      ) : (
        <form onSubmit={submitTotp} className="flex flex-col gap-4" data-testid="login-totp-form" noValidate>
          <h1 className="text-xl font-semibold text-fg">{t('auth:admin.totpTitle')}</h1>
          <OtpInput
            label={t('auth:admin.totpLabel')}
            hint={t('auth:admin.totpHint')}
            value={totp}
            onValueChange={setTotp}
            onComplete={(code) => void verifyTotp(code)}
            webOtp={false}
            error={login.mfaError ? t([`errors:${login.mfaError.code}`, 'errors:MFA_INVALID']) : undefined}
            ref={focusTotp}
          />
          <Button type="submit" size="lg" fullWidth loading={login.submitting} disabled={totp.length !== 6} data-testid="login-submit-totp">
            {t('auth:admin.totpSubmit')}
          </Button>
        </form>
      )}
    </main>
  );
}
