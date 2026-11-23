package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
)

// OTP policy (doc 12 §2.2, AUTH-D07).
const (
	otpTTL            = 5 * time.Minute
	otpMaxAttempts    = 5
	otpSendsPerHour   = 5
	otpSendsPerDay    = 10
	otpIPSendsPer10m  = 100 // 5× the soft limit of 20; the soft limit feeds the bot challenge later
	otpFailsPerHour   = 10
	otpResendLookback = 15 * time.Minute
)

// otpResendCooldowns is the wait before the 2nd, 3rd, 4th and later sends to one phone.
var otpResendCooldowns = []time.Duration{30 * time.Second, 60 * time.Second, 120 * time.Second, 300 * time.Second}

// indiaMobileRe is E.164 +91 with a 10-digit mobile starting 6–9 (no international numbers in V1).
var indiaMobileRe = regexp.MustCompile(`^\+91[6-9][0-9]{9}$`)

var otpCodeRe = regexp.MustCompile(`^[0-9]{6}$`)

// memberAudience reports whether aud is an app that members sign in to by OTP.
func memberAudience(aud auth.Audience) bool {
	return aud == auth.AudienceCustomer || aud == auth.AudienceRestaurant || aud == auth.AudienceRider
}

func (s *service) phoneBucket(prefix, phone string) string {
	return prefix + hex.EncodeToString(auth.MAC(s.limKey, []byte(phone))[:16])
}

func (s *service) codeMAC(challengeID uuid.UUID, code string) []byte {
	return auth.MAC(s.otpKey, challengeID[:], []byte(code))
}

func newOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func cooldownFor(priorSends int) time.Duration {
	if priorSends <= 0 {
		return 0
	}
	return otpResendCooldowns[min(priorSends, len(otpResendCooldowns))-1]
}

