import { RouterProvider } from '@tanstack/react-router';
import { AppProviders, ErrorBoundary } from '@rovo/shell';
import { router } from './router';

const version = import.meta.env.VITE_APP_VERSION ?? '0.0.0';

export function App() {
  return (
    <ErrorBoundary>
      {/* Admin V1 is English-first (doc 17 §10); the admin namespace keeps Telugu for completeness. */}
      <AppProviders app="admin" version={version} namespaces={['auth', 'admin']} allowPseudo={false}>
        <RouterProvider router={router} />
      </AppProviders>
    </ErrorBoundary>
  );
}
