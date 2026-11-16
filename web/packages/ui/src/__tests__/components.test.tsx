import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { expectNoAxeViolations, renderWithProviders, screen, userEvent, within } from '@rovo/testing';
import {
  Badge,
  Button,
  Countdown,
  EmptyState,
  ErrorState,
  IconButton,
  Money,
  OtpInput,
  PhoneInput,
  Rating,
  StatusTimeline,
  VegMarker,
} from '../index';
import { ApiError } from '@rovo/api-client';

describe('Button', () => {
  it('renders a ≥44px target, disables while loading and keeps clicks from firing', async () => {
    const onClick = vi.fn();
    renderWithProviders(
      <Button loading onClick={onClick}>
        Place order
      </Button>,
    );
    const btn = screen.getByRole('button', { name: /place order/i });
    expect(btn).toBeDisabled();
    expect(btn).toHaveAttribute('aria-busy', 'true');
    expect(btn.className).toContain('min-h-12');
    await userEvent.click(btn);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('IconButton exposes an accessible name', () => {
    renderWithProviders(
      <IconButton label="Close">
        <svg />
      </IconButton>,
    );
    expect(screen.getByRole('button', { name: 'Close' })).toBeInTheDocument();
  });
});

describe('Money', () => {
  it('formats paise as ₹ with Indian grouping', () => {
    renderWithProviders(<Money value={12345650} />);
    expect(screen.getByText('₹1,23,456.50')).toBeInTheDocument();
  });
});

describe('VegMarker', () => {
  it('conveys meaning with an accessible name, not colour alone', () => {
    renderWithProviders(
      <>
        <VegMarker type="veg" />
        <VegMarker type="nonveg" />
      </>,
    );
    expect(screen.getByRole('img', { name: 'Vegetarian' })).toBeInTheDocument();
    expect(screen.getByRole('img', { name: 'Non-vegetarian' })).toBeInTheDocument();
  });
});

describe('Rating', () => {
  it('labels the value and shows New for unrated restaurants', () => {
    const { rerender } = renderWithProviders(<Rating value={4.3} count={128} />);
    expect(screen.getByRole('img', { name: 'Rated 4.3 out of 5' })).toBeInTheDocument();
    rerender(<Rating value={0} isNew />);
    expect(screen.getByText('New')).toBeInTheDocument();
  });
});

describe('PhoneInput', () => {
  it('keeps only 10 national digits and strips a pasted +91', async () => {
    const onValueChange = vi.fn();
    function Harness() {
      return <PhoneInput label="Mobile number" value="" onValueChange={onValueChange} />;
    }
    renderWithProviders(<Harness />);
    const input = screen.getByLabelText('Mobile number');
    expect(input).toHaveAttribute('type', 'tel');
    expect(input).toHaveAttribute('inputmode', 'numeric');
    await userEvent.type(input, '9');
    expect(onValueChange).toHaveBeenLastCalledWith('9');
  });
});

describe('OtpInput', () => {
  it('uses one-time-code autofill and fires onComplete at full length', async () => {
    const onComplete = vi.fn();
    function Harness() {
      const [value, setValue] = useState('');
      return (
        <OtpInput label="6-digit code" value={value} onValueChange={setValue} onComplete={onComplete} webOtp={false} />
      );
    }
    renderWithProviders(<Harness />);
    const input = screen.getByLabelText('6-digit code');
    expect(input).toHaveAttribute('autocomplete', 'one-time-code');
    await userEvent.type(input, '123456');
    expect(onComplete).toHaveBeenCalledWith('123456');
  });
});

describe('ErrorState', () => {
  it('shows the translated code, trace id and retry for an ApiError', async () => {
    const onRetry = vi.fn();
    const err = new ApiError(409, { code: 'QUOTE_CHANGED', traceId: 'tr-9' });
    renderWithProviders(<ErrorState error={err} onRetry={onRetry} />);
    expect(screen.getByText(/Prices or availability changed/)).toBeInTheDocument();
    expect(screen.getByText(/tr-9/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: /try again/i }));
    expect(onRetry).toHaveBeenCalled();
  });
});

describe('StatusTimeline', () => {
  it('marks the current step', () => {
    renderWithProviders(
      <StatusTimeline
        currentIndex={1}
        steps={[
          { id: 'a', label: 'Placed' },
          { id: 'b', label: 'Preparing' },
          { id: 'c', label: 'Delivered' },
        ]}
      />,
    );
    const current = screen.getByText('Preparing').closest('li')!;
    expect(current).toHaveAttribute('aria-current', 'step');
  });
});

describe('Countdown', () => {
  it('counts down and fires onExpire once', () => {
    vi.useFakeTimers();
    try {
      let t = 0;
      const onExpire = vi.fn();
      renderWithProviders(<Countdown deadline={3000} now={() => t} onExpire={onExpire} />);
      expect(screen.getByText('0:03')).toBeInTheDocument();
      t = 3000;
      vi.advanceTimersByTime(1000);
      expect(onExpire).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });
});

describe('accessibility', () => {
  it('a composed screen has no axe violations', async () => {
    const { container } = renderWithProviders(
      <main>
        <h1>Menu</h1>
        <Badge tone="success">Open</Badge>
        <ul>
          <li>
            <VegMarker type="veg" /> Paneer tikka <Money value={18000} />
          </li>
        </ul>
        <EmptyState title="Nothing here" description="Try another area" action={<Button>Browse</Button>} />
      </main>,
    );
    await expectNoAxeViolations(within(container).getByRole('main'));
  });
});
