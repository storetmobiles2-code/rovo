import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations, renderWithProviders, screen } from '@rovo/testing';
import { BillBreakdown, type BillBreakdownLine } from '../index';

const line = (component: string, label: string, amountPaise: number): BillBreakdownLine => ({
  component,
  label,
  amount: { amountPaise, currency: 'INR' },
});

describe('BillBreakdown', () => {
  it('hides zero-value lines, so no "Platform fee ₹0" row is shown (R59)', async () => {
    renderWithProviders(
      <BillBreakdown
        lines={[
          line('ITEM_TOTAL', 'Item total', 24000),
          line('DELIVERY_FEE', 'Delivery fee', 2500),
          line('PLATFORM_FEE', 'Platform fee', 0),
          line('SMALL_CART_FEE', 'Small cart fee', 0),
          line('DISCOUNT', 'Discount', -5000),
          line('TAX', 'Taxes', 1200),
          line('ROUND_OFF', 'Round off', 0),
        ]}
        payable={{ amountPaise: 22700, currency: 'INR' }}
        savings={{ amountPaise: 5000, currency: 'INR' }}
      />,
    );
    expect(screen.getByTestId('bill-line-ITEM_TOTAL')).toBeInTheDocument();
    expect(screen.getByTestId('bill-line-DELIVERY_FEE')).toBeInTheDocument();
    expect(screen.getByTestId('bill-line-DISCOUNT')).toBeInTheDocument();
    expect(screen.queryByTestId('bill-line-PLATFORM_FEE')).not.toBeInTheDocument();
    expect(screen.queryByText(/platform fee/i)).not.toBeInTheDocument();
    expect(screen.queryByTestId('bill-line-SMALL_CART_FEE')).not.toBeInTheDocument();
    expect(screen.queryByTestId('bill-line-ROUND_OFF')).not.toBeInTheDocument();
    expect(screen.getByTestId('bill-payable')).toHaveTextContent('₹227');
    expect(screen.getByTestId('bill-savings')).toHaveTextContent('₹50');
    await expectNoAxeViolations();
  });

  it('falls back to the server label for an unknown component and hides zero savings', () => {
    renderWithProviders(
      <BillBreakdown lines={[line('SURGE', 'Rain surcharge', 1000)]} payable={{ amountPaise: 1000 }} savings={{ amountPaise: 0 }} />,
    );
    expect(screen.getByText('Rain surcharge')).toBeInTheDocument();
    expect(screen.queryByTestId('bill-savings')).not.toBeInTheDocument();
  });
});
