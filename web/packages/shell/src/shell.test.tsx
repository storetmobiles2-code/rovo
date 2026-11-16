import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations, renderWithProviders, screen, userEvent } from '@rovo/testing';
import { vi } from 'vitest';
import { ForbiddenScreen, NotFoundScreen, PlaceholderScreen, WrongAppScreen } from './screens';
import { __setRuntimeConfig, readRuntimeConfig } from './runtime-config';

describe('shell screens', () => {
  it('NotFoundScreen renders 404 with an accessible home action', async () => {
    const onHome = vi.fn();
    renderWithProviders(<NotFoundScreen onHome={onHome} />);
    expect(screen.getByTestId('page-not-found')).toBeInTheDocument();
    expect(screen.getByText('404')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: /home/i }));
    expect(onHome).toHaveBeenCalled();
    await expectNoAxeViolations();
  });

  it('ForbiddenScreen explains no access', async () => {
    renderWithProviders(<ForbiddenScreen />);
    expect(screen.getByTestId('page-forbidden')).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('WrongAppScreen names the right audience', () => {
    renderWithProviders(<WrongAppScreen audience="rider" />);
    expect(screen.getByText(/delivery partners/i)).toBeInTheDocument();
  });

  it('PlaceholderScreen shows the page header and coming-soon badge', async () => {
    renderWithProviders(<PlaceholderScreen title="Earnings" testId="rider-earnings-screen" />);
    expect(screen.getByRole('heading', { level: 1, name: 'Earnings' })).toBeInTheDocument();
    expect(screen.getByText('Coming soon')).toBeInTheDocument();
    await expectNoAxeViolations();
  });
});

describe('runtime config', () => {
  it('falls back to dev defaults when no config script is present', () => {
    __setRuntimeConfig(null);
    const cfg = readRuntimeConfig();
    expect(cfg.apiBase).toBe('/api/v1');
    expect(cfg.env).toBe('local');
  });

  it('reads the admin meta-tag config (no inline script needed) and the script-tag config', () => {
    const meta = document.createElement('meta');
    meta.name = 'rovo-config';
    meta.content = '{ "apiBase": "/x/v1", "env": "staging" }';
    document.head.append(meta);
    __setRuntimeConfig(null);
    expect(readRuntimeConfig()).toMatchObject({ apiBase: '/x/v1', env: 'staging', mediaBase: '/media' });
    meta.remove();

    const script = document.createElement('script');
    script.id = 'rovo-config';
    script.type = 'application/json';
    script.textContent = '{ "apiBase": "/y/v1" }';
    document.head.append(script);
    __setRuntimeConfig(null);
    expect(readRuntimeConfig().apiBase).toBe('/y/v1');
    script.remove();
    __setRuntimeConfig(null);
  });
});
