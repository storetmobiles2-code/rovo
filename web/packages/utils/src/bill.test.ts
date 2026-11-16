import { describe, expect, it } from 'vitest';
import { visibleBillLines } from './bill';

const line = (component: string, amountPaise: number) => ({ component, amount: { amountPaise, currency: 'INR' } });

describe('visibleBillLines', () => {
  it('hides zero-value lines such as the platform fee (R59)', () => {
    const lines = [
      line('ITEM_TOTAL', 24000),
      line('PACKAGING', 0),
      line('DELIVERY_FEE', 2500),
      line('PLATFORM_FEE', 0),
      line('SMALL_CART_FEE', 0),
      line('DISCOUNT', -5000),
      line('TAX', 1200),
      line('ROUND_OFF', 0),
    ];
    expect(visibleBillLines(lines).map((l) => l.component)).toEqual(['ITEM_TOTAL', 'DELIVERY_FEE', 'DISCOUNT', 'TAX']);
  });

  it('keeps the item total even at zero and shows a non-zero platform fee if one is ever configured', () => {
    expect(visibleBillLines([line('ITEM_TOTAL', 0), line('PLATFORM_FEE', 500)])).toHaveLength(2);
  });
});
