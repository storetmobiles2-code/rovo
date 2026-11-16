import { useNavigate } from '@tanstack/react-router';
import { NotFoundScreen } from '@rovo/shell';

/** 404 for unknown paths, with a way back to the app's home. */
export function NotFound() {
  const navigate = useNavigate();
  return <NotFoundScreen onHome={() => void navigate({ to: '/inbox' })} />;
}
