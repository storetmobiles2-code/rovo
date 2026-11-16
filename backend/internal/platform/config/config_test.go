package config

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDevDefaults(t *testing.T) {
	c, err := Load(env(map[string]string{"ROVO_DATABASE_URL": "postgres://u:secretpw@localhost:5432/rovo"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Env != EnvDev || c.HTTPAddr != ":8080" || c.InternalAddr != ":9090" || c.CookieSecure {
		t.Fatalf("defaults: %+v", c)
	}
	if len(c.Generated) != 3 {
		t.Fatalf("expected generated keys, got %v", c.Generated)
	}
	if c.AppHosts["localhost:5174"] != auth.AudienceRestaurant {
		t.Fatal("dev host map")
	}
	sum := c.Summary()
	if strings.Contains(sum["database_url"], "secretpw") {
		t.Fatal("password leaked in summary")
	}
	if len(c.DeriveKey("otp-pepper")) != 32 || string(c.DeriveKey("a")) == string(c.DeriveKey("b")) {
		t.Fatal("derived keys must be 32 bytes and label-specific")
	}
}

func TestProdRequiresSecrets(t *testing.T) {
	_, err := Load(env(map[string]string{"ROVO_ENV": "prod", "ROVO_DATABASE_URL": "postgres://x@db/rovo"}))
	if err == nil {
		t.Fatal("prod without secrets must fail")
	}
	for _, want := range []string{"ROVO_JWT_SIGNING_KEY", "ROVO_QUOTE_SIGNING_KEY", "ROVO_FIELD_ENCRYPTION_KEY",
		"ROVO_APP_HOSTS", "fake is not allowed", "ROVO_TRUSTED_ORIGIN_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}

func TestExplicitValues(t *testing.T) {
	k32 := base64.StdEncoding.EncodeToString(make([]byte, 32))
	c, err := Load(env(map[string]string{
		"ROVO_ENV": "staging", "ROVO_DATABASE_URL": "postgres://x@db/rovo",
		"ROVO_APP_HOSTS":       "app.example.in=customer, restaurant.example.in=restaurant,api.example.in=webhook",
		"ROVO_JWT_SIGNING_KEY": k32, "ROVO_QUOTE_SIGNING_KEY": k32, "ROVO_FIELD_ENCRYPTION_KEY": k32,
		"ROVO_FAKEPA_WEBHOOK_SECRET": "whsec", "ROVO_LOG_LEVEL": "debug",
		"ROVO_TESTHOOKS_ADDR": ":9999", "ROVO_TESTHOOKS_SECRET": "0123456789abcdef",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !c.CookieSecure || c.AppHosts["api.example.in"] != auth.AudienceWebhook || len(c.Generated) != 0 {
		t.Fatalf("parsed %+v", c)
	}
}

func TestValidationErrors(t *testing.T) {
	cases := map[string]map[string]string{
		"bad env":             {"ROVO_ENV": "qa", "ROVO_DATABASE_URL": "postgres://x@db/rovo"},
		"no db":               {},
		"bad host":            {"ROVO_DATABASE_URL": "postgres://x@db/rovo", "ROVO_APP_HOSTS": "app=superuser"},
		"short key":           {"ROVO_DATABASE_URL": "postgres://x@db/rovo", "ROVO_FIELD_ENCRYPTION_KEY": base64.StdEncoding.EncodeToString([]byte("short"))},
		"testhooks prod":      {"ROVO_ENV": "prod", "ROVO_DATABASE_URL": "postgres://x@db/rovo", "ROVO_TESTHOOKS_ADDR": ":1"},
		"testhooks no secret": {"ROVO_DATABASE_URL": "postgres://x@db/rovo", "ROVO_TESTHOOKS_ADDR": ":1"},
		"vapid half":          {"ROVO_DATABASE_URL": "postgres://x@db/rovo", "ROVO_PUSH_VAPID_PUBLIC_KEY": "x"},
	}
	for name, m := range cases {
		if _, err := Load(env(m)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
