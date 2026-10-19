import * as RadixToast from '@radix-ui/react-toast';
import { CircleCheck, Info, TriangleAlert, X } from 'lucide-react';
import { type ReactNode, useSyncExternalStore } from 'react';
import { useTranslation } from '@rovo/i18n';
import { cn } from '../cn';

export type ToastTone = 'info' | 'success' | 'error';

export interface ToastInput {
  title: ReactNode;
  description?: ReactNode;
  tone?: ToastTone;
  /** Optional action, e.g. { label: 'Reload', onClick: … }. */
  action?: { label: string; onClick: () => void };
  /** ms; `Infinity` keeps it until dismissed (e.g. "New version available"). */
  duration?: number;
}

interface ToastItem extends ToastInput {
  id: number;
}

let items: ToastItem[] = [];
let nextId = 1;
const listeners = new Set<() => void>();
const emit = () => listeners.forEach((l) => l());

/** Imperative toast API: `toast({ title: t('…'), tone: 'success' })`. Returns a dismiss fn. */
export function toast(input: ToastInput): () => void {
  const id = nextId++;
  items = [...items, { ...input, id }];
  emit();
  return () => dismissToast(id);
}

export function dismissToast(id: number) {
  items = items.filter((t) => t.id !== id);
  emit();
}

function useToasts(): ToastItem[] {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => items,
    () => items,
  );
}

const toneIcon = { info: Info, success: CircleCheck, error: TriangleAlert } as const;
const toneClass: Record<ToastTone, string> = {
  info: 'border-info',
  success: 'border-success',
  error: 'border-danger',
};

/** Mount once near the root (AppProviders does this). Polite live region, swipe to dismiss. */
export function Toaster() {
  const { t } = useTranslation();
  const list = useToasts();
  return (
    <RadixToast.Provider swipeDirection="down" label={t('toast.region')}>
      {list.map((item) => {
        const tone = item.tone ?? 'info';
        const Icon = toneIcon[tone];
        return (
          <RadixToast.Root
            key={item.id}
            duration={item.duration ?? 5000}
            type={tone === 'error' ? 'foreground' : 'background'}
            onOpenChange={(open) => !open && dismissToast(item.id)}
            className={cn('flex items-start gap-3 rounded-lg border-s-4 bg-surface p-4 text-fg shadow-sheet', toneClass[tone])}
          >
            <Icon aria-hidden className="mt-0.5 size-5 shrink-0" />
            <div className="flex flex-1 flex-col gap-1">
              <RadixToast.Title className="font-semibold">{item.title}</RadixToast.Title>
              {item.description && <RadixToast.Description className="text-sm text-fg-muted">{item.description}</RadixToast.Description>}
            </div>
            {item.action && (
              <RadixToast.Action altText={item.action.label} asChild>
                <button
                  type="button"
                  onClick={item.action.onClick}
                  className="min-h-11 rounded-md px-3 font-semibold text-link hover:bg-surface-muted"
                >
                  {item.action.label}
                </button>
              </RadixToast.Action>
            )}
            <RadixToast.Close aria-label={t('actions.dismiss')} className="inline-flex size-11 items-center justify-center rounded-full hover:bg-surface-muted">
              <X aria-hidden className="size-5" />
            </RadixToast.Close>
          </RadixToast.Root>
        );
      })}
      <RadixToast.Viewport className="fixed inset-x-0 bottom-0 z-[60] mx-auto flex w-full max-w-md flex-col gap-2 p-4 pb-[max(1rem,env(safe-area-inset-bottom))] outline-none" />
    </RadixToast.Provider>
  );
}
