import type { BackendModule, ReadCallback } from 'i18next';
import { pseudoLocalize } from './pseudo';

type Catalog = Record<string, unknown>;

// Each namespace/language becomes its own small chunk; only what the app uses is fetched.
const loaders = import.meta.glob<{ default: Catalog }>('../locales/*/*.json');

export async function loadCatalog(lng: string, ns: string): Promise<Catalog> {
  const source = lng === 'en-XA' ? 'en' : lng;
  const loader = loaders[`../locales/${source}/${ns}.json`];
  if (!loader) throw new Error(`Missing catalog ${source}/${ns}`);
  const mod = await loader();
  return lng === 'en-XA' ? pseudoLocalize(mod.default) : mod.default;
}

export const lazyCatalogBackend: BackendModule = {
  type: 'backend',
  init() {},
  read(language: string, namespace: string, callback: ReadCallback) {
    loadCatalog(language, namespace).then(
      (data) => callback(null, data),
      (err: unknown) => callback(err instanceof Error ? err : new Error(String(err)), false),
    );
  },
};
