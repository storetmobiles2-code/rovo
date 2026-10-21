package auth

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
)

func newIssuer(t *testing.T, clk clock.Clock) (*Issuer, ed25519.PrivateKey) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ks, err := NewKeyset(priv)
	if err != nil {
		t.Fatal(err)
	}
	return NewIssuer(ks, "https://rovo.test", clk), priv
}

func TestJWTRoundTrip(t *testing.T) {
	clk := clock.NewFake(time.Date(2026, 10, 4, 13, 0, 0, 0, time.UTC))
	iss, _ := newIssuer(t, clk)
	uid, sid, city := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	tok, exp, err := iss.Issue(TokenInput{UserID: uid, SessionID: sid, Audience: AudienceAdmin,
		Roles: []Role{RoleAdminOps}, CityScopes: []uuid.UUID{city}, AMR: []string{"pwd", "totp"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := clk.Now().Add(AccessTTLAdmin); !exp.Equal(want) {
		t.Fatalf("admin ttl: exp %v want %v", exp, want)
	}
	p, err := iss.Verify(tok, AudienceAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if p.UserID != uid || p.SessionID != sid || !p.HasRole(RoleAdminOps) || !p.InCityScope(city) || p.InCityScope(uid) {
		t.Fatalf("principal mismatch: %+v", p)
	}
	if _, err := iss.Verify(tok, AudienceCustomer); err != ErrTokenAudienceMismatch {
		t.Fatalf("audience: %v", err)
	}
	clk.Advance(AccessTTLAdmin + leeway + time.Second)
	if _, err := iss.Verify(tok, AudienceAdmin); err != ErrTokenExpired {
		t.Fatalf("expiry: %v", err)
	}
}

func TestJWTRejectsForgeries(t *testing.T) {
	clk := clock.NewFake(time.Now())
	iss, _ := newIssuer(t, clk)
	other, _ := newIssuer(t, clk)
	tok, _, err := other.Issue(TokenInput{UserID: uuid.New(), SessionID: uuid.New(), Audience: AudienceCustomer})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := iss.Verify(tok, AudienceCustomer); err != ErrTokenInvalid {
		t.Fatalf("unknown kid must fail: %v", err)
	}
	// alg=none / HS256 must be rejected even with a known kid.
	c := jwt.MapClaims{"sub": uuid.NewString(), "aud": "customer", "iss": "https://rovo.test",
		"exp": clk.Now().Add(time.Minute).Unix(), "iat": clk.Now().Unix(), "sid": uuid.NewString()}
	h := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	h.Header["kid"] = iss.keys.ActiveKID()
	h.Header["typ"] = tokenType
	hs, _ := h.SignedString([]byte("secret"))
	if _, err := iss.Verify(hs, AudienceCustomer); err != ErrTokenInvalid {
		t.Fatalf("HS256 must fail: %v", err)
	}
	if _, err := iss.Verify(strings.Repeat("a", 3000), AudienceCustomer); err != ErrTokenInvalid {
		t.Fatal("oversized token must fail")
	}
	// Tampered payload.
	good, _, _ := iss.Issue(TokenInput{UserID: uuid.New(), SessionID: uuid.New(), Audience: AudienceCustomer})
	parts := strings.Split(good, ".")
	parts[1] = parts[1][:len(parts[1])-2] + "AA"
	if _, err := iss.Verify(strings.Join(parts, "."), AudienceCustomer); err != ErrTokenInvalid {
		t.Fatalf("tampered: %v", err)
	}
}

func TestKeyRotationVerifiesPrevious(t *testing.T) {
	clk := clock.NewFake(time.Now())
	oldIss, oldPriv := newIssuer(t, clk)
	tok, _, _ := oldIss.Issue(TokenInput{UserID: uuid.New(), SessionID: uuid.New(), Audience: AudienceRider})
	_, newPriv, _ := ed25519.GenerateKey(rand.Reader)
	ks, err := NewKeyset(newPriv, oldPriv.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewIssuer(ks, "https://rovo.test", clk).Verify(tok, AudienceRider); err != nil {
		t.Fatalf("previous key should verify: %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	tok, h := NewRefreshToken()
	if !LooksLikeRefreshToken(tok) || !bytes.Equal(h, HashToken(tok)) || len(h) != 32 {
		t.Fatalf("refresh token shape %q", tok)
	}
	tok2, _ := NewRefreshToken()
	if tok == tok2 {
		t.Fatal("tokens must differ")
	}
}

func TestPasswordHash(t *testing.T) {
	fast := Argon2Params{MemoryKiB: 8 * 1024, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}
	h := NewPasswordHasher(fast, 2)
	ctx := context.Background()
	phc, err := h.Hash(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(phc, "$argon2id$v=19$m=8192,t=1,p=1$") {
		t.Fatalf("phc %q", phc)
	}
	if rehash, err := h.Verify(ctx, phc, "correct horse battery staple"); err != nil || rehash {
		t.Fatalf("verify: %v %v", rehash, err)
	}
	if _, err := h.Verify(ctx, phc, "wrong"); err != ErrPasswordMismatch {
		t.Fatalf("mismatch: %v", err)
	}
	stronger := NewPasswordHasher(Argon2Params{MemoryKiB: 16 * 1024, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}, 1)
	if rehash, err := stronger.Verify(ctx, phc, "correct horse battery staple"); err != nil || !rehash {
		t.Fatalf("expected rehash: %v %v", rehash, err)
	}
	if _, err := h.Verify(ctx, "$bcrypt$x", "x"); err == nil {
		t.Fatal("foreign hash must fail")
	}
}

func TestFieldCrypto(t *testing.T) {
	k1, k2 := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	kr, err := NewFieldKeyring("k1", map[string][]byte{"k1": k1})
	if err != nil {
		t.Fatal(err)
	}
	aad := []byte("admin_credentials.totp:u1")
	ct, kid, err := kr.Encrypt([]byte("JBSWY3DPEHPK3PXP"), aad)
	if err != nil || kid != "k1" {
		t.Fatal(err)
	}
	pt, err := kr.Decrypt(ct, kid, aad)
	if err != nil || string(pt) != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("decrypt: %v %q", err, pt)
	}
	if _, err := kr.Decrypt(ct, kid, []byte("other-row")); err == nil {
		t.Fatal("aad mismatch must fail")
	}
	ct[len(ct)-1] ^= 1
	if _, err := kr.Decrypt(ct, kid, aad); err == nil {
		t.Fatal("tamper must fail")
	}
	rotated, err := NewFieldKeyring("k2", map[string][]byte{"k1": k1, "k2": k2})
	if err != nil {
		t.Fatal(err)
	}
	ct1, _, _ := kr.Encrypt([]byte("x"), nil)
	if pt, err := rotated.Decrypt(ct1, "k1", nil); err != nil || string(pt) != "x" {
		t.Fatal("old key must still decrypt after rotation")
	}
	if _, err := rotated.Decrypt(ct1, "k9", nil); err != ErrUnknownKey {
		t.Fatal("unknown key id")
	}
	if _, err := NewFieldKeyring("k1", map[string][]byte{"k1": []byte("short")}); err == nil {
		t.Fatal("short key must fail")
	}
}

func TestTOTP(t *testing.T) {
	enr, err := NewTOTPSecret("rovo", "admin@rovo.local")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enr.URI, "otpauth://totp/") || len(enr.Secret) != 32 {
		t.Fatalf("enrolment %+v", enr)
	}
	now := time.Date(2026, 10, 4, 13, 0, 10, 0, time.UTC)
	code, _ := TOTPCode(enr.Secret, now)
	step, err := ValidateTOTP(enr.Secret, code, now, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTOTP(enr.Secret, code, now, step); err != ErrTOTPInvalid {
		t.Fatal("replay must be rejected")
	}
	prev, _ := TOTPCode(enr.Secret, now.Add(-30*time.Second))
	if _, err := ValidateTOTP(enr.Secret, prev, now, 0); err != nil {
		t.Fatal("previous step within window must pass")
	}
	old, _ := TOTPCode(enr.Secret, now.Add(-90*time.Second))
	if _, err := ValidateTOTP(enr.Secret, old, now, 0); err != ErrTOTPInvalid {
		t.Fatal("code outside window must fail")
	}
}

func TestCookies(t *testing.T) {
	now := time.Now()
	for _, secure := range []bool{true, false} {
		c := Cookies{Secure: secure}
		rec := httptest.NewRecorder()
		c.SetSession(rec, AudienceAdmin, "at", now.Add(5*time.Minute), "rt", 12*time.Hour, now)
		res := rec.Result()
		cookies := res.Cookies()
		if len(cookies) != 2 {
			t.Fatalf("cookies %d", len(cookies))
		}
		at, rt := cookies[0], cookies[1]
		if at.Name != c.AccessName() || at.Path != "/" || !at.HttpOnly || at.Domain != "" || at.SameSite != http.SameSiteStrictMode || at.Secure != secure {
			t.Fatalf("access cookie %+v", at)
		}
		if rt.Path != RefreshCookiePath || rt.SameSite != http.SameSiteStrictMode || rt.MaxAge != int((12*time.Hour)/time.Second) {
			t.Fatalf("refresh cookie %+v", rt)
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: c.AccessName(), Value: "tok"})
		if c.AccessToken(req) != "tok" {
			t.Fatal("read access cookie")
		}
	}
	if (Cookies{Secure: true}).AccessName() != "__Host-rovo_at" || (Cookies{Secure: true}).RefreshName() != "__Secure-rovo_rt" {
		t.Fatal("prefixed names")
	}
	rec := httptest.NewRecorder()
	Cookies{}.SetSession(rec, AudienceCustomer, "at", now.Add(time.Minute), "", 0, now)
	if ck := rec.Result().Cookies(); len(ck) != 1 || ck[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("customer access cookie must be Lax and refresh optional")
	}
}

func TestMAC(t *testing.T) {
	k := []byte("k")
	if bytes.Equal(MAC(k, []byte("ab"), []byte("c")), MAC(k, []byte("a"), []byte("bc"))) {
		t.Fatal("MAC must be unambiguous")
	}
	if !EqualMAC(MAC(k, []byte("x")), MAC(k, []byte("x"))) {
		t.Fatal("MAC must be deterministic")
	}
}
