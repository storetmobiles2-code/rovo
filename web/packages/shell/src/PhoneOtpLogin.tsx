import { type FormEvent, useState } from 'react';
import { usePhoneOtp } from '@rovo/auth';
import { useTranslation } from '@rovo/i18n';
import { isApiError } from '@rovo/api-client';
import { Button, LanguageSwitcher, OtpInput, PhoneInput } from '@rovo/ui';
import { BrandLogo } from './BrandLogo';

export interface PhoneOtpLoginScreenProps {
  /** Called after the OTP is verified and the session cookie is set. */
  onSuccess: () => void;
  /** Show the language switcher (customer/partner apps). */
  showLanguage?: boolean;
}

/** Shared phone + OTP login (customer, restaurant, rider). Wires @rovo/auth's usePhoneOtp. */
export function PhoneOtpLoginScreen({ onSuccess, showLanguage = true }: PhoneOtpLoginScreenProps) {
  const { t } = useTranslation(['auth', 'errors']);
  const otp = usePhoneOtp();
  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [localError, setLocalError] = useState<string | null>(null);

  const phoneError = (err: unknown): string | undefined => {
    if (isApiError(err) && err.fieldErrors[0]) return t('errors:field.invalidPhone');
    if (err && typeof err === 'object' && 'code' in err) return t('errors:field.invalidPhone');
    return undefined;
  };

  async function submitPhone(e: FormEvent) {
    e.preventDefault();
    setLocalError(null);
    try {
      await otp.requestOtp(phone);
    } catch (err) {
      setLocalError(phoneError(err) ?? t('errors:generic'));
    }
  }

  async function verify() {
    const ok = await otp.verify(code).catch(() => false);
    if (ok) onSuccess();
  }

  return (
    <main className="mx-auto flex min-h-dvh w-full max-w-screen-sm flex-col justify-center gap-6 p-6">
      <div className="flex items-center justify-between">
        <BrandLogo height={36} />
        {showLanguage && <LanguageSwitcher />}
      </div>
      {otp.step === 'phone' ? (
        <form onSubmit={submitPhone} className="flex flex-col gap-4" data-testid="login-phone-form" noValidate>
          <h1 className="text-xl font-semibold text-fg">{t('auth:phone.title')}</h1>
          <PhoneInput
            label={t('auth:phone.label')}
            hint={t('auth:phone.hint')}
            value={phone}
            onValueChange={setPhone}
            error={localError ?? phoneError(otp.requestError)}
            autoFocus
          />
          <Button type="submit" size="lg" fullWidth loading={otp.requesting} disabled={phone.length !== 10} data-testid="login-request-otp">
            {t('auth:phone.submit')}
          </Button>
        </form>
      ) : (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void verify();
          }}
          className="flex flex-col gap-4"
          data-testid="login-otp-form"
          noValidate
        >
          <h1 className="text-xl font-semibold text-fg">{t('auth:otp.title')}</h1>
          <OtpInput
            label={t('auth:otp.label')}
            hint={t('auth:otp.sentTo', { phone: otp.challenge?.phone ?? '' })}
            value={code}
            onValueChange={setCode}
            onComplete={() => void verify()}
            error={otp.verifyError ? t([`errors:${otp.verifyError.code}`, 'errors:generic']) : undefined}
            autoFocus
          />
          <Button type="submit" size="lg" fullWidth loading={otp.verifying} disabled={code.length !== 6} data-testid="login-verify-otp">
            {t('auth:otp.submit')}
          </Button>
          <div className="flex items-center justify-between text-sm">
            <button type="button" className="min-h-11 font-semibold text-link" onClick={otp.reset} data-testid="login-change-number">
              {t('auth:otp.changeNumber')}
            </button>
            {otp.resendIn > 0 && otp.challenge ? (
              <span className="text-fg-muted">{t('auth:otp.resendIn', { count: otp.resendIn })}</span>
            ) : (
              <button type="button" className="min-h-11 font-semibold text-link" onClick={() => void otp.resend()} data-testid="login-resend-otp">
                {t('auth:otp.resend')}
              </button>
            )}
          </div>
        </form>
      )}
    </main>
  );
}
