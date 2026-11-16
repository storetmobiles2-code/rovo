//go:build integration

package identity_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/storetmobiles2-code/rovo/backend/internal/app/apptest"
)

const phone = "+919000000012"

func otpRequest(c *apptest.Client, p string) *apptest.Response {
	return c.Do(http.MethodPost, "/api/v1/auth/otp/request", map[string]any{"phone": p, "channel": "SMS"})
}

func TestOTPLoginCreatesCustomerSession(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.CustomerHost)

	if res := c.Do(http.MethodGet, "/api/v1/me", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous /me = %d, want 401", res.Code)
	}
	res := c.Login(phone)
	if res.JSON["isNewUser"] != true {
		t.Errorf("isNewUser = %v, want true", res.JSON["isNewUser"])
	}
	at := c.Cookie("rovo_at")
	rt := c.Cookie("rovo_rt")
	if at == nil || rt == nil || !at.HttpOnly || !rt.HttpOnly || rt.Path != "/api/v1/auth" {
		t.Fatalf("session cookies not set correctly: at=%+v rt=%+v", at, rt)
	}

	me := c.Do(http.MethodGet, "/api/v1/me", nil)
	if me.Code != http.StatusOK {
		t.Fatalf("/me = %d %s", me.Code, me.Body)
	}
	if me.JSON["phoneMasked"] != "+91••••••••12" || me.JSON["kind"] != "MEMBER" {
		t.Errorf("me = %v", me.JSON)
	}
	roles, _ := me.JSON["roles"].([]any)
	if len(roles) != 1 || roles[0].(map[string]any)["role"] != "CUSTOMER" {
		t.Errorf("roles = %v, want one CUSTOMER", roles)
	}

	// The stored code is an HMAC, never the code itself.
	var n int
	if err := env.DB.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM otp_challenges WHERE verified_at IS NOT NULL AND length(code_hmac) = 32`).Scan(&n); err != nil || n != 1 {
		t.Errorf("verified challenges = %d (err %v), want 1", n, err)
	}

	// A second login is the same account.
	c2 := env.Client(apptest.CustomerHost)
	if _, err := env.DB.Pool.Exec(context.Background(), `UPDATE otp_challenges SET created_at = created_at - interval '10 minutes'`); err != nil {
		t.Fatal(err)
	}
	if res := c2.Login(phone); res.JSON["isNewUser"] != false {
		t.Errorf("second login isNewUser = %v, want false", res.JSON["isNewUser"])
	}
	if err := env.DB.Pool.QueryRow(context.Background(), `SELECT count(*) FROM users`).Scan(&n); err != nil || n != 1 {
		t.Errorf("users = %d (err %v), want 1", n, err)
	}
}

func TestOTPRequestValidation(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.CustomerHost)
	for _, p := range []string{"+14155550123", "+915000000012", "9000000012", ""} {
		if res := otpRequest(c, p); res.Code != http.StatusBadRequest {
			t.Errorf("phone %q = %d, want 400", p, res.Code)
		}
	}
	// Admins do not sign in by OTP.
	if res := otpRequest(env.Client(apptest.AdminHost), phone); res.Code != http.StatusForbidden {
		t.Errorf("admin host otp = %d, want 403", res.Code)
	}
	// Unsafe requests need the client header (CSRF defence).
	raw := env.Client("localhost:9") // unknown host: no header added, 421
	if res := otpRequest(raw, phone); res.Code != http.StatusMisdirectedRequest {
		t.Errorf("unknown host = %d, want 421", res.Code)
	}
}

func TestOTPResendCooldown(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.CustomerHost)
	first := otpRequest(c, phone)
	if first.Code != http.StatusAccepted || first.JSON["resendAfterSeconds"] != float64(30) {
		t.Fatalf("first = %d %v", first.Code, first.JSON)
	}
	again := otpRequest(c, phone)
	if again.Code != http.StatusTooManyRequests || again.JSON["code"] != "OTP_RESEND_TOO_SOON" || again.Header().Get("Retry-After") == "" {
		t.Fatalf("immediate resend = %d %v", again.Code, again.JSON)
	}
	// After the cooldown a resend works and kills the earlier challenge.
	if _, err := env.DB.Pool.Exec(context.Background(), `UPDATE otp_challenges SET created_at = created_at - interval '31 seconds'`); err != nil {
		t.Fatal(err)
	}
	oldCode := env.OTP(phone)
	second := otpRequest(c, phone)
	if second.Code != http.StatusAccepted {
		t.Fatalf("resend after cooldown = %d %s", second.Code, second.Body)
	}
	res := c.Do(http.MethodPost, "/api/v1/auth/otp/verify", map[string]any{
		"challengeId": first.JSON["challengeId"], "code": oldCode, "deviceInstallId": "d1"})
	if res.Code != http.StatusUnprocessableEntity || res.JSON["code"] != "OTP_EXPIRED" {
		t.Errorf("old challenge verify = %d %v, want 422 OTP_EXPIRED", res.Code, res.JSON)
	}
}

func TestOTPWrongCodeAttempts(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.CustomerHost)
	ch := otpRequest(c, phone)
	good := env.OTP(phone)
	bad := "000000"
	if good == bad {
		bad = "000001"
	}
	verify := func(code string) *apptest.Response {
		return c.Do(http.MethodPost, "/api/v1/auth/otp/verify", map[string]any{
			"challengeId": ch.JSON["challengeId"], "code": code, "deviceInstallId": "d1"})
	}
	for i := 1; i <= 4; i++ {
		res := verify(bad)
		if res.Code != http.StatusUnprocessableEntity || res.JSON["code"] != "OTP_INVALID" || res.JSON["attemptsLeft"] != float64(5-i) {
			t.Fatalf("wrong attempt %d = %d %v", i, res.Code, res.JSON)
		}
	}
	if res := verify(bad); res.JSON["code"] != "OTP_ATTEMPTS_EXCEEDED" {
		t.Fatalf("5th wrong attempt = %d %v", res.Code, res.JSON)
	}
	// The right code no longer works once attempts are used up.
	if res := verify(good); res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("correct code after lockout = %d %v", res.Code, res.JSON)
	}
	if c.Cookie("rovo_at") != nil {
		t.Error("session cookie set despite failed verification")
	}
}

func TestOTPChallengeIsBoundToAudience(t *testing.T) {
	env := apptest.New(t)
	ch := otpRequest(env.Client(apptest.CustomerHost), phone)
	res := env.Client(apptest.RiderHost).Do(http.MethodPost, "/api/v1/auth/otp/verify", map[string]any{
		"challengeId": ch.JSON["challengeId"], "code": env.OTP(phone), "deviceInstallId": "d1"})
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("cross-app verify = %d, want 422", res.Code)
	}
}

func TestBlockedAccountCannotLogIn(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.CustomerHost)
	c.Login(phone)
	ctx := context.Background()
	if _, err := env.DB.Pool.Exec(ctx, `UPDATE users SET status = 'BLOCKED'`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.DB.Pool.Exec(ctx, `UPDATE otp_challenges SET created_at = created_at - interval '10 minutes'`); err != nil {
		t.Fatal(err)
	}
	c2 := env.Client(apptest.CustomerHost)
	ch := otpRequest(c2, phone)
	if ch.Code != http.StatusAccepted {
		t.Fatalf("blocked account must get the same 202, got %d", ch.Code)
	}
	res := c2.Do(http.MethodPost, "/api/v1/auth/otp/verify", map[string]any{
		"challengeId": ch.JSON["challengeId"], "code": env.OTP(phone), "deviceInstallId": "d2"})
	if res.Code != http.StatusForbidden || res.JSON["code"] != "ACCOUNT_BLOCKED" {
		t.Fatalf("blocked verify = %d %v", res.Code, res.JSON)
	}
	// The existing session can no longer refresh.
	if res := c.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("blocked refresh = %d, want 401", res.Code)
	}
}

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.CustomerHost)
	c.Login(phone)
	ctx := context.Background()
	old := c.Cookie("rovo_rt").Value

	res := c.Do(http.MethodPost, "/api/v1/auth/refresh", nil)
	if res.Code != http.StatusOK || res.JSON["accessExpiresAt"] == nil {
		t.Fatalf("refresh = %d %s", res.Code, res.Body)
	}
	rotated := c.Cookie("rovo_rt").Value
	if rotated == old {
		t.Fatal("refresh token was not rotated")
	}

	// Replay inside the race window: 409, session survives.
	stale := env.Client(apptest.CustomerHost)
	stale.SetCookie("rovo_rt", old)
	if res := stale.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusConflict || res.JSON["code"] != "REFRESH_RACE" {
		t.Fatalf("replay in race window = %d %v", res.Code, res.JSON)
	}
	if res := c.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusOK {
		t.Fatalf("refresh after benign race = %d", res.Code)
	}

	// Replay after the window: the whole session is revoked.
	if _, err := env.DB.Pool.Exec(ctx, `UPDATE refresh_tokens SET used_at = used_at - interval '1 minute' WHERE used_at IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	stale.SetCookie("rovo_rt", old)
	if res := stale.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusUnauthorized || res.JSON["code"] != "SESSION_REVOKED" {
		t.Fatalf("replay after window = %d %v", res.Code, res.JSON)
	}
	var reason string
	if err := env.DB.Pool.QueryRow(ctx, `SELECT revoke_reason FROM sessions`).Scan(&reason); err != nil || reason != "REFRESH_REUSE" {
		t.Fatalf("revoke_reason = %q (err %v)", reason, err)
	}
	if res := c.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("legitimate token after family revoke = %d, want 401", res.Code)
	}
}

