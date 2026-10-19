import { useMutation, useQuery, useQueryClient, type UseMutationResult } from '@tanstack/react-query';
import { createContext, type ReactNode, useContext, useEffect, useMemo } from 'react';
import { type ApiError, type Me, qk, unwrap } from '@rovo/api-client';
import { useApiClient } from '@rovo/api-client/react';

export type SessionStatus = 'loading' | 'authenticated' | 'guest';

export interface SessionValue {
  me: Me | null;
  status: SessionStatus;
  isLoading: boolean;
  error: ApiError | null;
  refetch: () => void;
  logout: UseMutationResult<void, ApiError, void>;
}

const SessionContext = createContext<SessionValue | null>(null);

/**
 * Loads `GET /me` and exposes the session. On refresh failure the api-client's session-expired
 * callback clears the `me` query, which flips this to `guest` and triggers route guards.
 */
export function SessionProvider({ children }: { children: ReactNode }) {
  const client = useApiClient();
  const queryClient = useQueryClient();

  const query = useQuery<Me, ApiError>({
    queryKey: qk.me(),
    queryFn: () => unwrap(client.api.GET('/me')),
    retry: false,
    staleTime: 60_000,
  });

  useEffect(() => {
    client.setOnSessionExpired(() => {
      queryClient.setQueryData(qk.me(), null);
      queryClient.removeQueries({ predicate: (q) => q.queryKey[0] !== 'client-config' });
    });
    return () => client.setOnSessionExpired(undefined);
  }, [client, queryClient]);

  const logout = useMutation<void, ApiError, void>({
    mutationFn: async () => {
      await unwrap(client.api.POST('/auth/logout'));
    },
    onSettled: () => {
      queryClient.setQueryData(qk.me(), null);
      queryClient.clear();
    },
  });

  const me = (query.data ?? null) as Me | null;
  const isAuthError = query.error?.status === 401;

  const value = useMemo<SessionValue>(
    () => ({
      me,
      status: query.isLoading ? 'loading' : me ? 'authenticated' : 'guest',
      isLoading: query.isLoading,
      error: isAuthError ? null : (query.error ?? null),
      refetch: () => void query.refetch(),
      logout,
    }),
    [me, query.isLoading, query.error, isAuthError, logout],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionValue {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error('useSession must be used inside <SessionProvider>');
  return ctx;
}

export function useMe(): Me | null {
  return useSession().me;
}
