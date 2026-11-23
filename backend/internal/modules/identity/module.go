// Package identity owns phone-OTP auth, admin email+argon2id+TOTP login, access/refresh token
// issuance with reuse detection, sessions and RBAC subject resolution (charter §3, doc 12).
//
// Implemented: member phone-OTP login, refresh rotation with reuse detection, logout and the
// server-side session check. Admin password+TOTP login is not implemented yet.
package identity

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the identity module's public API (charter §5). It also satisfies the httpx
// Authenticator/SessionChecker ports used by the middleware chain.
type Service interface {
	// StartOTP issues a login OTP challenge for an India mobile and sends the code.
	StartOTP(ctx context.Context, in StartOTPInput) (Challenge, error)
	// VerifyOTP checks an OTP and establishes a session, returning the issued tokens.
	VerifyOTP(ctx context.Context, in VerifyOTPInput) (Tokens, error)
	// Refresh rotates a refresh token and mints a new access token (12 §4.3).
	Refresh(ctx context.Context, refreshToken string, aud auth.Audience, ip string) (Tokens, error)
	// Logout revokes a session; unknown or already-revoked sessions are a no-op.
	Logout(ctx context.Context, sessionID uuid.UUID) error
	// LogoutByRefreshToken revokes the session a refresh token belongs to and reports whether one
	// was found (used when the access token has already expired).
	LogoutByRefreshToken(ctx context.Context, refreshToken string) (bool, error)
	// Verify authenticates an access token for an audience (httpx.Authenticator).
	Verify(token string, aud auth.Audience) (*auth.Principal, error)
	// SessionValid reports whether a server-side session is still active (httpx.SessionChecker).
	SessionValid(sessionID uuid.UUID) bool
}

// OTPSender delivers a one-time code to a phone. The composition root supplies the adapter
// (fake outbox locally and in CI; the DLT SMS provider in production).
type OTPSender interface {
	SendOTP(ctx context.Context, phoneE164, code string, aud auth.Audience) error
}

// StartOTPInput is a login OTP request.
type StartOTPInput struct {
	Phone    string
	Audience auth.Audience
	IP       string // caller IP for rate limiting; may be empty
}

// Challenge is an issued OTP challenge.
type Challenge struct {
	ID                 uuid.UUID
	ExpiresAt          time.Time
	ResendAfterSeconds int
}

// VerifyOTPInput is an OTP verification attempt.
type VerifyOTPInput struct {
	ChallengeID     uuid.UUID
	Code            string
	Audience        auth.Audience
	DeviceInstallID string
	IP              string
	UserAgent       string
}

// Tokens is a freshly minted session's token pair and metadata.
type Tokens struct {
	AccessToken     string
	AccessExpiresAt time.Time
	RefreshToken    string
	// RefreshMaxAge is the refresh cookie lifetime: the time left until the session's absolute expiry.
	RefreshMaxAge time.Duration
	Principal     *auth.Principal
	IsNewUser     bool
}

type service struct {
	deps   *app.Deps
	tokens *auth.Issuer
	sender OTPSender
	otpKey []byte // HMAC pepper for OTP codes; derived, never stored in the DB
	limKey []byte // HMAC key for phone-based rate-limit bucket names
}

// New constructs the identity module. sender may be nil, in which case OTP requests fail closed.
func New(deps *app.Deps, sender OTPSender) *app.Module {
	s := &service{deps: deps, tokens: deps.Tokens, sender: sender}
	if deps.Config != nil {
		s.otpKey = deps.Config.DeriveKey("otp-pepper")
		s.limKey = deps.Config.DeriveKey("phone-hmac")
	}
	return &app.Module{Name: app.ServiceIdentity, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

// Verify delegates to the platform issuer.
func (s *service) Verify(token string, aud auth.Audience) (*auth.Principal, error) {
	if s.tokens == nil {
		return nil, auth.ErrTokenInvalid
	}
	return s.tokens.Verify(token, aud)
}

// SessionValid reports whether the session exists, is not revoked and is inside its absolute
// lifetime (AUTH-D05). It fails closed: a lookup error counts as invalid.
func (s *service) SessionValid(sessionID uuid.UUID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var ok bool
	err := s.deps.Pool.QueryRow(ctx, `SELECT revoked_at IS NULL AND absolute_expires_at > $2
		FROM sessions WHERE id = $1`, sessionID, s.deps.Clock.Now()).Scan(&ok)
	return err == nil && ok
}

func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}

var _ httpx.Authenticator = (*service)(nil)
var _ httpx.SessionChecker = (*service)(nil)
