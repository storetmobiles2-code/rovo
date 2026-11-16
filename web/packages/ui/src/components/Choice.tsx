import * as RadixCheckbox from '@radix-ui/react-checkbox';
import * as RadixRadio from '@radix-ui/react-radio-group';
import * as RadixSwitch from '@radix-ui/react-switch';
import { Check } from 'lucide-react';
import { type ReactNode, useId } from 'react';
import { cn } from '../cn';

const rowClass = 'flex min-h-11 cursor-pointer items-center gap-3 text-fg';

export interface CheckboxProps {
  label: ReactNode;
  description?: ReactNode;
  checked?: boolean;
  defaultChecked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  disabled?: boolean;
  name?: string;
  className?: string;
}

export function Checkbox({
  label,
  description,
  checked,
  defaultChecked,
  onCheckedChange,
  disabled,
  name,
  className,
}: CheckboxProps) {
  const id = `c${useId()}`;
  return (
    <div className={cn('flex items-start gap-3', className)}>
      <RadixCheckbox.Root
        id={id}
        name={name}
        checked={checked}
        defaultChecked={defaultChecked}
        disabled={disabled}
        onCheckedChange={(v) => onCheckedChange?.(v === true)}
        aria-describedby={description ? `${id}-d` : undefined}
        className="mt-2.5 flex size-6 shrink-0 items-center justify-center rounded-sm border-2 border-border bg-surface disabled:opacity-60 data-[state=checked]:border-primary data-[state=checked]:bg-primary"
      >
        <RadixCheckbox.Indicator>
          <Check aria-hidden className="size-4 text-on-primary" strokeWidth={3} />
        </RadixCheckbox.Indicator>
      </RadixCheckbox.Root>
      <div className="flex flex-col">
        <label htmlFor={id} className={rowClass}>
          {label}
        </label>
        {description && (
          <p id={`${id}-d`} className="text-sm text-fg-muted">
            {description}
          </p>
        )}
      </div>
    </div>
  );
}

export interface RadioOption {
  value: string;
  label: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
}

export interface RadioGroupProps {
  label: ReactNode;
  options: RadioOption[];
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  name?: string;
  orientation?: 'vertical' | 'horizontal';
  className?: string;
}

export function RadioGroup({
  label,
  options,
  value,
  defaultValue,
  onValueChange,
  name,
  orientation = 'vertical',
  className,
}: RadioGroupProps) {
  const id = `r${useId()}`;
  return (
    <fieldset className={cn('flex flex-col gap-2', className)}>
      <legend className="mb-1 text-sm font-semibold text-fg">{label}</legend>
      <RadixRadio.Root
        value={value}
        defaultValue={defaultValue}
        onValueChange={onValueChange}
        name={name}
        orientation={orientation}
        className={cn('flex gap-2', orientation === 'vertical' ? 'flex-col' : 'flex-row flex-wrap')}
      >
        {options.map((o) => {
          const itemId = `${id}-${o.value}`;
          return (
            <div key={o.value} className="flex items-start gap-3">
              <RadixRadio.Item
                id={itemId}
                value={o.value}
                disabled={o.disabled}
                aria-describedby={o.description ? `${itemId}-d` : undefined}
                className="mt-2.5 flex size-6 shrink-0 items-center justify-center rounded-full border-2 border-border bg-surface disabled:opacity-60 data-[state=checked]:border-primary"
              >
                <RadixRadio.Indicator className="size-3 rounded-full bg-primary" />
              </RadixRadio.Item>
              <div className="flex flex-col">
                <label htmlFor={itemId} className={rowClass}>
                  {o.label}
                </label>
                {o.description && (
                  <p id={`${itemId}-d`} className="text-sm text-fg-muted">
                    {o.description}
                  </p>
                )}
              </div>
            </div>
          );
        })}
      </RadixRadio.Root>
    </fieldset>
  );
}

export interface SwitchProps {
  label: ReactNode;
  description?: ReactNode;
  checked?: boolean;
  defaultChecked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  disabled?: boolean;
  name?: string;
  className?: string;
}

/** On/off toggle (rider online, outlet open, item in stock). */
export function Switch({
  label,
  description,
  checked,
  defaultChecked,
  onCheckedChange,
  disabled,
  name,
  className,
}: SwitchProps) {
  const id = `w${useId()}`;
  return (
    <div className={cn('flex items-center justify-between gap-4', className)}>
      <div className="flex flex-col">
        <label htmlFor={id} className={cn(rowClass, 'font-medium')}>
          {label}
        </label>
        {description && (
          <p id={`${id}-d`} className="text-sm text-fg-muted">
            {description}
          </p>
        )}
      </div>
      <RadixSwitch.Root
        id={id}
        name={name}
        checked={checked}
        defaultChecked={defaultChecked}
        onCheckedChange={onCheckedChange}
        disabled={disabled}
        aria-describedby={description ? `${id}-d` : undefined}
        className="relative inline-flex h-8 w-14 shrink-0 items-center rounded-full border-2 border-border bg-ink-200 transition-colors disabled:opacity-60 data-[state=checked]:border-success data-[state=checked]:bg-success"
      >
        <RadixSwitch.Thumb className="block size-6 translate-x-0.5 rounded-full bg-white shadow-card transition-transform data-[state=checked]:translate-x-6" />
      </RadixSwitch.Root>
    </div>
  );
}
