import { useTranslation } from '@rovo/i18n';
import { type Paise, visibleBillLines } from '@rovo/utils';
import { cn } from '../cn';
import { Money } from './Display';

/** Structural match for the OpenAPI `Bill` (kept local so `ui` does not depend on the client). */
export interface BillBreakdownLine {
  component: string;
  /** Server-localised label; used when the catalog has no entry for `component`. */
  label: string;
  amount: { amountPaise: Paise; currency?: string };
  /** True when the amount is already inside another line (e.g. tax-inclusive prices). */
  isIncluded?: boolean;
}

export interface BillBreakdownProps {
  lines: readonly BillBreakdownLine[];
  payable: { amountPaise: Paise; currency?: string };
  savings?: { amountPaise: Paise; currency?: string } | null;
  className?: string;
}

/**
 * Bill / price breakdown. Zero-value lines are never rendered (R59: no "Platform fee ₹0" row).
 */
export function BillBreakdown({ lines, payable, savings, className }: BillBreakdownProps) {
  const { t } = useTranslation();
  const shown = visibleBillLines(lines);
  return (
    <section aria-label={t('bill.title')} className={cn('text-fg', className)} data-testid="bill-breakdown">
      <dl className="flex flex-col gap-2">
        {shown.map((line, index) => (
          <div key={`${line.component}-${index}`} className="flex items-baseline justify-between gap-4" data-testid={`bill-line-${line.component}`}>
            <dt className="text-fg-muted">
              {t(`bill.${line.component}`, { defaultValue: line.label })}
              {line.isIncluded && <span className="ms-1 text-sm">({t('bill.included')})</span>}
            </dt>
            <dd className={cn(line.amount.amountPaise < 0 && 'font-semibold text-success')}>
              <Money value={line.amount.amountPaise} />
            </dd>
          </div>
        ))}
        <div className="mt-1 flex items-baseline justify-between gap-4 border-t border-border-subtle pt-3 text-lg font-semibold" data-testid="bill-payable">
          <dt>{t('bill.payable')}</dt>
          <dd>
            <Money value={payable.amountPaise} />
          </dd>
        </div>
      </dl>
      {savings && savings.amountPaise > 0 && (
        <p className="mt-2 text-sm font-semibold text-success" data-testid="bill-savings">
          {t('bill.savings')} <Money value={savings.amountPaise} />
        </p>
      )}
    </section>
  );
}
