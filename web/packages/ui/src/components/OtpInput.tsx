import { forwardRef, type InputHTMLAttributes, type ReactNode, useEffect } from 'react';
import { cn } from '../cn';
import { Field, inputClass } from './Field';

export interface OtpInputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type' | 'id'> {
  label: ReactNode;
  hint?: ReactNode;
  error?: ReactNode;
  value: string;
  onValueChange: (code: string) => void;
  /** Called once all digits are present (typed, pasted or autofilled). */
  onComplete?: (code: string) => void;
  length?: number;
  /** Use the WebOTP API (Android Chrome) to read the code from the SMS. Default true. */
  webOtp?: boolean;
  id?: string;
}

interface OtpCredentialLike {
  code?: string;
}

/**
 * One-time-code input: a single field (best for autofill, paste and screen readers —
 * WCAG 2.2 SC 3.3.8), `autocomplete="one-time-code"`, numeric keypad, plus WebOTP
 * (`navigator.credentials.get({ otp })`) where supported. The SMS must end with
 * `@<host> #<code>` for WebOTP to match (doc 12 §2.2).
 */
export const OtpInput = forwardRef<HTMLInputElement, OtpInputProps>(function OtpInput(
  { label, hint, error, value, onValueChange, onComplete, length = 6, webOtp = true, id, className, disabled, ...rest },
  ref,
) {
  useEffect(() => {
    if (!webOtp || disabled || typeof window === 'undefined' || !('OTPCredential' in window)) return;
    const ac = new AbortController();
    const credentials = navigator.credentials as unknown as {
      get(opts: { otp: { transport: string[] }; signal: AbortSignal }): Promise<OtpCredentialLike | null>;
    };
    credentials
      .get({ otp: { transport: ['sms'] }, signal: ac.signal })
      .then((cred) => {
        const code = cred?.code?.replace(/\D/g, '').slice(0, length);
        if (code) {
          onValueChange(code);
          if (code.length === length) onComplete?.(code);
        }
      })
      .catch(() => {
        /* aborted, unsupported or dismissed — manual entry still works */
      });
    return () => ac.abort();
    // Only (re)arm when the feature toggles; callbacks may change every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [webOtp, disabled, length]);

  return (
    <Field label={label} hint={hint} error={error} id={id}>
      {(field) => (
        <input
          ref={ref}
          {...field}
          {...rest}
          disabled={disabled}
          type="text"
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern={`\\d{${length}}`}
          maxLength={length}
          value={value}
          onChange={(e) => {
            const code = e.target.value.replace(/\D/g, '').slice(0, length);
            onValueChange(code);
            if (code.length === length) onComplete?.(code);
          }}
          className={cn(inputClass, 'text-center font-mono text-2xl tracking-[0.5em]', className)}
        />
      )}
    </Field>
  );
});
