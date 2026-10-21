// Package config parses and validates the ROVO_* environment (12-factor). The variable list is
// fixed and shared with DevOps' .env.example; adding a variable means updating both.
package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/crypto/hkdf"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
)

// Env is the deployment environment.
type Env string

// Environments.
const (
	EnvDev     Env = "dev"
	EnvTest    Env = "test"
	EnvStaging Env = "staging"
	EnvProd    Env = "prod"
)

// S3 holds S3-compatible object storage settings (MinIO locally).
type S3 struct {
	Endpoint           string
	Region             string
	AccessKey          string
	SecretKey          string
	BucketMedia        string
	BucketKYC          string
	PublicMediaBaseURL string
}

// Config is the validated process configuration.
type Config struct {
	Env                    Env
	HTTPAddr               string
	InternalAddr           string
	DatabaseURL            string
	AppHosts               map[string]auth.Audience
	CustomerBaseURL        string
	JWTSigningKey          ed25519.PrivateKey
	QuoteSigningKey        []byte
	FieldEncryptionKey     []byte
	CookieSecure           bool
	TrustedOriginSecret    string
	PaymentProvider        string
	FakePAWebhookSecret    string
	SMSProvider            string
	PushVAPIDPublicKey     string
	PushVAPIDPrivateKey    string
	S3                     S3
	SMTPAddr               string
	LogLevel               slog.Level
	OTLPEndpoint           string
	TesthooksAddr          string
	TesthooksSecret        string
	AdminBootstrapPassword string

	// Generated lists keys that were generated at start-up because they were unset (dev/test
	// only). Tokens signed with generated keys do not survive a restart.
	Generated []string
}

