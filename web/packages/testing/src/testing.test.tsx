import { describe, expect, it } from 'vitest';
import { createApiClient, unwrap } from '@rovo/api-client';
import { authHandlers, createAuthState, fixtures, setupMsw } from './msw';
import { axeViolations, expectNoAxeViolations, renderWithProviders } from './index';

const state = createAuthState();
setupMsw(...authHandlers(state));

describe('@rovo/testing', () => {
  it('MSW auth scaffold exercises the real client incl. refresh-retry', async () => {
    const { api } = createApiClient({ app: 'customer', version: '0.0.0' });
    await expect(unwrap(api.GET('/me'))).rejects.toMatchObject({ code: 'UNAUTHENTICATED' });
    state.session = fixtures.customer();
    state.expireAccessOnce = true;
    state.calls.length = 0;
    await expect(unwrap(api.GET('/me'))).resolves.toMatchObject({ id: 'usr_customer_1' });
    expect(state.calls).toEqual(['me', 'refresh', 'me']);
  });

  it('axe helper reports violations and passes clean markup', async () => {
    const { container } = renderWithProviders(<img src="x.png" />);
    expect((await axeViolations(container)).map((v) => v.id)).toContain('image-alt');
    const ok = renderWithProviders(<button type="button">Save</button>);
    await expectNoAxeViolations(ok.container);
  });
});
