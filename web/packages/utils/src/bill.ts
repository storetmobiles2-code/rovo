import type { Paise } from './money';

/** Structural subset of the OpenAPI `BillLine` this helper needs. */
export interface BillLineLike {
  component: string;
  amount: { amountPaise: Paise };
}

/**
 * Lines a bill/price breakdown should show. Zero-value lines are hidden (R59: there is no
 * platform fee, so a "Platform fee ₹0" row must never appear; the same goes for a ₹0 small-cart
 * fee, discount, packaging or round-off). The item total always stays.
 */
export function visibleBillLines<T extends BillLineLike>(lines: readonly T[]): T[] {
  return lines.filter((line) => line.component === 'ITEM_TOTAL' || line.amount.amountPaise !== 0);
}
