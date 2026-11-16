import { RouterProvider } from '@tanstack/react-router';
import { AppProviders, ErrorBoundary } from '@rovo/shell';
import { router } from './router';

const version = import.meta.env.VITE_APP_VERSION ?? '0.0.0';

export function App() {
  return (
    <ErrorBoundary>
      <AppProviders app="restaurant" version={version} namespaces={['auth', 'restaurant']}>
        <RouterProvider router={router} />
      </AppProviders>
    </ErrorBoundary>
  );
}
