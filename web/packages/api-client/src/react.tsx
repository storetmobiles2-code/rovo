import { type QueryClient, type QueryKey, useQueryClient } from '@tanstack/react-query';
import { createContext, type ReactNode, useCallback, useContext, useEffect, useState, useSyncExternalStore } from 'react';
import { newIdempotencyKey } from '@rovo/utils';
import type { ApiClient } from './client';
import { createQueryInvalidator, defaultInvalidationMap, type InvalidationMap, type RealtimeEvent } from './realtime';
import { createEventStream, type EventSourceFactory, type StreamStatus } from './sse';

const ApiClientContext = createContext<ApiClient | null>(null);

export function ApiClientProvider({ client, children }: { client: ApiClient; children: ReactNode }) {
  return <ApiClientContext.Provider value={client}>{children}</ApiClientContext.Provider>;
}

export function useApiClient(): ApiClient {
  const client = useContext(ApiClientContext);
  if (!client) throw new Error('useApiClient must be used inside <ApiClientProvider>');
  return client;
}

// --- Realtime health store (drives polling fallback and the connection indicator) ---
let healthStatus: StreamStatus = 'idle';
const healthListeners = new Set<() => void>();

export const realtimeHealth = {
  get: (): StreamStatus => healthStatus,
  set(s: StreamStatus) {
    healthStatus = s;
    for (const l of healthListeners) l();
  },
  subscribe(l: () => void) {
    healthListeners.add(l);
    return () => healthListeners.delete(l);
  },
};

export function useRealtimeHealth(): StreamStatus {
  return useSyncExternalStore(realtimeHealth.subscribe, realtimeHealth.get, realtimeHealth.get);
}

/**
 * Polling fallback (doc 17 §5.2): `refetchInterval: usePollingInterval(10_000)` polls only
 * while the SSE stream is not open.
 */
export function usePollingInterval(intervalMs: number): number | false {
  const status = useRealtimeHealth();
  return status === 'open' ? false : intervalMs;
}

export interface UseEventStreamOptions {
  topics: string[];
  enabled?: boolean;
  /** Extra/overriding event → keys entries merged over the default map. */
  invalidations?: InvalidationMap;
  /** Keys refetched after every (re)connect (no replay buffer, R10). */
  refetchOnConnect?: QueryKey[];
  onEvent?: (name: string, event: RealtimeEvent) => void;
  closeWhenHiddenAfterMs?: number;
  createEventSource?: EventSourceFactory;
}

/** One SSE stream per tab; invalidates TanStack Query keys on events. */
export function useEventStream(opts: UseEventStreamOptions): StreamStatus {
  const client = useApiClient();
  const queryClient = useQueryClient();
  const status = useRealtimeHealth();
  const { enabled = true, topics, invalidations, refetchOnConnect, onEvent, closeWhenHiddenAfterMs, createEventSource } = opts;
  const topicKey = topics.join(',');

  useEffect(() => {
    if (!enabled) return;
    const map = { ...defaultInvalidationMap, ...invalidations };
    const stream = createEventStream({
      url: `${client.baseUrl}/stream`,
      topics: topicKey ? topicKey.split(',') : [],
      handlers: createQueryInvalidator(queryClient, map, onEvent),
      onStatusChange: realtimeHealth.set,
      onConnect: () => refetchActive(queryClient, refetchOnConnect),
      refresh: () => client.refresher.refresh(),
      onSessionExpired: () => client.expireSession(),
      closeWhenHiddenAfterMs,
      createEventSource,
    });
    stream.start();
    return () => {
      stream.stop();
      realtimeHealth.set('idle');
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- restart only when topics/enabled change
  }, [enabled, topicKey, client, queryClient]);

  return status;
}

function refetchActive(queryClient: QueryClient, keys: QueryKey[] | undefined) {
  for (const queryKey of keys ?? []) void queryClient.invalidateQueries({ queryKey });
}

/**
 * Stable `Idempotency-Key` per user intent: keep it across retries of the same action and
 * call `reset()` after success (or when the user changes the intent).
 */
export function useIdempotencyKey(): { key: string; reset: () => void } {
  const [key, setKey] = useState(newIdempotencyKey);
  const reset = useCallback(() => setKey(newIdempotencyKey()), []);
  return { key, reset };
}
