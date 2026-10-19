import type { RefreshResult } from './refresh';

/**
 * Server-Sent Events client over `EventSource('/api/v1/stream?topics=…')` (doc 17 §5, R10).
 *
 * - Same-origin, cookies only — never a token in the URL.
 * - Owns reconnection (native auto-retry is cancelled): first retry after a random 2–10 s,
 *   then 4 s, 8 s … capped at 30 s, each ±30 % jitter. Three failures within 60 s switch the
 *   status to `polling` (consumers enable `refetchInterval`) while SSE keeps retrying.
 * - `event: reauth` (session revoked/expired) → single-flight refresh, then reconnect; an
 *   expired session stops the stream and calls `onSessionExpired`.
 * - A connection that fails before opening may be a 401 (EventSource hides the status), so
 *   the first such failure refreshes before reconnecting.
 * - No `Last-Event-ID`/replay: `onConnect` runs after every (re)connect so the app refetches
 *   its active queries.
 * - `online` and `visibilitychange → visible` reconnect immediately.
 */
export type StreamStatus = 'idle' | 'connecting' | 'open' | 'reconnecting' | 'polling' | 'paused' | 'closed';

export interface StreamEventLike {
  data: string;
  type: string;
  lastEventId?: string;
}

/** Minimal EventSource surface, so tests and React Native can inject an implementation. */
export interface EventSourceLike {
  readonly readyState: number;
  addEventListener(type: string, listener: (ev: StreamEventLike) => void): void;
  close(): void;
  onopen: ((ev: unknown) => void) | null;
  onerror: ((ev: unknown) => void) | null;
}

export type EventSourceFactory = (url: string) => EventSourceLike;

export type StreamHandler = (data: unknown, raw: StreamEventLike) => void;

export interface EventStreamOptions {
  topics: string[];
  /** Handlers by SSE event name, e.g. `{ 'order.status': … }`. */
  handlers: Record<string, StreamHandler>;
  url?: string;
  onStatusChange?: (status: StreamStatus) => void;
  /** After every successful (re)connect: refetch the active order/inbox/offer queries. */
  onConnect?: () => void;
  refresh?: () => Promise<RefreshResult>;
  onSessionExpired?: () => void;
  /** Close after the tab has been hidden this long (customer: 60 s). Default: keep open. */
  closeWhenHiddenAfterMs?: number;
  createEventSource?: EventSourceFactory;
  random?: () => number;
  now?: () => number;
}

export interface EventStream {
  start(): void;
  stop(): void;
  reconnectNow(): void;
  readonly status: StreamStatus;
}

const CLOSED = 2;
const MAX_DELAY_MS = 30_000;
const POLLING_WINDOW_MS = 60_000;
const POLLING_FAILURES = 3;
const POLLING_RETRY_MS = 60_000;

/** Delay before reconnect attempt `failures` (1-based). Exported for tests. */
export function backoffDelay(failures: number, random: () => number = Math.random): number {
  if (failures <= 1) return 2_000 + Math.round(random() * 8_000);
  const base = Math.min(MAX_DELAY_MS, 2_000 * 2 ** (failures - 1));
  const jitter = 1 + (random() * 0.6 - 0.3);
  return Math.min(MAX_DELAY_MS, Math.round(base * jitter));
}

export function streamUrl(url: string, topics: string[]): string {
  if (!topics.length) return url;
  const sep = url.includes('?') ? '&' : '?';
  return `${url}${sep}topics=${encodeURIComponent(topics.join(','))}`;
}

