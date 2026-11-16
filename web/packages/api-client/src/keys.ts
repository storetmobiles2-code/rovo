/**
 * Query key factories (doc 17 §4.2). Build every TanStack Query key through `qk` so
 * invalidation can target prefixes. Locale is never part of a key.
 */
export interface RestaurantFilters {
  veg?: boolean;
  cuisine?: string;
  sort?: string;
  q?: string;
}

export interface ListFilters {
  status?: string;
  cursor?: string;
  [k: string]: string | number | boolean | undefined;
}

export const qk = {
  me: () => ['me'] as const,
  config: (app: string) => ['client-config', app] as const,
  restaurants: {
    all: () => ['restaurants'] as const,
    list: (city: string, f: RestaurantFilters = {}) => ['restaurants', 'list', city, f] as const,
    detail: (slug: string) => ['restaurants', 'detail', slug] as const,
    menu: (id: string) => ['restaurants', id, 'menu'] as const,
  },
  addresses: { all: () => ['addresses'] as const },
  quote: (hash: string) => ['quote', hash] as const,
  orders: {
    all: () => ['orders'] as const,
    list: (f: ListFilters = {}) => ['orders', 'list', f] as const,
    detail: (id: string) => ['orders', 'detail', id] as const,
  },
  inbox: {
    all: () => ['inbox'] as const,
    restaurant: (restaurantId: string) => ['inbox', restaurantId] as const,
  },
  rider: {
    state: () => ['rider', 'state'] as const,
    offers: () => ['rider', 'offer'] as const,
    offer: (id: string) => ['rider', 'offer', id] as const,
    delivery: (id: string) => ['rider', 'delivery', id] as const,
    earnings: (week?: string) => ['rider', 'earnings', week ?? 'current'] as const,
  },
  notifications: { all: () => ['notifications'] as const },
  admin: {
    all: () => ['admin'] as const,
    live: (cityId?: string) => ['admin', 'live', cityId ?? 'all'] as const,
    list: (resource: string, f: ListFilters = {}) => ['admin', resource, 'list', f] as const,
    detail: (resource: string, id: string) => ['admin', resource, 'detail', id] as const,
  },
} as const;
