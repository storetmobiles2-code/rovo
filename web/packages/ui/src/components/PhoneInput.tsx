import { forwardRef, type InputHTMLAttributes, type ReactNode } from 'react';
import { useTranslation } from '@rovo/i18n';
import { cn } from '../cn';
import { Field, inputClass } from './Field';

export interface PhoneInputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type' | 'id'> {
  label: ReactNode;
  hint?: ReactNode;
  error?: ReactNode;
  /** 10-digit national number (without +91). */
  value: string;
  onValueChange: (national: string) => void;
  id?: string;
}

/**
 * Indian mobile input with a fixed +91 prefix. Accepts pasted "+91 98765 43210" /
 * "09876543210" and keeps only the 10 national digits. Convert with `normalizeIndianMobile`.
 */
export const PhoneInput = forwardRef<HTMLInputElement, PhoneInputProps>(function PhoneInput(
  { label, hint, error, value, onValueChange, id, className, ...rest },
  ref,
) {
  const { t } = useTranslation('auth');
  return (
    <Field label={label} hint={hint} error={error} id={id}>
      {(field) => (
        <div className="flex items-stretch gap-2">
          <span
            className="inline-flex min-h-12 items-center rounded-md border border-border bg-surface-muted px-3 font-semibold text-fg"
            title={t('phone.countryCode')}
          >
            <span aria-hidden>+91</span>
            <span className="sr-only">{t('phone.countryCode')}</span>
          </span>
          <input
            ref={ref}
            {...field}
            {...rest}
            type="tel"
            inputMode="numeric"
            autoComplete="tel-national"
            maxLength={16}
            value={value}
            onChange={(e) => {
              let digits = e.target.value.replace(/\D/g, '');
              if (digits.length > 10 && digits.startsWith('91')) digits = digits.slice(2);
              if (digits.length > 10 && digits.startsWith('0')) digits = digits.slice(1);
              onValueChange(digits.slice(0, 10));
            }}
            className={cn(inputClass, 'tracking-wide', className)}
          />
        </div>
      )}
    </Field>
  );
});