export function createEventStream(opts: EventStreamOptions): EventStream {
  const url = streamUrl(opts.url ?? '/api/v1/stream', opts.topics);
  const random = opts.random ?? Math.random;
  const now = opts.now ?? (() => Date.now());
  const factory: EventSourceFactory =
    opts.createEventSource ?? ((u) => new EventSource(u) as unknown as EventSourceLike);

  let es: EventSourceLike | null = null;
  let status: StreamStatus = 'idle';
  let running = false;
  let retryTimer: ReturnType<typeof setTimeout> | null = null;
  let hiddenTimer: ReturnType<typeof setTimeout> | null = null;
  let failures = 0;
  let failureTimes: number[] = [];
  let openedThisAttempt = false;
  let refreshedForThisOutage = false;

  const setStatus = (s: StreamStatus) => {
    if (s === status) return;
    status = s;
    opts.onStatusChange?.(s);
  };

  const clearRetry = () => {
    if (retryTimer) clearTimeout(retryTimer);
    retryTimer = null;
  };

  const teardown = () => {
    if (es) {
      es.onopen = null;
      es.onerror = null;
      es.close();
      es = null;
    }
  };

  const connect = () => {
    clearRetry();
    teardown();
    if (!running) return;
    openedThisAttempt = false;
    setStatus(failures === 0 ? 'connecting' : status === 'polling' ? 'polling' : 'reconnecting');
    const source = factory(url);
    es = source;
    source.onopen = () => {
      if (es !== source) return;
      openedThisAttempt = true;
      failures = 0;
      failureTimes = [];
      refreshedForThisOutage = false;
      setStatus('open');
      opts.onConnect?.();
    };
    source.onerror = () => {
      if (es !== source) return;
      void handleFailure();
    };
    source.addEventListener('reauth', () => {
      if (es !== source) return;
      void reauth();
    });
    for (const [name, handler] of Object.entries(opts.handlers)) {
      source.addEventListener(name, (ev) => {
        if (es !== source) return;
        let data: unknown = ev.data;
        try {
          data = JSON.parse(ev.data);
        } catch {
          /* non-JSON payloads are passed through as strings */
        }
        handler(data, ev);
      });
    }
  };

  const schedule = (delay: number) => {
    clearRetry();
    retryTimer = setTimeout(connect, delay);
  };

  const handleFailure = async () => {
    const neverOpened = !openedThisAttempt;
    teardown();
    if (!running) return;
    failures += 1;
    const t = now();
    failureTimes = [...failureTimes.filter((x) => t - x < POLLING_WINDOW_MS), t];
    const polling = failureTimes.length >= POLLING_FAILURES;
    setStatus(polling ? 'polling' : 'reconnecting');

    if (neverOpened && opts.refresh && !refreshedForThisOutage) {
      refreshedForThisOutage = true;
      const result = await opts.refresh();
      if (!running) return;
      if (result === 'expired') {
        expire();
        return;
      }
    }
    schedule(polling ? POLLING_RETRY_MS : backoffDelay(failures, random));
  };

  const reauth = async () => {
    teardown();
    setStatus('reconnecting');
    if (!opts.refresh) {
      schedule(backoffDelay(1, random));
      return;
    }
    const result = await opts.refresh();
    if (!running) return;
    if (result === 'expired') expire();
    else if (result === 'ok') connect();
    else schedule(backoffDelay(Math.max(1, failures + 1), random));
  };

  const expire = () => {
    stopInternal();
    opts.onSessionExpired?.();
  };

  const onOnline = () => {
    if (running && status !== 'open') reconnectNow();
  };

  const onVisibility = () => {
    if (typeof document === 'undefined') return;
    if (document.visibilityState === 'hidden') {
      if (opts.closeWhenHiddenAfterMs !== undefined && !hiddenTimer) {
        hiddenTimer = setTimeout(() => {
          hiddenTimer = null;
          clearRetry();
          teardown();
          setStatus('paused');
        }, opts.closeWhenHiddenAfterMs);
      }
      return;
    }
    if (hiddenTimer) clearTimeout(hiddenTimer);
    hiddenTimer = null;
    if (running && status !== 'open') reconnectNow();
  };

  const stopInternal = () => {
    running = false;
    clearRetry();
    if (hiddenTimer) clearTimeout(hiddenTimer);
    hiddenTimer = null;
    teardown();
    if (typeof window !== 'undefined') window.removeEventListener('online', onOnline);
    if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibility);
    setStatus('closed');
  };

  function reconnectNow() {
    if (!running) return;
    failures = Math.min(failures, 1);
    refreshedForThisOutage = false;
    connect();
  }

  return {
    start() {
      if (running) return;
      running = true;
      failures = 0;
      failureTimes = [];
      if (typeof window !== 'undefined') window.addEventListener('online', onOnline);
      if (typeof document !== 'undefined') document.addEventListener('visibilitychange', onVisibility);
      connect();
    },
    stop: stopInternal,
    reconnectNow,
    get status() {
      return status;
    },
  };
}

export const EVENT_SOURCE_CLOSED = CLOSED;
