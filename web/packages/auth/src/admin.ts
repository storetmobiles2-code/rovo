import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { type AdminMfaChallenge, type ApiError, qk, unwrap } from '@rovo/api-client';
import { useApiClient } from '@rovo/api-client/react';

export interface UseAdminLoginResult {
  step: 'credentials' | 'totp';
  mfaMethods: AdminMfaChallenge['mfaMethods'];
  submitting: boolean;
  credentialsError: ApiError | null;
  mfaError: ApiError | null;
  /** Email + password → TOTP step. */
  submitCredentials: (email: string, password: string) => Promise<void>;
  /** TOTP (or recovery) code → session. Resolves true on success. */
  submitTotp: (input: { totp?: string; recoveryCode?: string }) => Promise<boolean>;
  reset: () => void;
}

/** Admin login: email + password → mandatory TOTP step (doc 12; admin host only). */
export function useAdminLogin(): UseAdminLoginResult {
  const client = useApiClient();
  const queryClient = useQueryClient();
  const [mfaToken, setMfaToken] = useState<string | null>(null);
  const [mfaMethods, setMfaMethods] = useState<AdminMfaChallenge['mfaMethods']>([]);

  const credentials = useMutation<AdminMfaChallenge, ApiError, { email: string; password: string }>({
    mutationFn: (body) => unwrap(client.api.POST('/auth/admin/login', { body })),
    onSuccess: (data) => {
      setMfaToken(data.mfaToken);
      setMfaMethods(data.mfaMethods);
    },
  });

  const mfa = useMutation<boolean, ApiError, { totp?: string; recoveryCode?: string }>({
    mutationFn: async (codes) => {
      if (!mfaToken) throw new Error('No MFA challenge in progress');
      await unwrap(client.api.POST('/auth/admin/mfa', { body: { mfaToken, ...codes } }));
      return true;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: qk.me() });
    },
  });

  return {
    step: mfaToken ? 'totp' : 'credentials',
    mfaMethods,
    submitting: credentials.isPending || mfa.isPending,
    credentialsError: credentials.error ?? null,
    mfaError: mfa.error ?? null,
    submitCredentials: async (email, password) => {
      await credentials.mutateAsync({ email, password });
    },
    submitTotp: (input) => mfa.mutateAsync(input),
    reset: () => {
      setMfaToken(null);
      setMfaMethods([]);
      credentials.reset();
      mfa.reset();
    },
  };
}
