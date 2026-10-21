package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/config"
)

func TestDecodeJSON(t *testing.T) {
	type body struct {
		Name string `json:"name"`
	}
	req := func(s string) (*httptest.ResponseRecorder, *http.Request) {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(s))
		r.Header.Set("Content-Type", "application/json")
		return httptest.NewRecorder(), r
	}
	w, r := req(`{"name":"ok"}`)
	v, err := DecodeJSON[body](w, r, 1024)
	if err != nil || v.Name != "ok" {
		t.Fatalf("decode: %v %+v", err, v)
	}
	w, r = req(`{"name":"x","extra":1}`)
	if _, err := DecodeJSON[body](w, r, 1024); !apperr.IsCode(err, apperr.CodeValidationFailed) {
		t.Fatalf("unknown field should fail: %v", err)
	}
	w, r = req(strings.Repeat("a", 5000))
	if _, err := DecodeJSON[body](w, r, 64); err == nil {
		t.Fatal("oversized body should fail")
	}
	w, r = req(`{"name":1}`)
	if _, err := DecodeJSON[body](w, r, 1024); err == nil {
		t.Fatal("type error should fail")
	}
	w, r = req(`{}{}`)
	if _, err := DecodeJSON[body](w, r, 1024); err == nil {
		t.Fatal("trailing data should fail")
	}
}

func TestErrorRenders9457(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	e := apperr.Conflict(apperr.CodeQuoteChanged, "price changed").With("diff", []string{"a"})
	Error(w, r, e)
	res := w.Result()
	if res.StatusCode != http.StatusConflict || res.Header.Get("Content-Type") != problemContentType {
		t.Fatalf("status/ct: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body["code"] != "QUOTE_CHANGED" || body["type"] != "/problems/quote-changed" || body["instance"] != "/api/v1/orders" {
		t.Fatalf("body %v", body)
	}
	if _, ok := body["diff"]; !ok {
		t.Fatal("extension not rendered")
	}
	if _, ok := body["fieldErrors"]; !ok {
		t.Fatal("fieldErrors must always be present")
	}
}

func TestErrorRetryAfter(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	Error(w, r, apperr.TooManyRequests(apperr.CodeRateLimited, 5))
	if w.Result().Header.Get("Retry-After") != "5" {
		t.Fatal("retry-after header")
	}
}

func TestResolveAudienceAndHelpers(t *testing.T) {
	auds, err := config.ParseAppHosts("app.rovo.in=customer,admin.rovo.in=admin,api.rovo.in=webhook")
	if err != nil {
		t.Fatal(err)
	}
	mw := Audiences(auds).ResolveAudience()
	run := func(host string) (int, auth.Audience) {
		var got auth.Audience
		h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = AudienceFrom(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = host
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Result().StatusCode, got
	}
	if code, aud := run("app.rovo.in"); code != 200 || aud != auth.AudienceCustomer {
		t.Fatalf("customer host: %d %s", code, aud)
	}
	if code, aud := run("admin.rovo.in:443"); code != 200 || aud != auth.AudienceAdmin {
		t.Fatalf("port stripping: %d %s", code, aud)
	}
	if code, _ := run("evil.example.com"); code != http.StatusMisdirectedRequest {
		t.Fatalf("unknown host should be 421, got %d", code)
	}
}

func TestRequireRoles(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
	base := httptest.NewRequest(http.MethodGet, "/", nil)
	// Anonymous.
	w := httptest.NewRecorder()
	Require(auth.RoleAdminOps)(ok)(w, base)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatal("anon must be 401")
	}
	// Wrong role.
	w = httptest.NewRecorder()
	r := base.WithContext(auth.WithPrincipal(base.Context(), &auth.Principal{Roles: []auth.Role{auth.RoleCustomer}}))
	Require(auth.RoleAdminOps)(ok)(w, r)
	if w.Result().StatusCode != http.StatusForbidden {
		t.Fatal("wrong role must be 403")
	}
	// Right role.
	w = httptest.NewRecorder()
	r = base.WithContext(auth.WithPrincipal(base.Context(), &auth.Principal{Roles: []auth.Role{auth.RoleAdminOps}}))
	Require(auth.RoleAdminOps)(ok)(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("right role must pass")
	}
}

func TestRequireClientHeader(t *testing.T) {
	auds, _ := config.ParseAppHosts("app.rovo.in=customer")
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := Chain(ok, Audiences(auds).ResolveAudience(), RequireClientHeader())
	// Safe method: no header needed.
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Host = "app.rovo.in"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("GET must pass without header")
	}
	// Unsafe method without header: rejected.
	r = httptest.NewRequest(http.MethodPost, "/", nil)
	r.Host = "app.rovo.in"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("POST without client header must be 400, got %d", w.Result().StatusCode)
	}
	// With header: passes.
	r = httptest.NewRequest(http.MethodPost, "/", nil)
	r.Host = "app.rovo.in"
	r.Header.Set("X-Rovo-Client", "customer-web")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("POST with client header must pass")
	}
}

func TestOriginVerify(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := OriginVerify("s3cr3t")(ok)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusForbidden {
		t.Fatal("missing origin secret must be 403")
	}
	r.Header.Set("X-Rovo-Origin-Verify", "s3cr3t")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("correct secret must pass")
	}
	// Disabled when empty.
	w = httptest.NewRecorder()
	OriginVerify("")(ok).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("empty secret disables check")
	}
}

func TestRequestHash(t *testing.T) {
	a := RequestHash("POST", "/orders", []byte(`{"q":1}`))
	b := RequestHash("POST", "/orders", []byte(`{"q":1}`))
	c := RequestHash("POST", "/orders", []byte(`{"q":2}`))
	if !bytes.Equal(a, b) || bytes.Equal(a, c) {
		t.Fatal("request hash")
	}
}

var _ = io.Discard
