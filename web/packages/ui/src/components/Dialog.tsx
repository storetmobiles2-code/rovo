import * as RadixDialog from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import type { ReactNode } from 'react';
import { useTranslation } from '@rovo/i18n';
import { cn } from '../cn';

interface BaseDialogProps {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** Optional trigger element (rendered asChild). */
  trigger?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  /** Action buttons row. */
  footer?: ReactNode;
  /** Hide the × close button (e.g. blocking 409 conflict dialogs). */
  hideClose?: boolean;
  className?: string;
}

const overlay = 'fixed inset-0 z-40 bg-ink-900/50';

function CloseButton() {
  const { t } = useTranslation();
  return (
    <RadixDialog.Close
      aria-label={t('actions.close')}
      className="absolute end-2 top-2 inline-flex size-12 items-center justify-center rounded-full text-fg hover:bg-surface-muted"
    >
      <X aria-hidden className="size-6" />
    </RadixDialog.Close>
  );
}

function Description({ description, title }: { description?: ReactNode; title: ReactNode }) {
  return description ? (
    <RadixDialog.Description className="mt-1 text-fg-muted">{description}</RadixDialog.Description>
  ) : (
    <RadixDialog.Description className="sr-only">{title}</RadixDialog.Description>
  );
}

/** Centred modal dialog. Use for blocking business conflicts (QUOTE_CHANGED, OFFER_EXPIRED …). */
export function Dialog({
  open,
  onOpenChange,
  trigger,
  title,
  description,
  children,
  footer,
  hideClose,
  className,
}: BaseDialogProps) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      {trigger && <RadixDialog.Trigger asChild>{trigger}</RadixDialog.Trigger>}
      <RadixDialog.Portal>
        <RadixDialog.Overlay className={overlay} />
        <RadixDialog.Content
          className={cn(
            'fixed start-1/2 top-1/2 z-50 w-[min(92vw,28rem)] -translate-x-1/2 -translate-y-1/2 rounded-lg bg-surface p-5 text-fg shadow-sheet',
            className,
          )}
        >
          <RadixDialog.Title className="pe-10 text-lg font-semibold">{title}</RadixDialog.Title>
          <Description description={description} title={title} />
          {children && <div className="mt-4">{children}</div>}
          {footer && <div className="mt-6 flex flex-wrap justify-end gap-2">{footer}</div>}
          {!hideClose && <CloseButton />}
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
}

export interface SheetProps extends BaseDialogProps {
  /** `bottom` (mobile item sheet, default) or `end` (side panel on tablets/desktop). */
  side?: 'bottom' | 'end';
}

/** Bottom sheet / side sheet on Radix Dialog (focus trap, Esc, scroll lock, labelled). */
export function Sheet({
  open,
  onOpenChange,
  trigger,
  title,
  description,
  children,
  footer,
  hideClose,
  side = 'bottom',
  className,
}: SheetProps) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      {trigger && <RadixDialog.Trigger asChild>{trigger}</RadixDialog.Trigger>}
      <RadixDialog.Portal>
        <RadixDialog.Overlay className={overlay} />
        <RadixDialog.Content
          className={cn(
            'fixed z-50 flex flex-col bg-surface text-fg shadow-sheet',
            side === 'bottom'
              ? 'inset-x-0 bottom-0 max-h-[90dvh] rounded-t-xl pb-[max(1rem,env(safe-area-inset-bottom))]'
              : 'inset-y-0 end-0 w-[min(100vw,26rem)]',
            className,
          )}
        >
          {side === 'bottom' && (
            <div aria-hidden className="mx-auto mt-2 h-1.5 w-10 rounded-full bg-ink-200" />
          )}
          <div className="px-5 pt-4">
            <RadixDialog.Title className="pe-10 text-lg font-semibold">{title}</RadixDialog.Title>
            <Description description={description} title={title} />
          </div>
          <div className="flex-1 overflow-y-auto px-5 py-4">{children}</div>
          {footer && (
            <div className="flex gap-2 border-t border-border-subtle px-5 pt-3">{footer}</div>
          )}
          {!hideClose && <CloseButton />}
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
}

/** Alias: the bottom variant of `Sheet`. */
export const BottomSheet = (props: Omit<SheetProps, 'side'>) => <Sheet {...props} side="bottom" />;

export const DialogClose = RadixDialog.Close;
