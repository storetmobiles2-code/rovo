import { QueryClient } from '@tanstack/react-query';
import { isApiError } from '@rovo/api-client';

/** Query defaults from doc 17 §4.2. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        gcTime: 10 * 60_000,
        refetchOnWindowFocus: true,
        networkMode: 'online',
        retry: (failureCount, error) => {
          // Never retry auth/permission/business errors; retry transient ones up to twice.
          if (isApiError(error) && [400, 401, 403, 404, 409, 422].includes(error.status)) return false;
          return failureCount < 2;
        },
        retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 8000) + Math.random() * 300,
      },
      mutations: { retry: 0 },
    },
  });
}
