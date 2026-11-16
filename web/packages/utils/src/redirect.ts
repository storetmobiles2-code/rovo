/**
 * Sanitises a post-login `?redirect=` target. Only same-origin app paths are accepted, so a
 * crafted link such as `?redirect=https://evil.example` or `//evil.example` cannot bounce a
 * freshly signed-in user off-site; the login page itself is never a target (no loops).
 */
export function safeRedirect(target: string | null | undefined, fallback = '/'): string {
  if (!target || !target.startsWith('/') || target.startsWith('//') || target.includes('\\')) return fallback;
  if (target === '/login' || target.startsWith('/login?') || target.startsWith('/login/')) return fallback;
  return target;
}
