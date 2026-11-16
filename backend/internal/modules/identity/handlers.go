package identity

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/modules/users"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
)

func (s *service) registerRoutes(mux *httpx.Mux, _ *app.Deps) {
	mux.Handle("POST /api/v1/auth/otp/request", s.handleOTPRequest)
	mux.Handle("POST /api/v1/auth/otp/verify", s.handleOTPVerify)
	mux.Handle("POST /api/v1/auth/refresh", s.handleRefresh)
	mux.Handle("POST /api/v1/auth/logout", s.handleLogout)
}

type otpRequestBody struct {
	Phone        string  `json:"phone"`
	Channel      string  `json:"channel"`
	CaptchaToken *string `json:"captchaToken"`
}

type otpRequestResponse struct {
	ChallengeID        uuid.UUID `json:"challengeId"`
	ResendAfterSeconds int       `json:"resendAfterSeconds"`
	ExpiresAt          time.Time `json:"expiresAt"`
}

func (s *service) handleOTPRequest(w http.ResponseWriter, r *http.Request) {
	body, err := httpx.DecodeJSON[otpRequestBody](w, r, 4<<10)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if body.Channel != "SMS" {
		httpx.Error(w, r, apperr.Validation(apperr.FieldError{Field: "channel", Code: "INVALID_CHANNEL", Detail: "must be SMS"}))
		return
	}
	ch, err := s.StartOTP(r.Context(), StartOTPInput{Phone: body.Phone, Audience: httpx.AudienceFrom(r.Context()), IP: httpx.ClientIP(r)})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, otpRequestResponse{ChallengeID: ch.ID, ResendAfterSeconds: ch.ResendAfterSeconds, ExpiresAt: ch.ExpiresAt})
}

type otpVerifyBody struct {
	ChallengeID     uuid.UUID `json:"challengeId"`
	Code            string    `json:"code"`
	DeviceInstallID string    `json:"deviceInstallId"`
}

// meWithExpiry is schema Me including accessExpiresAt.
type meWithExpiry struct {
	users.Me
	AccessExpiresAt time.Time `json:"accessExpiresAt"`
}

type otpVerifyResponse struct {
	User            meWithExpiry `json:"user"`
	IsNewUser       bool         `json:"isNewUser"`
	AccessExpiresAt time.Time    `json:"accessExpiresAt"`
}

func (s *service) handleOTPVerify(w http.ResponseWriter, r *http.Request) {
	body, err := httpx.DecodeJSON[otpVerifyBody](w, r, 4<<10)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	aud := httpx.AudienceFrom(r.Context())
	tok, err := s.VerifyOTP(r.Context(), VerifyOTPInput{
		ChallengeID: body.ChallengeID, Code: body.Code, Audience: aud, DeviceInstallID: body.DeviceInstallID,
		IP: httpx.ClientIP(r), UserAgent: r.UserAgent(),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	usersSvc, ok := s.deps.Service(app.ServiceUsers).(users.Service)
	if !ok {
		httpx.Error(w, r, apperr.Unavailable("", "Profile service unavailable."))
		return
	}
	me, err := usersSvc.Me(r.Context(), tok.Principal.UserID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if tok.Principal.Context != "" {
		me.ActiveContext = &tok.Principal.Context
	}
	s.deps.Cookies.SetSession(w, aud, tok.AccessToken, tok.AccessExpiresAt, tok.RefreshToken, tok.RefreshMaxAge, s.deps.Clock.Now())
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, otpVerifyResponse{
		User: meWithExpiry{Me: me, AccessExpiresAt: tok.AccessExpiresAt}, IsNewUser: tok.IsNewUser, AccessExpiresAt: tok.AccessExpiresAt,
	})
}

func (s *service) handleRefresh(w http.ResponseWriter, r *http.Request) {
	aud := httpx.AudienceFrom(r.Context())
	tok, err := s.Refresh(r.Context(), s.deps.Cookies.RefreshToken(r), aud, httpx.ClientIP(r))
	if err != nil {
		// A lost race keeps the cookies: the winning request already set the new pair.
		if e, ok := apperr.As(err); ok && e.Status == http.StatusUnauthorized {
			s.deps.Cookies.Clear(w, aud)
		}
		httpx.Error(w, r, err)
		return
	}
	s.deps.Cookies.SetSession(w, aud, tok.AccessToken, tok.AccessExpiresAt, tok.RefreshToken, tok.RefreshMaxAge, s.deps.Clock.Now())
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, map[string]time.Time{"accessExpiresAt": tok.AccessExpiresAt})
}

func (s *service) handleLogout(w http.ResponseWriter, r *http.Request) {
	aud := httpx.AudienceFrom(r.Context())
	found := false
	var err error
	if p := httpx.Principal(r); p != nil {
		found, err = true, s.Logout(r.Context(), p.SessionID)
	} else {
		// The access token may have expired; the refresh cookie still identifies the session.
		found, err = s.LogoutByRefreshToken(r.Context(), s.deps.Cookies.RefreshToken(r))
	}
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	s.deps.Cookies.Clear(w, aud)
	if !found {
		httpx.Error(w, r, apperr.Unauthenticated(apperr.CodeUnauthenticated, "Authentication required."))
		return
	}
	httpx.NoContent(w)
}
