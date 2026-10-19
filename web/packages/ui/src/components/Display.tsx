import { Star } from 'lucide-react';
import { type HTMLAttributes, type ReactNode } from 'react';
import { useFormatters, useTranslation } from '@rovo/i18n';
import { type Money as MoneyValue, type Paise } from '@rovo/utils';
import { cn } from '../cn';

// --- Money -----------------------------------------------------------------
export interface MoneyProps extends HTMLAttributes<HTMLSpanElement> {
  value: Paise | MoneyValue;
  /** Always show paise (e.g. ₹149.00). */
  fixed?: boolean;
}

/** Renders paise as ₹ with en-IN / te-IN lakh grouping. Never does float maths. */
export function Money({ value, fixed, className, ...rest }: MoneyProps) {
  const { money } = useFormatters();
  return (
    <span className={cn('tabular-nums', className)} {...rest}>
      {money(value, { fixed })}
    </span>
  );
}

// --- Badge ------------------------------------------------------------------
export type BadgeTone = 'neutral' | 'brand' | 'success' | 'warning' | 'danger' | 'info';

const badgeTones: Record<BadgeTone, string> = {
  neutral: 'bg-surface-muted text-fg',
  brand: 'bg-flame-50 text-flame-700',
  success: 'bg-success-bg text-success',
  warning: 'bg-warning-bg text-warning',
  danger: 'bg-danger-bg text-danger',
  info: 'bg-info-bg text-info',
};

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: BadgeTone;
  icon?: ReactNode;
}

export function Badge({ tone = 'neutral', icon, className, children, ...rest }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-sm font-semibold',
        badgeTones[tone],
        className,
      )}
      {...rest}
    >
      {icon}
      {children}
    </span>
  );
}

// --- VegMarker --------------------------------------------------------------
export type FoodType = 'veg' | 'nonveg' | 'egg';

export interface VegMarkerProps {
  type: FoodType;
  /** px size of the marker square. */
  size?: number;
  /** Hide the accessible label (when text already states veg/non-veg next to it). */
  hideLabel?: boolean;
  className?: string;
}

/**
 * FSSAI food marker. Veg: green square outline + filled green circle. Non-veg: brown square
 * outline + filled brown triangle. Egg: amber square + circle. Meaning is carried by shape and
 * an accessible name, never colour alone (doc 17 §9).
 */
export function VegMarker({ type, size = 16, hideLabel, className }: VegMarkerProps) {
  const { t } = useTranslation();
  const color = type === 'veg' ? 'var(--color-veg)' : type === 'egg' ? 'var(--color-egg)' : 'var(--color-nonveg)';
  const label = type === 'veg' ? t('food.veg') : type === 'egg' ? t('food.egg') : t('food.nonVeg');
  return (
    <span
      role="img"
      aria-label={hideLabel ? undefined : label}
      aria-hidden={hideLabel || undefined}
      title={label}
      className={cn('inline-flex shrink-0', className)}
      style={{ width: size, height: size }}
    >
      <svg viewBox="0 0 16 16" width={size} height={size} focusable="false" aria-hidden>
        <rect x="1" y="1" width="14" height="14" rx="2" fill="none" stroke={color} strokeWidth="1.5" />
        {type === 'nonveg' ? (
          <path d="M8 4.5 L12 11.5 L4 11.5 Z" fill={color} />
        ) : (
          <circle cx="8" cy="8" r="3.5" fill={color} />
        )}
      </svg>
    </span>
  );
}

// --- Rating -----------------------------------------------------------------
export interface RatingProps {
  /** 0–5, may be fractional. */
  value: number;
  count?: number;
  /** Show the "New" badge instead of stars (restaurants with < 5 ratings). */
  isNew?: boolean;
  size?: number;
  className?: string;
}

/** Read-only star rating with an accessible label ("Rated 4.3 out of 5"). */
export function Rating({ value, count, isNew, size = 16, className }: RatingProps) {
  const { t } = useTranslation();
  const { number } = useFormatters();
  if (isNew) return <Badge tone="brand" className={className}>{t('rating.new')}</Badge>;
  const clamped = Math.max(0, Math.min(5, value));
  return (
    <span
      className={cn('inline-flex items-center gap-1', className)}
      role="img"
      aria-label={t('rating.label', { value: clamped.toFixed(1) })}
    >
      <span className="relative inline-flex" aria-hidden style={{ width: size * 5, height: size }}>
        <span className="absolute inset-0 flex text-ink-200">
          {[0, 1, 2, 3, 4].map((i) => (
            <Star key={i} className="fill-current" style={{ width: size, height: size }} />
          ))}
        </span>
        <span className="absolute inset-0 flex overflow-hidden text-flame-500" style={{ width: `${(clamped / 5) * 100}%` }}>
          {[0, 1, 2, 3, 4].map((i) => (
            <Star key={i} className="shrink-0 fill-current" style={{ width: size, height: size }} />
          ))}
        </span>
      </span>
      <span className="text-sm font-semibold text-fg tabular-nums">{clamped.toFixed(1)}</span>
      {count !== undefined && <span className="text-sm text-fg-muted">({number(count)})</span>}
    </span>
  );
}

// --- Card -------------------------------------------------------------------
export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  as?: 'div' | 'article' | 'section' | 'li';
}

export function Card({ as: Tag = 'div', className, children, ...rest }: CardProps) {
  return (
    <Tag className={cn('rounded-lg border border-border-subtle bg-surface p-4 shadow-card', className)} {...rest}>
      {children}
    </Tag>
  );
}