func TestRefreshRejectsOtherAppAndGarbage(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.CustomerHost)
	c.Login(phone)
	rider := env.Client(apptest.RiderHost)
	rider.SetCookie("rovo_rt", c.Cookie("rovo_rt").Value)
	if res := rider.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("customer refresh token on rider app = %d, want 401", res.Code)
	}
	anon := env.Client(apptest.CustomerHost)
	if res := anon.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("refresh with no cookie = %d, want 401", res.Code)
	}
	anon.SetCookie("rovo_rt", "rvr_not-a-real-token")
	if res := anon.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("refresh with garbage = %d, want 401", res.Code)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.RiderHost)
	c.Login(phone)
	rt := c.Cookie("rovo_rt").Value
	at := c.Cookie("rovo_at").Value

	if res := c.Do(http.MethodPost, "/api/v1/auth/logout", nil); res.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %s", res.Code, res.Body)
	}
	if c.Cookie("rovo_at") != nil || c.Cookie("rovo_rt") != nil {
		t.Error("cookies not cleared on logout")
	}
	// Old tokens are dead: the rider app re-checks the session on every request.
	old := env.Client(apptest.RiderHost)
	old.SetCookie("rovo_at", at)
	if res := old.Do(http.MethodGet, "/api/v1/me", nil); res.Code != http.StatusUnauthorized || res.JSON["code"] != "SESSION_REVOKED" {
		t.Fatalf("/me with revoked session = %d %v", res.Code, res.JSON)
	}
	old = env.Client(apptest.RiderHost)
	old.SetCookie("rovo_rt", rt)
	if res := old.Do(http.MethodPost, "/api/v1/auth/refresh", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout = %d, want 401", res.Code)
	}
	if res := env.Client(apptest.RiderHost).Do(http.MethodPost, "/api/v1/auth/logout", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous logout = %d, want 401", res.Code)
	}
}

