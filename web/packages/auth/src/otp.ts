import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useRef, useState } from 'react';
import { type ApiError, qk, unwrap } from '@rovo/api-client';
import { useApiClient } from '@rovo/api-client/react';
import { normalizeIndianMobile, uuidv7 } from '@rovo/utils';

const DEVICE_KEY = 'rovo.deviceInstallId';

/** Stable per-device install id (used for OTP verify + sessions). Falls back to memory. */
export function getDeviceInstallId(): string {
  try {
    const existing = localStorage.getItem(DEVICE_KEY);
    if (existing) return existing;
    const id = uuidv7();
    localStorage.setItem(DEVICE_KEY, id);
    return id;
  } catch {
    return uuidv7();
  }
}

export interface OtpChallengeState {
  challengeId: string;
  phone: string;
  resendAfterSeconds: number;
  expiresAt: string;
}

export interface UsePhoneOtpResult {
  step: 'phone' | 'code';
  challenge: OtpChallengeState | null;
  /** Seconds until resend is allowed (0 = allowed). */
  resendIn: number;
  requesting: boolean;
  verifying: boolean;
  requestError: ApiError | null;
  verifyError: ApiError | null;
  /** Normalises to E.164 and requests an OTP. Rejects an invalid number without a request. */
  requestOtp: (phoneInput: string) => Promise<void>;
  resend: () => Promise<void>;
  verify: (code: string) => Promise<boolean>;
  reset: () => void;
}

/**
 * Phone-OTP login flow (request → verify) with a resend countdown. On success the session
 * cookies are set by the server and the `me` query is invalidated so the guard re-evaluates.
 */
export function usePhoneOtp(): UsePhoneOtpResult {
  const client = useApiClient();
  const queryClient = useQueryClient();
  const [challenge, setChallenge] = useState<OtpChallengeState | null>(null);
  const [resendIn, setResendIn] = useState(0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const startCountdown = useCallback((seconds: number) => {
    setResendIn(seconds);
    if (timerRef.current) clearInterval(timerRef.current);
    timerRef.current = setInterval(() => {
      setResendIn((s) => {
        if (s <= 1 && timerRef.current) clearInterval(timerRef.current);
        return Math.max(0, s - 1);
      });
    }, 1000);
  }, []);

  useEffect(() => () => void (timerRef.current && clearInterval(timerRef.current)), []);

  const request = useMutation<OtpChallengeState, ApiError, string>({
    mutationFn: async (phone: string) => {
      const data = await unwrap(client.api.POST('/auth/otp/request', { body: { phone, channel: 'SMS' } }));
      return { ...data, phone };
    },
    onSuccess: (data) => {
      setChallenge(data);
      startCountdown(data.resendAfterSeconds);
    },
  });

  const verifyMutation = useMutation<boolean, ApiError, string>({
    mutationFn: async (code: string) => {
      if (!challenge) throw new Error('No OTP challenge in progress');
      await unwrap(
        client.api.POST('/auth/otp/verify', {
          body: { challengeId: challenge.challengeId, code, deviceInstallId: getDeviceInstallId() },
        }),
      );
      return true;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: qk.me() });
    },
  });

  const requestOtp = useCallback(
    async (phoneInput: string) => {
      const e164 = normalizeIndianMobile(phoneInput);
      if (!e164) {
        // Surface as a field error without hitting the network.
        throw Object.assign(new Error('invalidPhone'), { code: 'VALIDATION_FAILED', fieldErrors: [{ field: 'phone', code: 'invalidPhone' }] });
      }
      await request.mutateAsync(e164);
    },
    [request],
  );

  return {
    step: challenge ? 'code' : 'phone',
    challenge,
    resendIn,
    requesting: request.isPending,
    verifying: verifyMutation.isPending,
    requestError: request.error ?? null,
    verifyError: verifyMutation.error ?? null,
    requestOtp,
    resend: async () => {
      if (challenge && resendIn === 0) await request.mutateAsync(challenge.phone);
    },
    verify: (code: string) => verifyMutation.mutateAsync(code),
    reset: () => {
      setChallenge(null);
      setResendIn(0);
      request.reset();
      verifyMutation.reset();
    },
  };
}
