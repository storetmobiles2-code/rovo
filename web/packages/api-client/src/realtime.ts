import type { QueryClient, QueryKey } from '@tanstack/react-query';
import { qk } from './keys';
import type { StreamHandler } from './sse';

/** Thin invalidation hint (doc 17 §5.1): the client refetches over REST, never patches the cache. */
export interface RealtimeEvent {
  type?: string;
  id: string;
  version?: number;
  occurredAt?: string;
  restaurantId?: string;
  orderId?: string;
  deliveryId?: string;
  [extra: string]: unknown;
}

export type InvalidationMap = Record<string, (event: RealtimeEvent) => QueryKey[]>;

/**
 * Default event → query-key map. Event names are owned by docs 08/11; app teams extend this
 * map (spread + override) when the OpenAPI spec lists new events.
 */
export const defaultInvalidationMap: InvalidationMap = {
  'order.status': (e) => [
    qk.orders.detail(e.orderId ?? e.id),
    qk.orders.all(),
    ...(e.restaurantId ? [qk.inbox.restaurant(e.restaurantId)] : []),
  ],
  'order.placed': (e) => [e.restaurantId ? qk.inbox.restaurant(e.restaurantId) : qk.inbox.all()],
  'order.cancelled': (e) => [
    qk.orders.detail(e.orderId ?? e.id),
    qk.orders.all(),
    e.restaurantId ? qk.inbox.restaurant(e.restaurantId) : qk.inbox.all(),
  ],
  'offer.created': () => [qk.rider.offers(), qk.rider.state()],
  'offer.expired': (e) => [qk.rider.offer(e.id), qk.rider.state()],
  'offer.revoked': (e) => [qk.rider.offer(e.id), qk.rider.state()],
  'delivery.updated': (e) => [qk.rider.delivery(e.deliveryId ?? e.id), qk.rider.state()],
  'delivery.cancelled': (e) => [qk.rider.delivery(e.deliveryId ?? e.id), qk.rider.state()],
  'rider.state': () => [qk.rider.state()],
  'ops.alert': () => [qk.admin.live()],
  'restaurant.presence': () => [qk.admin.live()],
  'notification.created': () => [qk.notifications.all()],
};

function isRealtimeEvent(v: unknown): v is RealtimeEvent {
  return typeof v === 'object' && v !== null && typeof (v as { id?: unknown }).id === 'string';
}

/**
 * Builds SSE handlers that invalidate TanStack Query keys. Duplicate or stale events
 * (same entity, version ≤ last seen) are dropped.
 */
export function createQueryInvalidator(
  queryClient: QueryClient,
  map: InvalidationMap = defaultInvalidationMap,
  onEvent?: (name: string, event: RealtimeEvent) => void,
): Record<string, StreamHandler> {
  const seen = new Map<string, number>();
  const handlers: Record<string, StreamHandler> = {};
  for (const [name, toKeys] of Object.entries(map)) {
    handlers[name] = (data) => {
      if (!isRealtimeEvent(data)) return;
      if (typeof data.version === 'number') {
        const dedupeKey = `${name}:${data.id}`;
        const last = seen.get(dedupeKey);
        if (last !== undefined && data.version <= last) return;
        seen.set(dedupeKey, data.version);
      }
      for (const queryKey of toKeys(data)) void queryClient.invalidateQueries({ queryKey });
      onEvent?.(name, data);
    };
  }
  return handlers;
}
