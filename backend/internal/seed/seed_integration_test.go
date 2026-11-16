package seed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/config"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/testutil"
	"github.com/storetmobiles2-code/rovo/backend/internal/seed"
)

func options(t *testing.T, env map[string]string) seed.Options {
	t.Helper()
	cfg, err := config.Load(func(k string) string {
		if v, ok := env[k]; ok {
			return v
		}
		switch k {
		case "ROVO_ENV":
			return "test"
		case "ROVO_DATABASE_URL":
			return "postgres://u:p@localhost:5432/x"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	fields, err := auth.NewFieldKeyring(cfg.FieldKeyID(), map[string][]byte{cfg.FieldKeyID(): cfg.FieldEncryptionKey})
	if err != nil {
		t.Fatal(err)
	}
	// Cheap argon2 parameters keep the test fast; the hash format is the same.
	hasher := auth.NewPasswordHasher(auth.Argon2Params{MemoryKiB: 1024, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}, 1)
	return seed.Options{Config: cfg, Passwords: hasher, Fields: fields, Now: time.Date(2026, 11, 16, 6, 0, 0, 0, time.UTC)}
}

func TestRunRefusedInProd(t *testing.T) {
	opts := options(t, nil)
	prod := *opts.Config
	prod.Env = config.EnvProd
	opts.Config = &prod
	// A nil pool proves the refusal happens before any database access.
	if _, err := seed.Run(context.Background(), nil, opts); !errors.Is(err, seed.ErrProd) {
		t.Fatalf("err = %v, want ErrProd", err)
	}
}

func TestRunSeedsAndIsIdempotent(t *testing.T) {
	d := testutil.NewDB(t)
	ctx := context.Background()
	opts := options(t, nil)

	res, err := seed.Run(ctx, d.Pool, opts)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	count := func(sql string, args ...any) int {
		t.Helper()
		var n int
		if err := d.Pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return n
	}
	for table, want := range map[string]int{
		"cities": 1, "zones": 1, "localities": 12, "restaurants": 4, "riders": 4, "coupons": 2,
		"fee_configs": 1, "commission_plans": 4, "ledger_accounts": 18, "rider_availability": 4,
	} {
		if got := count(`SELECT count(*) FROM ` + table); got != want {
			t.Errorf("%s = %d, want %d", table, got, want)
		}
	}

	// City and timezone.
	var tz, status string
	if err := d.Pool.QueryRow(ctx, `SELECT timezone, status FROM cities WHERE code = 'MBNR'`).Scan(&tz, &status); err != nil {
		t.Fatal(err)
	}
	if tz != "Asia/Kolkata" || status != "LIVE" {
		t.Errorf("city tz/status = %s/%s", tz, status)
	}

	// R59: no platform fee. R18: five slabs, last bound 10 km, ₹60.
	var platformFee int64
	var slabs, lastTo, lastFee int
	if err := d.Pool.QueryRow(ctx, `
		SELECT platform_fee_paise, jsonb_array_length(delivery_fee_slabs),
		       (delivery_fee_slabs->-1->>'toM')::int, (delivery_fee_slabs->-1->>'feePaise')::int
		  FROM fee_configs WHERE status = 'APPROVED'`).Scan(&platformFee, &slabs, &lastTo, &lastFee); err != nil {
		t.Fatal(err)
	}
	if platformFee != 0 || slabs != 5 || lastTo != 10000 || lastFee != 6000 {
		t.Errorf("fee config: platform=%d slabs=%d lastTo=%d lastFee=%d", platformFee, slabs, lastTo, lastFee)
	}

	// Every restaurant is ACTIVE inside the zone with a menu covering variants and add-ons.
	if n := count(`SELECT count(*) FROM restaurants r JOIN zones z ON z.id = r.zone_id
	                WHERE r.status = 'ACTIVE' AND r.accepting_orders AND ST_Covers(z.boundary, r.location::geometry)`); n != 4 {
		t.Errorf("active restaurants inside the zone = %d, want 4", n)
	}
	if n := count(`SELECT count(*) FROM restaurants r WHERE
	      EXISTS (SELECT 1 FROM menu_items i JOIN item_variants v ON v.menu_item_id = i.id WHERE i.restaurant_id = r.id)
	  AND EXISTS (SELECT 1 FROM menu_item_addon_groups g WHERE g.restaurant_id = r.id)
	  AND EXISTS (SELECT 1 FROM restaurant_operating_hours h WHERE h.restaurant_id = r.id)
	  AND EXISTS (SELECT 1 FROM restaurant_cuisines c WHERE c.restaurant_id = r.id AND c.is_primary)`); n != 4 {
		t.Errorf("restaurants with a complete menu = %d, want 4", n)
	}
	if n := count(`SELECT count(DISTINCT veg_type) FROM menu_items`); n != 3 {
		t.Errorf("distinct veg types = %d, want 3", n)
	}
	if n := count(`SELECT count(*) FROM menu_items WHERE NOT (name_i18n ? 'te')`); n != 0 {
		t.Errorf("%d menu items lack a Telugu name", n)
	}
	// has_variants agrees with the variant rows, and the base price equals the default variant.
	if n := count(`SELECT count(*) FROM menu_items i WHERE i.has_variants <>
	      EXISTS (SELECT 1 FROM item_variants v WHERE v.menu_item_id = i.id)`); n != 0 {
		t.Errorf("%d items have an inconsistent has_variants flag", n)
	}
	if n := count(`SELECT count(*) FROM menu_items i JOIN item_variants v ON v.menu_item_id = i.id AND v.is_default
	                WHERE v.price_paise <> i.price_paise`); n != 0 {
		t.Errorf("%d items differ from their default variant price", n)
	}

	// Admin: STAFF user with a global ADMIN_SUPER role; the printed password verifies and the
	// printed TOTP secret round-trips through field encryption.
	a := res.Admin
	if !a.Created || a.Password == "" || !a.TOTPEnrolledAtRun {
		t.Fatalf("admin result = %+v", a)
	}
	if n := count(`SELECT count(*) FROM user_roles WHERE user_id = $1 AND role = 'ADMIN_SUPER' AND city_id IS NULL AND revoked_at IS NULL`, a.UserID); n != 1 {
		t.Errorf("ADMIN_SUPER roles = %d, want 1", n)
	}
	var hash, keyID string
	var sealed []byte
	var mustChange bool
	if err := d.Pool.QueryRow(ctx, `SELECT password_hash, totp_secret_enc, pii_key_id, must_change_password
	                                  FROM admin_credentials WHERE user_id = $1`, a.UserID).Scan(&hash, &sealed, &keyID, &mustChange); err != nil {
		t.Fatal(err)
	}
	if _, err := opts.Passwords.Verify(ctx, hash, a.Password); err != nil {
		t.Errorf("generated password does not verify: %v", err)
	}
	if !mustChange {
		t.Error("a generated password must be flagged must_change_password")
	}
	plain, err := opts.Fields.Decrypt(sealed, keyID, seed.TOTPAAD(a.UserID))
	if err != nil || string(plain) != a.TOTPSecret {
		t.Errorf("TOTP secret does not round-trip: %v", err)
	}
	if _, err := auth.TOTPCode(a.TOTPSecret, opts.Now); err != nil {
		t.Errorf("TOTP secret is unusable: %v", err)
	}

	// Riders are ACTIVE and offline; coupons are live.
	if n := count(`SELECT count(*) FROM riders r JOIN rider_availability a ON a.rider_id = r.id
	                WHERE r.status = 'ACTIVE' AND a.state = 'OFFLINE'`); n != 4 {
		t.Errorf("active offline riders = %d, want 4", n)
	}
	if n := count(`SELECT count(*) FROM coupons WHERE status = 'ACTIVE' AND code IN ('WELCOME50','FLAT30')
	                AND $1 BETWEEN valid_from AND valid_until`, opts.Now); n != 2 {
		t.Errorf("live coupons = %d, want 2", n)
	}
	if n := count(`SELECT count(*) FROM app_config WHERE scope_type = 'CITY' AND scope_id = $1`, res.CityID); n < 6 {
		t.Errorf("city app_config rows = %d", n)
	}
	if n := count(`SELECT count(*) FROM reason_codes WHERE label_i18n ? 'te'`); n == 0 {
		t.Error("no reason code has a Telugu label")
	}

	// Second run: nothing new, credentials untouched.
	res2, err := seed.Run(ctx, d.Pool, opts)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	for k, n := range res2.Created {
		if n != 0 {
			t.Errorf("second run inserted %d %s rows", n, k)
		}
	}
	if res2.Admin.Created || res2.Admin.Password != "" || res2.Admin.TOTPEnrolledAtRun {
		t.Errorf("second run changed the admin: %+v", res2.Admin)
	}
	if res2.CityID != res.CityID || res2.ZoneID != res.ZoneID {
		t.Error("IDs changed between runs")
	}
	var hash2 string
	if err := d.Pool.QueryRow(ctx, `SELECT password_hash FROM admin_credentials WHERE user_id = $1`, a.UserID).Scan(&hash2); err != nil {
		t.Fatal(err)
	}
	if hash2 != hash {
		t.Error("second run rewrote the password hash")
	}

	// A password from the environment rotates the stored one and clears must-change.
	env := options(t, map[string]string{"ROVO_ADMIN_BOOTSTRAP_PASSWORD": "correct horse battery staple"})
	env.Fields = opts.Fields
	res3, err := seed.Run(ctx, d.Pool, env)
	if err != nil {
		t.Fatalf("third run: %v", err)
	}
	if res3.Admin.Password != "" || !res3.Admin.PasswordFromEnv {
		t.Errorf("env password run = %+v", res3.Admin)
	}
	if err := d.Pool.QueryRow(ctx, `SELECT password_hash, must_change_password FROM admin_credentials WHERE user_id = $1`, a.UserID).Scan(&hash, &mustChange); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Passwords.Verify(ctx, hash, "correct horse battery staple"); err != nil || mustChange {
		t.Errorf("env password not applied: verify=%v mustChange=%v", err, mustChange)
	}
}
