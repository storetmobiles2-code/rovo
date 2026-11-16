import { useNavigate } from '@tanstack/react-router';
import { useTranslation } from '@rovo/i18n';
import { PlaceholderScreen } from '@rovo/shell';
import { Protected } from './guard';

// Screen factories for the route table. Each returns a placeholder screen component until the
// real screen lands; kept apart from router.ts so that file only wires routes.

export function makeScreen(title: string, note?: string, testId?: string, withBack = true) {
  return function Screen() {
    const { t } = useTranslation('rider');
    const navigate = useNavigate();
    return (
      <Protected>
        <PlaceholderScreen title={t(title)} note={note} testId={testId} onBack={withBack ? () => void navigate({ to: '/' }) : undefined} />
      </Protected>
    );
  };
}
