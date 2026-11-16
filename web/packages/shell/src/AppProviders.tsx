import { QueryClientProvider } from '@tanstack/react-query';
import { type ReactNode, useEffect, useRef, useState } from 'react';
import { type AppId, createApiClient, type ApiClient } from '@rovo/api-client';
import { ApiClientProvider } from '@rovo/api-client/react';
import { SessionProvider } from '@rovo/auth';
import { i18n, initI18n, type Namespace } from '@rovo/i18n';
import { Toaster } from '@rovo/ui';
import { createQueryClient } from './query';
import { readRuntimeConfig } from './runtime-config';

export interface AppProvidersProps {
  app: AppId;
  version: string;
  /** i18n namespaces this app loads (common + errors are always included). */
  namespaces: Namespace[];
  children: ReactNode;
  /** Allow the en-XA pseudo-locale (dev/staging). Defaults to import.meta.env.DEV. */
  allowPseudo?: boolean;
}

/**
 * Root providers for every rovo app: i18n (awaited so no untranslated flash), TanStack Query,
 * the API client (credentials, refresh, headers), session, and the toast viewport.
 */
export function AppProviders({ app, version, namespaces, children, allowPseudo }: AppProvidersProps) {
  const [ready, setReady] = useState(() => i18n.isInitialized);
  const queryClientRef = useRef(createQueryClient());
  const apiClientRef = useRef<ApiClient | null>(null);
  if (!apiClientRef.current) {
    const config = readRuntimeConfig();
    apiClientRef.current = createApiClient({
      app,
      version,
      baseUrl: config.apiBase,
      getLanguage: () => i18n.language,
    });
  }

  useEffect(() => {
    let active = true;
    void initI18n({ namespaces, allowPseudo: allowPseudo ?? import.meta.env.DEV }).then(() => {
      if (active) setReady(true);
    });
    return () => {
      active = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!ready) {
    return (
      <div aria-busy="true" className="flex min-h-dvh items-center justify-center bg-bg text-fg-muted">
        <span className="sr-only">Loading</span>
      </div>
    );
  }

  return (
    <QueryClientProvider client={queryClientRef.current}>
      <ApiClientProvider client={apiClientRef.current}>
        <SessionProvider>
          {children}
          <Toaster />
        </SessionProvider>
      </ApiClientProvider>
    </QueryClientProvider>
  );
}
