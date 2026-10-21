import { Construction, ShieldX } from 'lucide-react';
import { type ReactNode } from 'react';
import { useTranslation } from '@rovo/i18n';
import { Button, EmptyState, PageHeader } from '@rovo/ui';

/** 404 screen (every app, `noindex`). */
export function NotFoundScreen({ onHome, homeLabel }: { onHome?: () => void; homeLabel?: ReactNode }) {
  const { t } = useTranslation();
  return (
    <main className="flex min-h-dvh flex-col items-center justify-center bg-bg p-6 text-center" data-testid="page-not-found">
      <p className="text-6xl font-extrabold text-flame-700">404</p>
      <h1 className="mt-4 text-xl font-semibold text-fg">{t('pages.notFound.title')}</h1>
      <p className="mt-2 max-w-sm text-fg-muted">{t('pages.notFound.body')}</p>
      {onHome && (
        <Button className="mt-6" onClick={onHome}>
          {homeLabel ?? t('actions.goHome')}
        </Button>
      )}
    </main>
  );
}

/** 403 / wrong-role screen. */
export function ForbiddenScreen({ action }: { action?: ReactNode }) {
  const { t } = useTranslation();
  return (
    <main className="flex min-h-dvh items-center justify-center bg-bg p-6" data-testid="page-forbidden">
      <EmptyState icon={<ShieldX className="size-10" />} title={t('pages.forbidden.title')} description={t('pages.forbidden.body')} action={action} />
    </main>
  );
}

/**
 * "Wrong app" screen — the user is signed in with a role for another host (doc 17 §6), e.g. a
 * rider opening the restaurant app. `audience` is a key under `auth:wrongApp.*`.
 */
export function WrongAppScreen({ audience, action }: { audience: 'customer' | 'restaurant' | 'rider' | 'admin'; action?: ReactNode }) {
  const { t } = useTranslation('auth');
  return (
    <main className="flex min-h-dvh items-center justify-center bg-bg p-6" data-testid="page-wrong-app">
      <EmptyState
        icon={<ShieldX className="size-10" />}
        title={t('wrongApp.title', { audience: t(`wrongApp.${audience}`) })}
        description={t('wrongApp.body')}
        action={action}
      />
    </main>
  );
}

export interface PlaceholderScreenProps {
  title: ReactNode;
  /** Short note about what this screen will do (from docs 04–07). */
  note?: ReactNode;
  onBack?: () => void;
  /** data-testid for the screen root (area-element convention). */
  testId?: string;
}

/**
 * Placeholder for a screen an app team will build out. Shows the real page header and a
 * "coming soon" body, so the shell, navigation and routing are testable before the screen lands.
 */
export function PlaceholderScreen({ title, note, onBack, testId }: PlaceholderScreenProps) {
  const { t } = useTranslation();
  return (
    <div className="flex min-h-dvh flex-col bg-bg" data-testid={testId}>
      <PageHeader title={title} onBack={onBack} />
      <main className="flex flex-1 flex-col items-center justify-center p-6">
        <EmptyState
          icon={<Construction className="size-10" />}
          title={<span className="inline-flex items-center gap-2">{title}</span>}
          description={
            <>
              <span className="mb-2 inline-block rounded-full bg-flame-50 px-3 py-0.5 text-sm font-semibold text-flame-700">
                {t('placeholder.badge')}
              </span>
              <span className="block">{note ?? t('placeholder.body')}</span>
            </>
          }
        />
      </main>
    </div>
  );
}