// Load reads configuration through getenv (os.Getenv in production, a map in tests).
func Load(getenv func(string) string) (*Config, error) {
	var errs []error
	get := func(k, def string) string {
		if v := strings.TrimSpace(getenv(k)); v != "" {
			return v
		}
		return def
	}
	c := &Config{
		Env:                 Env(get("ROVO_ENV", "dev")),
		HTTPAddr:            get("ROVO_HTTP_ADDR", ":8080"),
		InternalAddr:        get("ROVO_INTERNAL_ADDR", ":9090"),
		DatabaseURL:         get("ROVO_DATABASE_URL", ""),
		CustomerBaseURL:     get("ROVO_CUSTOMER_BASE_URL", ""),
		TrustedOriginSecret: get("ROVO_TRUSTED_ORIGIN_SECRET", ""),
		PaymentProvider:     strings.ToLower(get("ROVO_PAYMENT_PROVIDER", "fake")),
		FakePAWebhookSecret: get("ROVO_FAKEPA_WEBHOOK_SECRET", ""),
		SMSProvider:         strings.ToLower(get("ROVO_SMS_PROVIDER", "fake")),
		PushVAPIDPublicKey:  get("ROVO_PUSH_VAPID_PUBLIC_KEY", ""),
		PushVAPIDPrivateKey: get("ROVO_PUSH_VAPID_PRIVATE_KEY", ""),
		S3: S3{
			Endpoint:           get("ROVO_S3_ENDPOINT", ""),
			Region:             get("ROVO_S3_REGION", "ap-south-1"),
			AccessKey:          get("ROVO_S3_ACCESS_KEY", ""),
			SecretKey:          get("ROVO_S3_SECRET_KEY", ""),
			BucketMedia:        get("ROVO_S3_BUCKET_MEDIA", ""),
			BucketKYC:          get("ROVO_S3_BUCKET_KYC", ""),
			PublicMediaBaseURL: get("ROVO_S3_PUBLIC_MEDIA_BASE_URL", ""),
		},
		SMTPAddr:               get("ROVO_SMTP_ADDR", ""),
		OTLPEndpoint:           get("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		TesthooksAddr:          get("ROVO_TESTHOOKS_ADDR", ""),
		TesthooksSecret:        get("ROVO_TESTHOOKS_SECRET", ""),
		AdminBootstrapPassword: get("ROVO_ADMIN_BOOTSTRAP_PASSWORD", ""),
	}
	switch c.Env {
	case EnvDev, EnvTest, EnvStaging, EnvProd:
	default:
		errs = append(errs, fmt.Errorf("ROVO_ENV must be one of dev|test|staging|prod, got %q", c.Env))
	}
	devLike := c.Env == EnvDev || c.Env == EnvTest

	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("ROVO_DATABASE_URL is required"))
	} else if u, err := url.Parse(c.DatabaseURL); err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		errs = append(errs, errors.New("ROVO_DATABASE_URL must be a postgres:// URL"))
	}

	var err error
	if c.CookieSecure, err = parseBool(get("ROVO_COOKIE_SECURE", strconv.FormatBool(!devLike))); err != nil {
		errs = append(errs, fmt.Errorf("ROVO_COOKIE_SECURE: %w", err))
	}
	if c.LogLevel, err = parseLevel(get("ROVO_LOG_LEVEL", "info")); err != nil {
		errs = append(errs, err)
	}

	hosts := get("ROVO_APP_HOSTS", "")
	if hosts == "" && devLike {
		hosts = DefaultDevHosts
	}
	if c.AppHosts, err = ParseAppHosts(hosts); err != nil {
		errs = append(errs, fmt.Errorf("ROVO_APP_HOSTS: %w", err))
	}

	c.JWTSigningKey, err = c.loadSeedKey(getenv("ROVO_JWT_SIGNING_KEY"), devLike)
	if err != nil {
		errs = append(errs, fmt.Errorf("ROVO_JWT_SIGNING_KEY: %w", err))
	}
	c.QuoteSigningKey, err = c.loadSecret("ROVO_QUOTE_SIGNING_KEY", getenv("ROVO_QUOTE_SIGNING_KEY"), 32, devLike)
	if err != nil {
		errs = append(errs, err)
	}
	c.FieldEncryptionKey, err = c.loadSecret("ROVO_FIELD_ENCRYPTION_KEY", getenv("ROVO_FIELD_ENCRYPTION_KEY"), 32, devLike)
	if err != nil {
		errs = append(errs, err)
	} else if len(c.FieldEncryptionKey) != 32 {
		errs = append(errs, errors.New("ROVO_FIELD_ENCRYPTION_KEY must decode to exactly 32 bytes"))
	}

	switch c.PaymentProvider {
	case "fake":
		if c.Env == EnvProd {
			errs = append(errs, errors.New("ROVO_PAYMENT_PROVIDER=fake is not allowed in prod"))
		}
		if c.FakePAWebhookSecret == "" {
			if devLike {
				c.FakePAWebhookSecret = "dev-fakepa-webhook-secret"
			} else {
				errs = append(errs, errors.New("ROVO_FAKEPA_WEBHOOK_SECRET is required with the fake provider"))
			}
		}
	case "razorpay", "cashfree":
		errs = append(errs, fmt.Errorf("ROVO_PAYMENT_PROVIDER=%s: adapter not available in this build", c.PaymentProvider))
	default:
		errs = append(errs, fmt.Errorf("ROVO_PAYMENT_PROVIDER: unknown provider %q", c.PaymentProvider))
	}
	if c.SMSProvider != "fake" {
		errs = append(errs, fmt.Errorf("ROVO_SMS_PROVIDER: only \"fake\" is available in this build, got %q", c.SMSProvider))
	} else if c.Env == EnvProd {
		errs = append(errs, errors.New("ROVO_SMS_PROVIDER=fake is not allowed in prod"))
	}

	if c.TesthooksAddr != "" {
		if c.Env == EnvProd {
			errs = append(errs, errors.New("ROVO_TESTHOOKS_ADDR must not be set in prod"))
		}
		if len(c.TesthooksSecret) < 16 {
			errs = append(errs, errors.New("ROVO_TESTHOOKS_SECRET (>= 16 chars) is required when ROVO_TESTHOOKS_ADDR is set"))
		}
	}
	if c.Env == EnvProd {
		if !c.CookieSecure {
			errs = append(errs, errors.New("ROVO_COOKIE_SECURE must be true in prod"))
		}
		if c.TrustedOriginSecret == "" {
			errs = append(errs, errors.New("ROVO_TRUSTED_ORIGIN_SECRET is required in prod"))
		}
		if c.CustomerBaseURL == "" {
			errs = append(errs, errors.New("ROVO_CUSTOMER_BASE_URL is required in prod"))
		}
	}
	if c.CustomerBaseURL != "" {
		if u, err := url.Parse(c.CustomerBaseURL); err != nil || u.Scheme == "" || u.Host == "" {
			errs = append(errs, errors.New("ROVO_CUSTOMER_BASE_URL must be an absolute URL"))
		}
	}
	if (c.PushVAPIDPublicKey == "") != (c.PushVAPIDPrivateKey == "") {
		errs = append(errs, errors.New("ROVO_PUSH_VAPID_PUBLIC_KEY and ROVO_PUSH_VAPID_PRIVATE_KEY must be set together"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return c, nil
}

// DefaultDevHosts maps the local Caddy ports (one per app) and the API port to audiences.
const DefaultDevHosts = "localhost:5173=customer,localhost:5174=restaurant,localhost:5175=rider," +
	"localhost:5176=admin,localhost:8080=customer,127.0.0.1:8080=customer,api.localhost:8080=webhook"

// ParseAppHosts parses "host[:port]=audience,..." into a lower-cased lookup table.
func ParseAppHosts(s string) (map[string]auth.Audience, error) {
	out := map[string]auth.Audience{}
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		host, aud, ok := strings.Cut(item, "=")
		host = strings.ToLower(strings.TrimSpace(host))
		a := auth.Audience(strings.TrimSpace(aud))
		if !ok || host == "" || !a.Valid() {
			return nil, fmt.Errorf("invalid entry %q (want host[:port]=customer|restaurant|rider|admin|webhook)", item)
		}
		if _, dup := out[host]; dup {
			return nil, fmt.Errorf("duplicate host %q", host)
		}
		out[host] = a
	}
	if len(out) == 0 {
		return nil, errors.New("at least one host mapping is required")
	}
	return out, nil
}

func (c *Config) loadSeedKey(v string, devLike bool) (ed25519.PrivateKey, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		if !devLike {
			return nil, errors.New("required outside dev/test")
		}
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		c.Generated = append(c.Generated, "ROVO_JWT_SIGNING_KEY")
		return priv, err
	}
	b, err := decodeB64(v)
	if err != nil {
		return nil, err
	}
	switch len(b) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(b), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(b), nil
	}
	return nil, fmt.Errorf("must be a base64 32-byte Ed25519 seed, got %d bytes", len(b))
}

