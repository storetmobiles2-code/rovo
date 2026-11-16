import { Slot } from '@radix-ui/react-slot';
import { LoaderCircle } from 'lucide-react';
import { type ButtonHTMLAttributes, forwardRef, type ReactNode } from 'react';
import { cn } from '../cn';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'accent';
export type ButtonSize = 'sm' | 'md' | 'lg';

const base =
  'inline-flex items-center justify-center gap-2 rounded-md font-semibold select-none ' +
  'transition-colors duration-(--duration-fast) disabled:cursor-not-allowed disabled:opacity-60 ' +
  'focus-visible:outline-3 focus-visible:outline-offset-2 focus-visible:outline-focus';

const variants: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-on-primary hover:bg-primary-hover',
  secondary: 'border border-border bg-surface text-fg hover:bg-surface-muted',
  ghost: 'bg-transparent text-fg hover:bg-surface-muted',
  danger: 'bg-danger-700 text-white hover:bg-danger-700/90',
  accent: 'bg-flame-500 text-ink-900 hover:bg-flame-400',
};

// Every size keeps a ≥ 44 px touch target (doc 17 §13.1).
const sizes: Record<ButtonSize, string> = {
  sm: 'min-h-11 px-3 text-sm',
  md: 'min-h-12 px-4 text-base',
  lg: 'min-h-14 px-6 text-lg',
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Shows a spinner, sets aria-busy and disables the button (double-tap protection). */
  loading?: boolean;
  fullWidth?: boolean;
  leadingIcon?: ReactNode;
  /** Render as the child element (e.g. a router Link) with button styling. */
  asChild?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'primary', size = 'md', loading = false, fullWidth, leadingIcon, asChild, className, children, disabled, type, ...rest },
  ref,
) {
  const classes = cn(base, variants[variant], sizes[size], fullWidth && 'w-full', className);
  if (asChild) {
    return (
      <Slot ref={ref} className={classes} {...rest}>
        {children}
      </Slot>
    );
  }
  return (
    <button
      ref={ref}
      type={type ?? 'button'}
      className={classes}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {loading ? <LoaderCircle aria-hidden className="size-5 animate-spin motion-reduce:animate-none" /> : leadingIcon}
      {children}
    </button>
  );
});

export interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'aria-label'> {
  /** Accessible name — required because the button has no visible text. */
  label: string;
  variant?: Exclude<ButtonVariant, 'accent'>;
  children: ReactNode;
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { label, variant = 'ghost', className, children, type, ...rest },
  ref,
) {
  return (
    <button
      ref={ref}
      type={type ?? 'button'}
      aria-label={label}
      title={label}
      className={cn(base, variants[variant], 'size-12 shrink-0 rounded-full p-0 [&_svg]:size-6', className)}
      {...rest}
    >
      {children}
    </button>
  );
});
