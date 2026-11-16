package auth

import (
	"errors"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTP parameters (12 §3.2): SHA-1, 6 digits, 30 s step, ±1 step window, replay guard.
const (
	totpPeriod = 30
	totpDigits = otp.DigitsSix
)

// ErrTOTPInvalid is returned for a wrong or replayed code.
var ErrTOTPInvalid = errors.New("auth: invalid totp code")

// TOTPEnrolment is shown once to the admin.
type TOTPEnrolment struct {
	Secret string // base32
	URI    string // otpauth:// for the QR code
}

// NewTOTPSecret generates a 160-bit secret for account at issuer.
func NewTOTPSecret(issuer, account string) (TOTPEnrolment, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: issuer, AccountName: account, Period: totpPeriod, Digits: totpDigits,
		Algorithm: otp.AlgorithmSHA1, SecretSize: 20,
	})
	if err != nil {
		return TOTPEnrolment{}, err
	}
	return TOTPEnrolment{Secret: key.Secret(), URI: key.URL()}, nil
}

// ValidateTOTP checks code at now within ±1 step. It returns the matched step, which the caller
// stores as admin_credentials.totp_last_step; any step ≤ lastStep is rejected as a replay.
func ValidateTOTP(secret, code string, now time.Time, lastStep int64) (int64, error) {
	if len(code) != 6 {
		return 0, ErrTOTPInvalid
	}
	current := now.Unix() / totpPeriod
	for _, delta := range []int64{0, -1, 1} {
		step := current + delta
		if step <= lastStep {
			continue
		}
		want, err := totp.GenerateCodeCustom(secret, time.Unix(step*totpPeriod, 0), totp.ValidateOpts{
			Period: totpPeriod, Digits: totpDigits, Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil {
			return 0, err
		}
		if EqualMAC([]byte(want), []byte(code)) {
			return step, nil
		}
	}
	return 0, ErrTOTPInvalid
}

// TOTPCode computes the code for secret at t (seed output and tests).
func TOTPCode(secret string, t time.Time) (string, error) {
	return totp.GenerateCodeCustom(secret, t, totp.ValidateOpts{Period: totpPeriod, Digits: totpDigits, Algorithm: otp.AlgorithmSHA1})
}