func (c *Config) loadSecret(name, v string, n int, devLike bool) ([]byte, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		if !devLike {
			return nil, fmt.Errorf("%s is required outside dev/test", name)
		}
		b := make([]byte, n)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		c.Generated = append(c.Generated, name)
		return b, nil
	}
	b, err := decodeB64(v)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if len(b) < n {
		return nil, fmt.Errorf("%s must decode to at least %d bytes", name, n)
	}
	return b, nil
}

func decodeB64(v string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(v); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("must be base64")
}

func parseBool(v string) (bool, error) { return strconv.ParseBool(strings.ToLower(v)) }

func parseLevel(v string) (slog.Level, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(v)); err != nil {
		return 0, fmt.Errorf("ROVO_LOG_LEVEL: %w", err)
	}
	return l, nil
}

// IsProd reports whether this is production.
func (c *Config) IsProd() bool { return c.Env == EnvProd }

// DevLike reports whether dev conveniences (generated keys, printed secrets) are allowed.
func (c *Config) DevLike() bool { return c.Env == EnvDev || c.Env == EnvTest }

// Issuer is the JWT iss claim for this deployment.
func (c *Config) Issuer() string {
	if c.CustomerBaseURL != "" {
		return strings.TrimRight(c.CustomerBaseURL, "/")
	}
	return "rovo-" + string(c.Env)
}

// DeriveKey derives a 32-byte purpose-specific key from ROVO_FIELD_ENCRYPTION_KEY with HKDF
// (labels such as "otp-pepper", "phone-hmac", "cursor"). Rotating the root key rotates them all.
func (c *Config) DeriveKey(label string) []byte {
	r := hkdf.New(sha256.New, c.FieldEncryptionKey, []byte("rovo/v1"), []byte(label))
	out := make([]byte, 32)
	if _, err := io.ReadFull(r, out); err != nil {
		panic("config: hkdf: " + err.Error())
	}
	return out
}

// FieldKeyID is the pii_key_id stored with values encrypted under the current root key: the
// first 8 hex chars of SHA-256(key), so a rotated key yields a new id.
func (c *Config) FieldKeyID() string {
	s := sha256.Sum256(c.FieldEncryptionKey)
	return fmt.Sprintf("fk-%x", s[:4])
}

// LogValue renders a redacted summary for the start-up log line.
func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(summaryAttrs(c)...)
}

// Summary returns the redacted configuration as a sorted map (rovo version / start-up log).
func (c *Config) Summary() map[string]string {
	out := map[string]string{}
	for _, a := range summaryAttrs(c) {
		out[a.Key] = a.Value.String()
	}
	return out
}

func summaryAttrs(c *Config) []slog.Attr {
	hostList := make([]string, 0, len(c.AppHosts))
	for h, a := range c.AppHosts {
		hostList = append(hostList, h+"="+string(a))
	}
	sort.Strings(hostList)
	set := func(v string) string {
		if v == "" {
			return "unset"
		}
		return "set"
	}
	return []slog.Attr{
		slog.String("env", string(c.Env)),
		slog.String("http_addr", c.HTTPAddr),
		slog.String("internal_addr", c.InternalAddr),
		slog.String("database_url", redactURL(c.DatabaseURL)),
		slog.String("app_hosts", strings.Join(hostList, ",")),
		slog.String("customer_base_url", c.CustomerBaseURL),
		slog.Bool("cookie_secure", c.CookieSecure),
		slog.String("trusted_origin_secret", set(c.TrustedOriginSecret)),
		slog.String("payment_provider", c.PaymentProvider),
		slog.String("sms_provider", c.SMSProvider),
		slog.String("push_vapid", set(c.PushVAPIDPublicKey)),
		slog.String("s3_endpoint", c.S3.Endpoint),
		slog.String("s3_bucket_media", c.S3.BucketMedia),
		slog.String("s3_bucket_kyc", c.S3.BucketKYC),
		slog.String("smtp_addr", c.SMTPAddr),
		slog.String("log_level", c.LogLevel.String()),
		slog.String("otlp_endpoint", c.OTLPEndpoint),
		slog.String("testhooks_addr", c.TesthooksAddr),
		slog.String("generated_keys", strings.Join(c.Generated, ",")),
	}
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || raw == "" {
		return ""
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "xxxxx")
	}
	return u.String()
}
