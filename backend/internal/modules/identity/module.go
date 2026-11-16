// Package identity owns phone-OTP auth, admin email+argon2id+TOTP login, access/refresh token
// issuance with reuse detection, sessions and RBAC subject resolution (charter §3, doc 12).
// Feature engineer A implements the service; this file fixes the public contract and wiring.
package identity

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the identity module's public API (charter §5). It also satisfies the httpx
// Authenticator/SessionChecker ports used by the middleware chain.
type Service interface {
	// StartOTP issues an OTP challenge for an India mobile and returns the challenge id.
	StartOTP(ctx context.Context, phoneE164 string, aud auth.Audience) (challengeID uuid.UUID, err error)
	// VerifyOTP checks an OTP and establishes a session, returning the issued tokens.
	VerifyOTP(ctx context.Context, challengeID uuid.UUID, code string) (Tokens, error)
	// Verify authenticates an access token for an audience (httpx.Authenticator).
	Verify(token string, aud auth.Audience) (*auth.Principal, error)
	// SessionValid reports whether a server-side session is still active (httpx.SessionChecker).
	SessionValid(sessionID uuid.UUID) bool
}

// Tokens is a freshly minted session's token pair and metadata.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	Principal    *auth.Principal
}

type service struct {
	deps   *app.Deps
	tokens *auth.Issuer
}

// New constructs the identity module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps, tokens: deps.Tokens}
	return &app.Module{Name: app.ServiceIdentity, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) StartOTP(context.Context, string, auth.Audience) (uuid.UUID, error) {
	return uuid.Nil, apperr.NotImplemented("identity.StartOTP")
}

func (s *service) VerifyOTP(context.Context, uuid.UUID, string) (Tokens, error) {
	return Tokens{}, apperr.NotImplemented("identity.VerifyOTP")
}

// Verify delegates to the platform issuer so the middleware chain can authenticate before the
// full service lands.
func (s *service) Verify(token string, aud auth.Audience) (*auth.Principal, error) {
	if s.tokens == nil {
		return nil, auth.ErrTokenInvalid
	}
	return s.tokens.Verify(token, aud)
}

// SessionValid is a stub that trusts the token until session revocation is wired (AUTH-D05).
func (s *service) SessionValid(uuid.UUID) bool { return true }

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps) {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}

var _ httpx.Authenticator = (*service)(nil)
var _ httpx.SessionChecker = (*service)(nil)
