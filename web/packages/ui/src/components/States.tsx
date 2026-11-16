import { Inbox, RefreshCw, TriangleAlert, WifiOff } from 'lucide-react';
import { type HTMLAttributes, type ReactNode } from 'react';
import { useTranslation } from '@rovo/i18n';
import { isApiError } from '@rovo/api-client';
import { cn } from '../cn';
import { Button } from './Button';

// --- Skeleton ---------------------------------------------------------------
export interface SkeletonProps extends HTMLAttributes<HTMLDivElement> {
  /** CSS width/height, e.g. "100%", "2rem". */
  width?: string | number;
  height?: string | number;
  rounded?: boolean;
}

/** Content-shaped placeholder; respects prefers-reduced-motion (animation disabled in theme.css). */
export function Skeleton({ width, height, rounded, className, style, ...rest }: SkeletonProps) {
  return (
    <div
      aria-hidden
      className={cn('animate-skeleton bg-ink-100', rounded ? 'rounded-full' : 'rounded-md', className)}
      style={{ width, height, ...style }}
      {...rest}
    />
  );
}

/** A screen-reader-announced loading region wrapping skeletons. */
export function SkeletonList({ rows = 5, className }: { rows?: number; className?: string }) {
  const { t } = useTranslation();
  return (
    <div className={cn('flex flex-col gap-3', className)} aria-busy="true" aria-label={t('status.loading')}>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="flex gap-3">
          <Skeleton width={64} height={64} />
          <div className="flex flex-1 flex-col gap-2 py-1">
            <Skeleton width="60%" height={16} />
            <Skeleton width="40%" height={14} />
          </div>
        </div>
      ))}
    </div>
  );
}

// --- EmptyState -------------------------------------------------------------
export interface EmptyStateProps {
  icon?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center gap-3 px-6 py-12 text-center', className)}>
      <span aria-hidden className="text-fg-muted [&_svg]:size-10">
        {icon ?? <Inbox className="size-10" />}
      </span>
      <h2 className="text-lg font-semibold text-fg">{title}</h2>
      {description && <p className="max-w-sm text-fg-muted">{description}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}

// --- ErrorState -------------------------------------------------------------
export interface ErrorStateProps {
  /** Any thrown value; `ApiError`s render their translated `code`. */
  error?: unknown;
  title?: ReactNode;
  onRetry?: () => void;
  /** Secondary action, e.g. "Go home". */
  secondaryAction?: ReactNode;
  className?: string;
}

/**
 * problem+json-aware error panel. For an `ApiError` it shows the message for `errors:<code>`
 * and the trace id; network errors get the offline icon. Server text is never shown raw.
 */
export function ErrorState({ error, title, onRetry, secondaryAction, className }: ErrorStateProps) {
  const { t } = useTranslation(['common', 'errors']);
  const api = isApiError(error) ? error : undefined;
  const isNetwork = api?.isNetworkError ?? false;
  const message = api
    ? t([`errors:${api.code}`, 'errors:generic'])
    : t('pages.error.body');
  return (
    <div role="alert" className={cn('flex flex-col items-center gap-3 px-6 py-12 text-center', className)}>
      <span aria-hidden className="text-danger">
        {isNetwork ? <WifiOff className="size-10" /> : <TriangleAlert className="size-10" />}
      </span>
      <h2 className="text-lg font-semibold text-fg">{title ?? t('pages.error.title')}</h2>
      <p className="max-w-sm text-fg-muted">{message}</p>
      {api?.traceId && (
        <p className="text-xs text-fg-muted">{t('errors:reference', { id: api.traceId })}</p>
      )}
      <div className="mt-2 flex flex-wrap items-center justify-center gap-2">
        {onRetry && (
          <Button variant="primary" onClick={onRetry} leadingIcon={<RefreshCw aria-hidden className="size-5" />}>
            {t('actions.retry')}
          </Button>
        )}
        {secondaryAction}
      </div>
    </div>
  );
}
