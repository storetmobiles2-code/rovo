import { Construction } from 'lucide-react';
import { useTranslation } from '@rovo/i18n';
import { RequireScreenRole } from './guard';
import type { AdminScreen } from './screens';

/**
 * Route skeleton for an admin screen: real title, role guard and test id; the body says what
 * will land here (doc 07 §15). Admin-only, so it lives in this app, not in `packages/ui`.
 */
export function AdminScreenSkeleton({ screen }: { screen: AdminScreen }) {
  const { t } = useTranslation(['admin', 'common']);
  return (
    <RequireScreenRole roles={screen.roles}>
      <section data-testid={screen.testId} aria-labelledby="screen-title" className="mx-auto flex w-full max-w-5xl flex-col gap-4">
        <h1 id="screen-title" className="text-2xl font-semibold text-fg">
          {t(`admin:screens.${screen.screen}.title`)}
        </h1>
        <div className="flex flex-col items-start gap-3 rounded-lg border border-border-subtle bg-surface p-6">
          <Construction aria-hidden className="size-8 text-fg-muted" />
          <span className="inline-block rounded-full bg-flame-50 px-3 py-0.5 text-sm font-semibold text-flame-700">{t('common:placeholder.badge')}</span>
          <p className="text-fg-muted">{t(`admin:screens.${screen.screen}.summary`)}</p>
          <p className="text-sm text-fg-muted">{t('admin:shell.screenIds', { ids: screen.ids })}</p>
        </div>
      </section>
    </RequireScreenRole>
  );
}
