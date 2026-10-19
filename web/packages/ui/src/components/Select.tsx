import * as RadixSelect from '@radix-ui/react-select';
import { Check, ChevronDown } from 'lucide-react';
import { type ReactNode, useId } from 'react';
import { cn } from '../cn';
import { inputClass } from './Field';

export interface SelectOption {
  value: string;
  label: ReactNode;
  disabled?: boolean;
}

export interface SelectProps {
  label: ReactNode;
  options: SelectOption[];
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  placeholder?: ReactNode;
  hint?: ReactNode;
  error?: ReactNode;
  disabled?: boolean;
  name?: string;
  className?: string;
}

export function Select({ label, options, value, defaultValue, onValueChange, placeholder, hint, error, disabled, name, className }: SelectProps) {
  const id = `s${useId()}`;
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;
  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <label id={`${id}-label`} htmlFor={id} className="text-sm font-semibold text-fg">
        {label}
      </label>
      <RadixSelect.Root value={value} defaultValue={defaultValue} onValueChange={onValueChange} disabled={disabled} name={name}>
        <RadixSelect.Trigger
          id={id}
          aria-labelledby={`${id}-label`}
          aria-describedby={[hintId, errorId].filter(Boolean).join(' ') || undefined}
          aria-invalid={error ? true : undefined}
          className={cn(inputClass, 'flex items-center justify-between gap-2 text-start')}
        >
          <RadixSelect.Value placeholder={placeholder} />
          <RadixSelect.Icon>
            <ChevronDown aria-hidden className="size-5 text-fg-muted" />
          </RadixSelect.Icon>
        </RadixSelect.Trigger>
        <RadixSelect.Portal>
          <RadixSelect.Content
            position="popper"
            sideOffset={4}
            className="z-50 max-h-(--radix-select-content-available-height) min-w-(--radix-select-trigger-width) overflow-hidden rounded-md border border-border-subtle bg-surface text-fg shadow-card"
          >
            <RadixSelect.Viewport className="p-1">
              {options.map((o) => (
                <RadixSelect.Item
                  key={o.value}
                  value={o.value}
                  disabled={o.disabled}
                  className="relative flex min-h-11 cursor-pointer items-center rounded-sm ps-9 pe-3 outline-none select-none data-disabled:opacity-50 data-highlighted:bg-surface-muted"
                >
                  <RadixSelect.ItemIndicator className="absolute start-2 inline-flex">
                    <Check aria-hidden className="size-5 text-primary" />
                  </RadixSelect.ItemIndicator>
                  <RadixSelect.ItemText>{o.label}</RadixSelect.ItemText>
                </RadixSelect.Item>
              ))}
            </RadixSelect.Viewport>
          </RadixSelect.Content>
        </RadixSelect.Portal>
      </RadixSelect.Root>
      {hint && (
        <p id={hintId} className="text-sm text-fg-muted">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} role="alert" className="text-sm font-medium text-danger">
          {error}
        </p>
      )}
    </div>
  );
}
