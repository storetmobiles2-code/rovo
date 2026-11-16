import { QueryClient } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { qk } from './keys';
import { createQueryInvalidator } from './realtime';
import { backoffDelay, createEventStream, type StreamStatus, streamUrl } from './sse';
import { FakeEventSource } from './testing/fake-event-source';

beforeEach(() => {
  FakeEventSource.reset();
  vi.useFakeTimers();
});
afterEach(() => {
  vi.useRealTimers();
});

const mid = () => 0.5; // deterministic jitter: first retry 6 s, then exact base delays

describe('backoffDelay', () => {
  it('waits 2–10 s first, then doubles from 4 s up to 30 s with ±30 % jitter', () => {
    expect(backoffDelay(1, () => 0)).toBe(2_000);
    expect(backoffDelay(1, () => 1)).toBe(10_000);
    expect(backoffDelay(2, mid)).toBe(4_000);
    expect(backoffDelay(3, mid)).toBe(8_000);
    expect(backoffDelay(10, mid)).toBe(30_000);
    expect(backoffDelay(3, () => 0)).toBe(5_600);
    expect(backoffDelay(3, () => 1)).toBe(10_400);
  });

  it('encodes topics into the URL', () => {
    expect(streamUrl('/api/v1/stream', ['order:1', 'order:2'])).toBe('/api/v1/stream?topics=order%3A1%2Corder%3A2');
  });
});

describe('createEventStream', () => {
  it('connects, dispatches parsed events and runs onConnect', () => {
    const handler = vi.fn();
    const onConnect = vi.fn();
    const stream = createEventStream({
      topics: ['order:42'],
      handlers: { 'order.status': handler },
      onConnect,
      createEventSource: FakeEventSource.factory,
    });
    stream.start();
    const es = FakeEventSource.latest!;
    expect(es.url).toBe('/api/v1/stream?topics=order%3A42');
    es.open();
    expect(stream.status).toBe('open');
    expect(onConnect).toHaveBeenCalledTimes(1);
    es.emit('order.status', { id: '42', version: 3 });
    expect(handler).toHaveBeenCalledWith({ id: '42', version: 3 }, expect.anything());
    stream.stop();
    expect(es.closed).toBe(true);
    expect(stream.status).toBe('closed');
  });

  it('reconnects with backoff after a drop and refetches on reconnect', async () => {
    const onConnect = vi.fn();
    const statuses: StreamStatus[] = [];
    const stream = createEventStream({
      topics: [],
      handlers: {},
      onConnect,
      onStatusChange: (s) => statuses.push(s),
      createEventSource: FakeEventSource.factory,
      random: mid,
    });
    stream.start();
    FakeEventSource.latest!.open();
    FakeEventSource.latest!.fail(); // drop after being open → no refresh needed
    await vi.advanceTimersByTimeAsync(5_999);
    expect(FakeEventSource.instances).toHaveLength(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(FakeEventSource.instances).toHaveLength(2);
    FakeEventSource.latest!.open();
    expect(onConnect).toHaveBeenCalledTimes(2);
    expect(statuses).toEqual(['connecting', 'open', 'reconnecting', 'open']);
    stream.stop();
  });

  it('switches to polling after 3 failures within 60 s and keeps retrying every 60 s', async () => {
    const statuses: StreamStatus[] = [];
    const refresh = vi.fn(async () => 'unavailable' as const);
    const stream = createEventStream({
      topics: [],
      handlers: {},
      refresh,
      onStatusChange: (s) => statuses.push(s),
      createEventSource: FakeEventSource.factory,
      random: () => 0,
    });
    stream.start();
    FakeEventSource.latest!.fail(); // 1 → refresh once (may be a 401), retry in 2 s
    await vi.advanceTimersByTimeAsync(2_000);
    FakeEventSource.latest!.fail(); // 2 → 2.8 s
    await vi.advanceTimersByTimeAsync(2_800);
    FakeEventSource.latest!.fail(); // 3 within 60 s → polling
    expect(stream.status).toBe('polling');
    expect(refresh).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(59_999);
    expect(FakeEventSource.instances).toHaveLength(3);
    await vi.advanceTimersByTimeAsync(1);
    expect(FakeEventSource.instances).toHaveLength(4);
    FakeEventSource.latest!.open();
    expect(stream.status).toBe('open');
    stream.stop();
  });

  it('handles reauth: refreshes then reconnects immediately', async () => {
    const refresh = vi.fn(async () => 'ok' as const);
    const stream = createEventStream({ topics: [], handlers: {}, refresh, createEventSource: FakeEventSource.factory });
    stream.start();
    const first = FakeEventSource.latest!;
    first.open();
    first.emit('reauth', '');
    await vi.advanceTimersByTimeAsync(0);
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(first.closed).toBe(true);
    expect(FakeEventSource.instances).toHaveLength(2);
    stream.stop();
  });

  it('stops and reports an expired session when reauth refresh is rejected', async () => {
    const onSessionExpired = vi.fn();
    const stream = createEventStream({
      topics: [],
      handlers: {},
      refresh: async () => 'expired',
      onSessionExpired,
      createEventSource: FakeEventSource.factory,
    });
    stream.start();
    FakeEventSource.latest!.open();
    FakeEventSource.latest!.emit('reauth', '');
    await vi.advanceTimersByTimeAsync(0);
    expect(onSessionExpired).toHaveBeenCalledTimes(1);
    expect(stream.status).toBe('closed');
    await vi.advanceTimersByTimeAsync(120_000);
    expect(FakeEventSource.instances).toHaveLength(1);
  });

  it('reconnects immediately when the browser comes back online', async () => {
    const stream = createEventStream({ topics: [], handlers: {}, createEventSource: FakeEventSource.factory, random: () => 1 });
    stream.start();
    FakeEventSource.latest!.open();
    FakeEventSource.latest!.fail();
    window.dispatchEvent(new Event('online'));
    expect(FakeEventSource.instances).toHaveLength(2);
    stream.stop();
  });
});

describe('createQueryInvalidator', () => {
  it('invalidates mapped keys and drops stale versions', () => {
    const qc = new QueryClient();
    const spy = vi.spyOn(qc, 'invalidateQueries');
    const handlers = createQueryInvalidator(qc);
    handlers['order.status']!({ id: 'o1', version: 2, restaurantId: 'r1' }, { type: 'order.status', data: '' });
    expect(spy.mock.calls.map((c) => c[0]?.queryKey)).toEqual([
      qk.orders.detail('o1'),
      qk.orders.all(),
      qk.inbox.restaurant('r1'),
    ]);
    spy.mockClear();
    handlers['order.status']!({ id: 'o1', version: 2 }, { type: 'order.status', data: '' });
    handlers['order.status']!({ id: 'o1', version: 1 }, { type: 'order.status', data: '' });
    expect(spy).not.toHaveBeenCalled();
  });
});