func TestPartnerAppLoginGrantsNoRole(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.RiderHost)
	c.Login(phone)
	me := c.Do(http.MethodGet, "/api/v1/me", nil)
	if roles, _ := me.JSON["roles"].([]any); me.Code != http.StatusOK || len(roles) != 0 {
		t.Fatalf("rider-app login: /me = %d roles %v, want no roles", me.Code, me.JSON["roles"])
	}
}

func TestUpdateMe(t *testing.T) {
	env := apptest.New(t)
	c := env.Client(apptest.CustomerHost)
	c.Login(phone)
	res := c.Do(http.MethodPatch, "/api/v1/me", map[string]any{"fullName": "  Ravi Kumar ", "preferredLocale": "te"})
	if res.Code != http.StatusOK || res.JSON["fullName"] != "Ravi Kumar" || res.JSON["preferredLocale"] != "te" {
		t.Fatalf("patch = %d %v", res.Code, res.JSON)
	}
	if res := c.Do(http.MethodPatch, "/api/v1/me", map[string]any{"preferredLocale": "fr"}); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("unsupported locale = %d, want 422", res.Code)
	}
	if res := c.Do(http.MethodPatch, "/api/v1/me", map[string]any{"phone": "+919999999999"}); res.Code != http.StatusBadRequest {
		t.Errorf("unknown field = %d, want 400", res.Code)
	}
	res = c.Do(http.MethodPatch, "/api/v1/me", map[string]any{"fullName": nil})
	if res.Code != http.StatusOK || res.JSON["fullName"] != nil || res.JSON["preferredLocale"] != "te" {
		t.Fatalf("clear name = %d %v", res.Code, res.JSON)
	}
}
