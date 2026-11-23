// Package apptest builds the fully wired application against a throwaway migrated database so
// module tests can exercise real HTTP routes through the same middleware chain as production.
package apptest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rovoapp "github.com/storetmobiles2-code/rovo/backend/internal/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/config"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/testutil"
	"github.com/storetmobiles2-code/rovo/backend/internal/seed"
)

// Hosts served by the default dev host table (config.DefaultDevHosts).
const (
	CustomerHost   = "localhost:5173"
	RestaurantHost = "localhost:5174"
	RiderHost      = "localhost:5175"
	AdminHost      = "localhost:5176"
)

// Env is a wired application over a fresh database.
type Env struct {
	T       testing.TB
	App     *rovoapp.App
	DB      *testutil.DB
	Config  *config.Config
	Handler http.Handler
}

// New builds the app (api role, insert-only queue) on a new migrated database. The test is
// skipped when ROVO_TEST_DATABASE_URL is unset.
func New(t testing.TB) *Env {
	t.Helper()
	d := testutil.NewDB(t)
	cfg, err := config.Load(func(k string) string {
		switch k {
		case "ROVO_ENV":
			return "test"
		case "ROVO_DATABASE_URL":
			return d.URL
		}
		return ""
	})
	if err != nil {
		t.Fatalf("apptest: config: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a, err := rovoapp.New(cfg, logger, d.Pool, rovoapp.Options{})
	if err != nil {
		t.Fatalf("apptest: app: %v", err)
	}
	mux := httpx.NewMux(a.PublicMiddleware()...)
	a.RegisterRoutes(mux)
	return &Env{T: t, App: a, DB: d, Config: cfg, Handler: mux}
}

// Seed loads the Mahabubnagar demo dataset (city, zone, localities, restaurants, menus, fees).
func (e *Env) Seed() *seed.Result {
	e.T.Helper()
	fields, err := auth.NewFieldKeyring(e.Config.FieldKeyID(), map[string][]byte{e.Config.FieldKeyID(): e.Config.FieldEncryptionKey})
	if err != nil {
		e.T.Fatalf("apptest: keyring: %v", err)
	}
	res, err := seed.Run(context.Background(), e.DB.Pool, seed.Options{
		Config: e.Config, Passwords: auth.NewPasswordHasher(auth.DefaultArgon2, 1), Fields: fields, Now: time.Now().UTC(),
	})
	if err != nil {
		e.T.Fatalf("apptest: seed: %v", err)
	}
	return res
}

// Client is a cookie-keeping HTTP client bound to one app host.
type Client struct {
	env     *Env
	Host    string
	cookies map[string]*http.Cookie
}

// Client returns a client for host (use the *Host constants).
func (e *Env) Client(host string) *Client {
	return &Client{env: e, Host: host, cookies: map[string]*http.Cookie{}}
}

// Response is a recorded response with its decoded JSON body (nil when not an object).
type Response struct {
	*httptest.ResponseRecorder
	JSON map[string]any
}

// Cookie returns the stored cookie by name, or nil.
func (c *Client) Cookie(name string) *http.Cookie { return c.cookies[name] }

// SetCookie overrides a stored cookie value (to replay an old token).
func (c *Client) SetCookie(name, value string) {
	c.cookies[name] = &http.Cookie{Name: name, Value: value}
}

// Do sends a request with the client header for the host's audience and stores returned cookies.
// A non-nil body is sent as JSON.
func (c *Client) Do(method, path string, body any) *Response {
	c.env.T.Helper()
	return c.do(method, path, body, nil)
}

// DoWithHeader is Do for a body-less request with one extra request header.
func (c *Client) DoWithHeader(method, path, header, value string) *Response {
	c.env.T.Helper()
	return c.do(method, path, nil, http.Header{header: {value}})
}

func (c *Client) do(method, path string, body any, extra http.Header) *Response {
	c.env.T.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.env.T.Fatalf("apptest: marshal: %v", err)
		}
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Host = c.Host
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if aud, ok := c.env.Config.AppHosts[c.Host]; ok {
		req.Header.Set("X-Rovo-Client", aud.ClientHeaderValue())
	}
	for k, v := range extra {
		req.Header[k] = v
	}
	for _, ck := range c.cookies {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	c.env.Handler.ServeHTTP(rec, req)
	for _, ck := range rec.Result().Cookies() {
		if ck.MaxAge < 0 {
			delete(c.cookies, ck.Name)
			continue
		}
		c.cookies[ck.Name] = ck
	}
	out := &Response{ResponseRecorder: rec}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.JSON)
	return out
}

// OTP returns the latest login code sent to phone through the fake provider.
func (e *Env) OTP(phone string) string {
	e.T.Helper()
	m, ok := e.App.Outbox.Latest("OTP", phone)
	if !ok {
		e.T.Fatalf("apptest: no OTP sent to %s", phone)
	}
	return m.Code
}

// Login signs phone in on the client's host and returns the verify response.
func (c *Client) Login(phone string) *Response {
	c.env.T.Helper()
	res := c.Do(http.MethodPost, "/api/v1/auth/otp/request", map[string]any{"phone": phone, "channel": "SMS"})
	if res.Code != http.StatusAccepted {
		c.env.T.Fatalf("apptest: otp request = %d %s", res.Code, res.Body)
	}
	res = c.Do(http.MethodPost, "/api/v1/auth/otp/verify", map[string]any{
		"challengeId": res.JSON["challengeId"], "code": c.env.OTP(phone), "deviceInstallId": "test-" + phone + "-" + c.Host,
	})
	if res.Code != http.StatusOK {
		c.env.T.Fatalf("apptest: otp verify = %d %s", res.Code, res.Body)
	}
	return res
}
