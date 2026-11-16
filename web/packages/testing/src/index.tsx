import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, type RenderOptions } from '@testing-library/react';
import axe, { type Result, type RunOptions } from 'axe-core';
import type { ReactElement, ReactNode } from 'react';
import { expect } from 'vitest';
import { type ApiClient, type AppId, createApiClient } from '@rovo/api-client';
import { ApiClientProvider } from '@rovo/api-client/react';

export * from '@testing-library/react';
export { default as userEvent } from '@testing-library/user-event';
export { FakeEventSource } from '@rovo/api-client/testing';

export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity, staleTime: 0 },
      mutations: { retry: false },
    },
  });
}

export interface ProviderOptions {
  app?: AppId;
  queryClient?: QueryClient;
  apiClient?: ApiClient;
}

export function createTestProviders(opts: ProviderOptions = {}) {
  const queryClient = opts.queryClient ?? createTestQueryClient();
  const apiClient = opts.apiClient ?? createApiClient({ app: opts.app ?? 'customer', version: '0.0.0-test' });
  function Providers({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ApiClientProvider client={apiClient}>{children}</ApiClientProvider>
      </QueryClientProvider>
    );
  }
  return { Providers, queryClient, apiClient };
}

/** Renders inside QueryClient + ApiClient providers (i18n is initialised by the setup file). */
export function renderWithProviders(
  ui: ReactElement,
  opts: ProviderOptions & Omit<RenderOptions, 'wrapper'> = {},
) {
  const { Providers, queryClient, apiClient } = createTestProviders(opts);
  return { ...render(ui, { wrapper: Providers, ...opts }), queryClient, apiClient };
}

const DEFAULT_AXE: RunOptions = {
  // jsdom has no layout engine, so contrast is verified by the token contrast test instead.
  rules: { 'color-contrast': { enabled: false } },
  resultTypes: ['violations'],
};

export async function axeViolations(container: Element = document.body, options: RunOptions = {}): Promise<Result[]> {
  const results = await axe.run(container, { ...DEFAULT_AXE, ...options, rules: { ...DEFAULT_AXE.rules, ...options.rules } });
  return results.violations;
}

/** Fails with a readable list when axe finds any violation. */
export async function expectNoAxeViolations(container: Element = document.body, options?: RunOptions) {
  const violations = await axeViolations(container, options);
  expect(
    violations.map((v) => `${v.id} (${v.impact}): ${v.help} → ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`),
  ).toEqual([]);
}
