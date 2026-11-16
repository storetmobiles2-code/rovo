import { useNavigate } from '@tanstack/react-router';
import { useTranslation } from '@rovo/i18n';
import { PlaceholderScreen } from '@rovo/shell';
import { Protected } from './guard';

// Screen factories for the route table. Each returns a placeholder screen component until the
// real screen lands; kept apart from router.ts so that file only wires routes.

/** A customer screen that needs login (doc 17 §3.1). */
export function guarded(title: string, note?: string, testId?: string) {
  return function GuardedScreen() {
    const { t } = useTranslation('customer');
    const navigate = useNavigate();
    return (
      <Protected>
        <PlaceholderScreen title={t(title)} note={note} onBack={() => void navigate({ to: '/' })} testId={testId} />
      </Protected>
    );
  };
}

/** A public screen (home, discovery, restaurant, search, cart, legal). */
export function publicScreen(title: string, testId?: string) {
  return function PublicScreen() {
    const { t } = useTranslation('customer');
    return <PlaceholderScreen title={t(title)} testId={testId} />;
  };
}
