import { ChevronLeft, Languages, WifiOff } from 'lucide-react';
import { type ComponentType, type ReactNode } from 'react';
import { type Language, useLanguage, useTranslation } from '@rovo/i18n';
import { cn } from '../cn';
import { IconButton } from './Button';

// --- PageHeader -------------------------------------------------------------
export interface PageHeaderProps {
  title: ReactNode;
  /** Back button handler (omit for top-level pages). */
  onBack?: () => void;
  /** Right-aligned actions. */
  actions?: ReactNode;
  /** Sticky at the top of the scroll container. */
  sticky?: boolean;
  className?: string;
}

export function PageHeader({ title, onBack, actions, sticky = true, className }: PageHeaderProps) {
  const { t } = useTranslation();
  return (
    <header
      className={cn(
        'z-20 flex min-h-14 items-center gap-2 border-b border-border-subtle bg-surface px-2 py-2',
        sticky && 'sticky top-0',
        className,
      )}
    >
      {onBack && (
        <IconButton label={t('actions.back')} onClick={onBack}>
          <ChevronLeft />
        </IconButton>
      )}
      <h1 className={cn('flex-1 truncate text-lg font-semibold text-fg', !onBack && 'ps-2')}>{title}</h1>
      {actions && <div className="flex items-center gap-1">{actions}</div>}
    </header>
  );
}

// --- BottomNav --------------------------------------------------------------
export interface BottomNavItem {
  key: string;
  label: ReactNode;
  icon: ComponentType<{ className?: string; 'aria-hidden'?: boolean }>;
  /** Marks the active tab. */
  active?: boolean;
  onSelect?: () => void;
  href?: string;
  /** Unread count bubble. */
  badge?: number;
}

export interface BottomNavProps {
  items: BottomNavItem[];
  'aria-label'?: string;
  className?: string;
  /** Render each item (router Link). Defaults to a button. */
  renderItem?: (item: BottomNavItem, content: ReactNode) => ReactNode;
}

/** Fixed bottom tab bar (customer/rider); 48 px targets, safe-area padding, active marked. */
export function BottomNav({ items, 'aria-label': ariaLabel, className, renderItem }: BottomNavProps) {
  const { t } = useTranslation();
  return (
    <nav
      aria-label={ariaLabel ?? t('nav.label')}
      className={cn(
        'sticky bottom-0 z-20 flex border-t border-border-subtle bg-surface pb-[env(safe-area-inset-bottom)]',
        className,
      )}
    >
      {items.map((item) => {
        const Icon = item.icon;
        const content = (
          <>
            <span className="relative">
              <Icon aria-hidden className="size-6" />
              {item.badge ? (
                <span className="absolute -end-2 -top-1 inline-flex min-w-4 items-center justify-center rounded-full bg-danger-700 px-1 text-[10px] font-bold text-white">
                  {item.badge > 99 ? '99+' : item.badge}
                </span>
              ) : null}
            </span>
            <span className="text-xs font-medium">{item.label}</span>
          </>
        );
        const className2 = cn(
          'flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 py-1',
          item.active ? 'text-primary' : 'text-fg-muted',
        );
        if (renderItem) return <span key={item.key}>{renderItem(item, content)}</span>;
        return (
          <button
            key={item.key}
            type="button"
            onClick={item.onSelect}
            aria-current={item.active ? 'page' : undefined}
            className={className2}
          >
            {content}
          </button>
        );
      })}
    </nav>
  );
}

// --- OfflineBanner ----------------------------------------------------------
export interface OfflineBannerProps {
  /** Pass the result of a `useOnline()` hook. */
  online: boolean;
  className?: string;
}

/** Top banner shown while the browser is offline (polite live region). */
export function OfflineBanner({ online, className }: OfflineBannerProps) {
  const { t } = useTranslation();
  if (online) return null;
  return (
    <div
      role="status"
      aria-live="polite"
      className={cn('flex items-center justify-center gap-2 bg-warning-bg px-4 py-2 text-sm font-medium text-warning', className)}
    >
      <WifiOff aria-hidden className="size-4" />
      {t('status.offline')}
    </div>
  );
}

// --- LanguageSwitcher -------------------------------------------------------
export interface LanguageSwitcherProps {
  /** Languages offered (default en + te). */
  languages?: Language[];
  variant?: 'inline' | 'icon';
  className?: string;
}

/** EN / తెలుగు toggle. Persists the choice and updates `<html lang>` (doc 17 §10). */
export function LanguageSwitcher({ languages = ['en', 'te'], variant = 'inline', className }: LanguageSwitcherProps) {
  const { t } = useTranslation();
  const [current, setLanguage] = useLanguage();
  const next = languages[(languages.indexOf(current) + 1) % languages.length] ?? 'en';

  if (variant === 'icon') {
    return (
      <IconButton label={`${t('language.label')}: ${t(`language.${next}`)}`} onClick={() => void setLanguage(next)} className={className}>
        <Languages />
      </IconButton>
    );
  }
  return (
    <div role="group" aria-label={t('language.label')} className={cn('inline-flex rounded-full border border-border p-0.5', className)}>
      {languages.map((lang) => (
        <button
          key={lang}
          type="button"
          onClick={() => void setLanguage(lang)}
          aria-pressed={current === lang}
          lang={lang === 'te' ? 'te' : 'en'}
          className={cn(
            'min-h-9 rounded-full px-3 text-sm font-semibold',
            current === lang ? 'bg-primary text-on-primary' : 'text-fg',
          )}
        >
          {t(`language.${lang}`)}
        </button>
      ))}
    </div>
  );
}
