import { forwardRef, type InputHTMLAttributes, type ReactNode, useId } from 'react';
import { cn } from '../cn';

export interface FieldRenderProps {
  id: string;
  'aria-describedby'?: string;
  'aria-invalid'?: boolean;
}

export interface FieldProps {
  label: ReactNode;
  hint?: ReactNode;
  error?: ReactNode;
  /** Visually hide the label (it stays the accessible name). */
  hideLabel?: boolean;
  className?: string;
  id?: string;
  children: (props: FieldRenderProps) => ReactNode;
}

/** Label + hint + error wiring for any control (ids, aria-describedby, aria-invalid). */
export function Field({ label, hint, error, hideLabel, className, id: idProp, children }: FieldProps) {
  const auto = useId();
  const id = idProp ?? `f${auto}`;
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;
  const describedBy = [hintId, errorId].filter(Boolean).join(' ') || undefined;
  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <label htmlFor={id} className={cn('text-sm font-semibold text-fg', hideLabel && 'sr-only')}>
        {label}
      </label>
      {children({ id, 'aria-describedby': describedBy, 'aria-invalid': error ? true : undefined })}
      {hint && (
        <p id={hintId} className="text-sm text-fg-muted">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} className="text-sm font-medium text-danger" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}

export const inputClass =
  'min-h-12 w-full rounded-md border border-border bg-surface px-3 text-base text-fg ' +
  'placeholder:text-fg-muted aria-[invalid=true]:border-danger aria-[invalid=true]:border-2 ' +
  'disabled:opacity-60 focus-visible:outline-3 focus-visible:outline-offset-1 focus-visible:outline-focus';

export interface InputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'id'> {
  label: ReactNode;
  hint?: ReactNode;
  error?: ReactNode;
  hideLabel?: boolean;
  id?: string;
  containerClassName?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, hint, error, hideLabel, id, className, containerClassName, ...rest },
  ref,
) {
  return (
    <Field label={label} hint={hint} error={error} hideLabel={hideLabel} id={id} className={containerClassName}>
      {(field) => <input ref={ref} {...field} {...rest} className={cn(inputClass, className)} />}
    </Field>
  );
});
