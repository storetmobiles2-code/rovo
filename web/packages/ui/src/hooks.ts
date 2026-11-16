import { useCallback, useRef, useSyncExternalStore } from 'react';

function subscribeOnline(callback: () => void): () => void {
  if (typeof window === 'undefined') return () => {};
  window.addEventListener('online', callback);
  window.addEventListener('offline', callback);
  return () => {
    window.removeEventListener('online', callback);
    window.removeEventListener('offline', callback);
  };
}

/** Tracks `navigator.onLine`. Pair with the API's failed-fetch heuristic for accuracy (doc 17 §8). */
export function useOnline(): boolean {
  return useSyncExternalStore(
    subscribeOnline,
    () => (typeof navigator === 'undefined' ? true : navigator.onLine),
    () => true,
  );
}

function subscribeReducedMotion(callback: () => void): () => void {
  if (typeof window === 'undefined' || !window.matchMedia) return () => {};
  const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
  mq.addEventListener('change', callback);
  return () => mq.removeEventListener('change', callback);
}

export function usePrefersReducedMotion(): boolean {
  return useSyncExternalStore(
    subscribeReducedMotion,
    () => (typeof window !== 'undefined' && window.matchMedia ? window.matchMedia('(prefers-reduced-motion: reduce)').matches : false),
    () => false,
  );
}

/**
 * Focuses an element once, when it mounts. Returns a callback ref to spread onto the control.
 * This is the accessible alternative to the `autoFocus` attribute (jsx-a11y/no-autofocus): it
 * focuses programmatically only on the first mount, e.g. the first field of a dedicated
 * single-purpose screen such as login.
 */
export function useAutoFocus<T extends HTMLElement>(enabled = true): (node: T | null) => void {
  const done = useRef(false);
  return useCallback(
    (node: T | null) => {
      if (!enabled || done.current || !node) return;
      done.current = true;
      node.focus();
    },
    [enabled],
  );
}