// StartOTP creates a challenge and sends the code. The response is the same whether or not an
// account exists for the phone (no enumeration); blocked accounts are refused only at verify.
func (s *service) StartOTP(ctx context.Context, in StartOTPInput) (Challenge, error) {
	if !memberAudience(in.Audience) {
		return Challenge{}, apperr.Forbidden(apperr.CodeForbidden, "OTP sign-in is not available on this app.")
	}
	if !indiaMobileRe.MatchString(in.Phone) {
		return Challenge{}, apperr.Validation(apperr.FieldError{Field: "phone", Code: "INVALID_PHONE",
			Detail: "Enter a 10-digit Indian mobile number."})
	}
	if s.sender == nil {
		return Challenge{}, apperr.Unavailable("", "SMS is not available right now.")
	}
	now := s.deps.Clock.Now()

	// Resend cooldown: escalates with the number of recent sends to this phone.
	var prior int
	var last *time.Time
	if err := s.deps.Pool.QueryRow(ctx, `SELECT count(*), max(created_at) FROM otp_challenges
		WHERE phone_e164 = $1 AND purpose = 'LOGIN' AND created_at > $2`,
		in.Phone, now.Add(-otpResendLookback)).Scan(&prior, &last); err != nil {
		return Challenge{}, err
	}
	if last != nil {
		if wait := last.Add(cooldownFor(prior)).Sub(now); wait > 0 {
			return Challenge{}, apperr.TooManyRequests(apperr.CodeOTPResendTooSoon, int(wait.Seconds())+1)
		}
	}

	// Send budgets. The limiter is durable and shared across replicas (R21).
	budgets := []struct {
		key    string
		limit  int
		window time.Duration
	}{
		{s.phoneBucket("otp:send:h:", in.Phone), otpSendsPerHour, time.Hour},
		{s.phoneBucket("otp:send:d:", in.Phone), otpSendsPerDay, 24 * time.Hour},
	}
	if in.IP != "" {
		budgets = append(budgets, struct {
			key    string
			limit  int
			window time.Duration
		}{"otp:send:ip:" + in.IP, otpIPSendsPer10m, 10 * time.Minute})
	}
	for _, b := range budgets {
		d, err := s.deps.RateLimit.Allow(ctx, b.key, b.limit, b.window)
		if err != nil {
			return Challenge{}, err // fail closed: OTP sends cost money
		}
		if !d.Allowed {
			return Challenge{}, apperr.TooManyRequests(apperr.CodeRateLimited, int(d.RetryAfter.Seconds())+1)
		}
	}

	code, err := newOTPCode()
	if err != nil {
		return Challenge{}, err
	}
	ch := Challenge{ID: s.deps.IDs.NewID(), ExpiresAt: now.Add(otpTTL), ResendAfterSeconds: int(cooldownFor(prior+1) / time.Second)}
	err = db.WithTx(ctx, s.deps.Pool, func(tx pgx.Tx) error {
		// Each new challenge invalidates earlier live ones for the phone.
		if _, err := tx.Exec(ctx, `UPDATE otp_challenges SET invalidated_at = $2
			WHERE phone_e164 = $1 AND purpose = 'LOGIN' AND verified_at IS NULL AND invalidated_at IS NULL`,
			in.Phone, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO otp_challenges
			(id, phone_e164, purpose, audience, channel, code_hmac, max_attempts, resend_count, expires_at, request_ip, created_at)
			VALUES ($1, $2, 'LOGIN', $3, 'SMS', $4, $5, $6, $7, $8::inet, $9)`,
			ch.ID, in.Phone, string(in.Audience), s.codeMAC(ch.ID, code), otpMaxAttempts, min(prior, 32767),
			ch.ExpiresAt, nullIfEmpty(in.IP), now)
		return err
	})
	if err != nil {
		return Challenge{}, err
	}
	if err := s.sender.SendOTP(ctx, in.Phone, code, in.Audience); err != nil {
		s.deps.Logger.ErrorContext(ctx, "otp send failed", "challenge_id", ch.ID, "error", err.Error())
		return Challenge{}, apperr.Unavailable("", "We could not send the code. Please try again.")
	}
	return ch, nil
}

// otpOutcome carries a business failure out of the verify transaction so the attempt counter
// commits instead of rolling back with the error.
type otpOutcome struct {
	fail *apperr.Error
	tok  Tokens
}

// VerifyOTP checks the code and, on success, creates or reuses the member and starts a session.
func (s *service) VerifyOTP(ctx context.Context, in VerifyOTPInput) (Tokens, error) {
	if !memberAudience(in.Audience) {
		return Tokens{}, apperr.Forbidden(apperr.CodeForbidden, "OTP sign-in is not available on this app.")
	}
	if !otpCodeRe.MatchString(in.Code) {
		return Tokens{}, apperr.Validation(apperr.FieldError{Field: "code", Code: "INVALID_CODE", Detail: "Enter the 6-digit code."})
	}
	in.DeviceInstallID = strings.TrimSpace(in.DeviceInstallID)
	if in.DeviceInstallID == "" || len(in.DeviceInstallID) > 100 {
		return Tokens{}, apperr.Validation(apperr.FieldError{Field: "deviceInstallId", Code: "REQUIRED", Detail: "1 to 100 characters"})
	}
	now := s.deps.Clock.Now()

	out, err := db.InTx(ctx, s.deps.Pool, func(tx pgx.Tx) (otpOutcome, error) {
		var (
			phone, audience         string
			mac                     []byte
			attempts, maxAttempts   int
			expiresAt               time.Time
			verifiedAt, invalidated *time.Time
		)
		err := tx.QueryRow(ctx, `SELECT phone_e164, audience, code_hmac, attempts, max_attempts, expires_at, verified_at, invalidated_at
			FROM otp_challenges WHERE id = $1 AND purpose = 'LOGIN' FOR UPDATE`, in.ChallengeID).
			Scan(&phone, &audience, &mac, &attempts, &maxAttempts, &expiresAt, &verifiedAt, &invalidated)
		if db.IsNoRows(err) || (err == nil && audience != string(in.Audience)) {
			return otpOutcome{fail: apperr.Unprocessable(apperr.CodeOTPInvalid, "That code is not correct.")}, nil
		}
		if err != nil {
			return otpOutcome{}, err
		}
		switch {
		case attempts >= maxAttempts:
			return otpOutcome{fail: apperr.Unprocessable(apperr.CodeOTPAttemptsExceeded, "Too many wrong attempts. Request a new code.")}, nil
		case verifiedAt != nil || invalidated != nil || !expiresAt.After(now):
			return otpOutcome{fail: apperr.Unprocessable(apperr.CodeOTPExpired, "This code has expired. Request a new one.")}, nil
		}
		failKey := s.phoneBucket("otp:fail:h:", phone)
		if fails, err := s.deps.RateLimit.Peek(ctx, failKey, time.Hour); err != nil {
			return otpOutcome{}, err
		} else if fails >= otpFailsPerHour {
			return otpOutcome{fail: apperr.TooManyRequests(apperr.CodeRateLimited, 3600)}, nil
		}

		if !auth.EqualMAC(mac, s.codeMAC(in.ChallengeID, in.Code)) {
			attempts++
			if _, err := tx.Exec(ctx, `UPDATE otp_challenges SET attempts = $2,
				invalidated_at = CASE WHEN $2 >= max_attempts THEN $3 ELSE invalidated_at END WHERE id = $1`,
				in.ChallengeID, attempts, now); err != nil {
				return otpOutcome{}, err
			}
			if _, err := s.deps.RateLimit.Allow(ctx, failKey, otpFailsPerHour, time.Hour); err != nil {
				return otpOutcome{}, err
			}
			if attempts >= maxAttempts {
				return otpOutcome{fail: apperr.Unprocessable(apperr.CodeOTPAttemptsExceeded, "Too many wrong attempts. Request a new code.")}, nil
			}
			return otpOutcome{fail: apperr.Unprocessable(apperr.CodeOTPInvalid, "That code is not correct.").
				With("attemptsLeft", maxAttempts-attempts)}, nil
		}
		if _, err := tx.Exec(ctx, `UPDATE otp_challenges SET verified_at = $2 WHERE id = $1`, in.ChallengeID, now); err != nil {
			return otpOutcome{}, err
		}

		userID, isNew, fail, err := s.findOrCreateMember(ctx, tx, phone, in.Audience, now)
		if err != nil || fail != nil {
			return otpOutcome{fail: fail}, err
		}
		tok, err := s.startSession(ctx, tx, userID, in, now)
		if err != nil {
			return otpOutcome{}, err
		}
		tok.IsNewUser = isNew
		return otpOutcome{tok: tok}, nil
	})
	if err != nil {
		return Tokens{}, err
	}
	if out.fail != nil {
		return Tokens{}, out.fail
	}
	return out.tok, nil
}

// findOrCreateMember resolves the member for a verified phone, creating the account on first
// login. On the customer app it also ensures the CUSTOMER role and profile; partner apps grant no
// role here (roles come from KYC approval, doc 12 §2.3).
func (s *service) findOrCreateMember(ctx context.Context, tx pgx.Tx, phone string, aud auth.Audience, now time.Time) (uuid.UUID, bool, *apperr.Error, error) {
	var (
		id     uuid.UUID
		status string
		isNew  bool
	)
	err := tx.QueryRow(ctx, `SELECT id, status FROM users
		WHERE phone_e164 = $1 AND kind = 'MEMBER' AND status NOT IN ('DETACHED','ANONYMIZED') FOR UPDATE`, phone).
		Scan(&id, &status)
	switch {
	case db.IsNoRows(err):
		id, status, isNew = s.deps.IDs.NewID(), "ACTIVE", true
		if _, err := tx.Exec(ctx, `INSERT INTO users (id, kind, phone_e164, phone_verified_at, status, last_active_at)
			VALUES ($1, 'MEMBER', $2, $3, 'ACTIVE', $3)`, id, phone, now); err != nil {
			return uuid.Nil, false, nil, err
		}
	case err != nil:
		return uuid.Nil, false, nil, err
	}
	if status != "ACTIVE" {
		return uuid.Nil, false, apperr.Forbidden(apperr.CodeAccountBlocked, "This account is not available. Contact support."), nil
	}
	if !isNew {
		if _, err := tx.Exec(ctx, `UPDATE users SET last_active_at = $2, phone_verified_at = COALESCE(phone_verified_at, $2)
			WHERE id = $1`, id, now); err != nil {
			return uuid.Nil, false, nil, err
		}
	}
	if aud == auth.AudienceCustomer {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (id, user_id, role, granted_at)
			SELECT $1, $2, 'CUSTOMER', $3 WHERE NOT EXISTS
			  (SELECT 1 FROM user_roles WHERE user_id = $2 AND role = 'CUSTOMER' AND revoked_at IS NULL)`,
			s.deps.IDs.NewID(), id, now); err != nil {
			return uuid.Nil, false, nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO customer_profiles (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, id); err != nil {
			return uuid.Nil, false, nil, err
		}
	}
	return id, isNew, nil, nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
