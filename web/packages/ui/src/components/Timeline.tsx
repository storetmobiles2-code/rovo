import { Check } from 'lucide-react';
import { type ReactNode, useEffect, useRef, useState } from 'react';
import { useTranslation } from '@rovo/i18n';
import { formatDuration, secondsUntil } from '@rovo/utils';
import { cn } from '../cn';

// --- Stepper / StatusTimeline ----------------------------------------------
export interface TimelineStep {
  id: string;
  label: ReactNode;
  description?: ReactNode;
  /** Timestamp text (e.g. "2:05 pm"), shown when done. */
  at?: ReactNode;
}

export type StepState = 'done' | 'current' | 'upcoming';

export interface StatusTimelineProps {
  steps: TimelineStep[];
  /** Index of the current step; earlier steps are "done", later ones "upcoming". */
  currentIndex: number;
  /** `horizontal` for the rider step flow, `vertical` for order tracking. */
  orientation?: 'vertical' | 'horizontal';
  className?: string;
}

function stateOf(index: number, current: number): StepState {
  if (index < current) return 'done';
  if (index === current) return 'current';
  return 'upcoming';
}

/** Ordered status timeline (order tracking milestones; rider step flow). */
export function StatusTimeline({ steps, currentIndex, orientation = 'vertical', className }: StatusTimelineProps) {
  return (
    <ol
      className={cn('flex', orientation === 'vertical' ? 'flex-col' : 'items-start justify-between', className)}
      aria-label="Status"
    >
      {steps.map((step, i) => {
        const state = stateOf(i, currentIndex);
        const last = i === steps.length - 1;
        return (
          <li
            key={step.id}
            aria-current={state === 'current' ? 'step' : undefined}
            className={cn('relative flex', orientation === 'vertical' ? 'gap-3 pb-6 last:pb-0' : 'flex-1 flex-col items-center text-center')}
          >
            {!last && (
              <span
                aria-hidden
                className={cn(
                  'absolute bg-border-subtle',
                  orientation === 'vertical' ? 'start-[15px] top-8 bottom-0 w-0.5' : 'top-4 start-1/2 h-0.5 w-full',
                  state === 'done' && 'bg-success',
                )}
              />
            )}
            <span
              aria-hidden
              className={cn(
                'relative z-10 flex size-8 shrink-0 items-center justify-center rounded-full border-2',
                state === 'done' && 'border-success bg-success text-white',
                state === 'current' && 'border-primary bg-surface text-primary',
                state === 'upcoming' && 'border-border bg-surface text-fg-muted',
              )}
            >
              {state === 'done' ? <Check className="size-5" strokeWidth={3} /> : <span className="text-sm font-bold">{i + 1}</span>}
            </span>
            <span className={cn('flex flex-col', orientation === 'horizontal' && 'mt-2 items-center')}>
              <span className={cn('font-semibold', state === 'upcoming' ? 'text-fg-muted' : 'text-fg')}>{step.label}</span>
              {step.description && <span className="text-sm text-fg-muted">{step.description}</span>}
              {step.at && state === 'done' && <span className="text-xs text-fg-muted tabular-nums">{step.at}</span>}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

// --- Countdown --------------------------------------------------------------
export interface CountdownProps {
  /** Deadline (ISO string, ms epoch or Date). */
  deadline: string | number | Date;
  onExpire?: () => void;
  /** Milliseconds between ticks (tests inject a clock). Default 1000. */
  intervalMs?: number;
  /** `now()` provider for tests. */
  now?: () => number;
  className?: string;
}

/**
 * MM:SS countdown. Announces the remaining time politely at 30 s and 10 s (doc 17 §9), turns
 * danger-coloured under 10 s, and fires `onExpire` once at zero (offer timers, resend timers).
 */
export function Countdown({ deadline, onExpire, intervalMs = 1000, now = Date.now, className }: CountdownProps) {
  const { t } = useTranslation();
  const [remaining, setRemaining] = useState(() => secondsUntil(deadline, now()));
  const firedRef = useRef(false);

  useEffect(() => {
    firedRef.current = false;
    setRemaining(secondsUntil(deadline, now()));
    const id = setInterval(() => {
      const next = secondsUntil(deadline, now());
      setRemaining(next);
      if (next <= 0 && !firedRef.current) {
        firedRef.current = true;
        onExpire?.();
        clearInterval(id);
      }
    }, intervalMs);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [deadline, intervalMs]);

  const announce = remaining === 30 || remaining === 10;
  return (
    <span
      className={cn('font-mono text-lg font-bold tabular-nums', remaining <= 10 ? 'text-danger' : 'text-fg', className)}
      aria-live={announce ? 'polite' : 'off'}
    >
      <span aria-hidden>{formatDuration(remaining)}</span>
      <span className="sr-only">
        {remaining <= 0 ? t('countdown.expired') : t('countdown.remaining', { count: remaining })}
      </span>
    </span>
  );
}
