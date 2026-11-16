package testutil_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/migrate"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/ratelimit"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/testutil"
)

func TestMigrationsUpDown(t *testing.T) {
	d := testutil.NewEmptyDB(t)
	ctx := context.Background()
	r, err := migrate.New(ctx, d.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	var tables int
	if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables < 50 {
		t.Fatalf("expected the full schema, got %d tables", tables)
	}
	// Reference seeds present.
	var cuisines, cfg int
	_ = d.Pool.QueryRow(ctx, `SELECT count(*) FROM cuisines`).Scan(&cuisines)
	_ = d.Pool.QueryRow(ctx, `SELECT count(*) FROM app_config WHERE key='ordering.accept_window_s'`).Scan(&cfg)
	if cuisines == 0 || cfg != 1 {
		t.Fatalf("reference seeds missing: cuisines=%d cfg=%d", cuisines, cfg)
	}
	// Down should roll every migration back without error (dev path, doc 10 §16).
	if err := r.Reset(ctx); err != nil {
		t.Fatalf("down/reset: %v", err)
	}
	var remaining int
	_ = d.Pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='orders'`).Scan(&remaining)
	if remaining != 0 {
		t.Fatal("orders table should be gone after reset")
	}
}

func TestLedgerBalanceTrigger(t *testing.T) {
	d := testutil.NewMigratedDB(t)
	ctx := context.Background()
	cityID := testutil.SeedCity(t, d)
	acctA := seedAccount(t, d, cityID, "BANK", "ASSET", "D")
	acctB := seedAccount(t, d, cityID, "REVENUE_PLATFORM_FEE", "REVENUE", "C")

	post := func(a, b int64) error {
		return db.WithTx(ctx, d.Pool, func(tx pgx.Tx) error {
			jid := uuid.Must(uuid.NewV7())
			if _, err := tx.Exec(ctx, `INSERT INTO ledger_journals
				(id, city_id, entry_type, source_type, source_id, rule, idempotency_key, occurred_at, accounting_date)
				VALUES ($1,$2,'PAYMENT_CAPTURED','order',$3,'test.v1',$4,now(),current_date)`,
				jid, cityID, uuid.New(), jid.String()); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO ledger_postings (id, journal_id, account_id, amount_paise)
				VALUES ($1,$2,$3,$4)`, uuid.New(), jid, acctA, a); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `INSERT INTO ledger_postings (id, journal_id, account_id, amount_paise)
				VALUES ($1,$2,$3,$4)`, uuid.New(), jid, acctB, b)
			return err
		})
	}
	if err := post(10000, -10000); err != nil {
		t.Fatalf("balanced journal must commit: %v", err)
	}
	if err := post(10000, -9999); err == nil {
		t.Fatal("unbalanced journal must be rejected at commit")
	}
	// A single-posting journal must also fail (n < 2).
	err := db.WithTx(ctx, d.Pool, func(tx pgx.Tx) error {
		jid := uuid.Must(uuid.NewV7())
		if _, err := tx.Exec(ctx, `INSERT INTO ledger_journals
			(id, city_id, entry_type, source_type, source_id, rule, idempotency_key, occurred_at, accounting_date)
			VALUES ($1,$2,'ADJUSTMENT','order',$3,'t.v1',$4,now(),current_date)`,
			jid, cityID, uuid.New(), jid.String()); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO ledger_postings (id, journal_id, account_id, amount_paise) VALUES ($1,$2,$3,0)`,
			uuid.New(), jid, acctA)
		return err
	})
	if err == nil {
		t.Fatal("zero-amount / single posting must be rejected")
	}
}

func TestForbidMutationOnAppendOnly(t *testing.T) {
	d := testutil.NewMigratedDB(t)
	ctx := context.Background()
	cityID := testutil.SeedCity(t, d)
	id := uuid.Must(uuid.NewV7())
	if _, err := d.Pool.Exec(ctx, `INSERT INTO audit_logs (id, occurred_at, actor_type, action, resource_type, outcome)
		VALUES ($1, now(), 'SYSTEM', 'test.action', 'order', 'SUCCESS')`, id); err != nil {
		t.Fatalf("insert audit: %v", err)
	}
	if _, err := d.Pool.Exec(ctx, `UPDATE audit_logs SET action='changed' WHERE id=$1`, id); err == nil {
		t.Fatal("UPDATE on audit_logs must be rejected by forbid_mutation()")
	}
	if _, err := d.Pool.Exec(ctx, `DELETE FROM audit_logs WHERE id=$1`, id); err == nil {
		t.Fatal("DELETE on audit_logs must be rejected")
	}
	_ = cityID
}

func TestServiceabilityPostGIS(t *testing.T) {
	d := testutil.NewMigratedDB(t)
	ctx := context.Background()
	cityID := testutil.SeedCity(t, d)
	// Fixture octagon around Mahabubnagar (16 §10.1).
	zoneID := uuid.Must(uuid.NewV7())
	_, err := d.Pool.Exec(ctx, `INSERT INTO zones (id, city_id, code, name, boundary, status)
		VALUES ($1,$2,'MBNR-FIX','fixture', ST_Multi(ST_GeomFromText($3,4326)), 'ACTIVE')`,
		zoneID, cityID, "POLYGON((78.0505 16.7488,78.0367 16.7806,78.0035 16.7938,77.9703 16.7806,"+
			"77.9565 16.7488,77.9703 16.7170,78.0035 16.7038,78.0367 16.7170,78.0505 16.7488))")
	if err != nil {
		t.Fatalf("seed zone: %v", err)
	}
	covered := func(lng, lat float64) bool {
		var ok bool
		if err := d.Pool.QueryRow(ctx, `SELECT ST_Covers(boundary, ST_SetSRID(ST_MakePoint($1,$2),4326))
			FROM zones WHERE id=$3`, lng, lat, zoneID).Scan(&ok); err != nil {
			t.Fatal(err)
		}
		return ok
	}
	if !covered(78.0035, 16.7488) {
		t.Fatal("centre C1 must be inside")
	}
	if covered(78.0600, 16.7488) {
		t.Fatal("C3 must be outside")
	}
	if !covered(78.0505, 16.7488) {
		t.Fatal("vertex C5 must be inside with ST_Covers")
	}
	// Serviceability = inside zone AND within straight-line radius of the restaurant.
	var withinRadius bool
	if err := d.Pool.QueryRow(ctx, `SELECT ST_DWithin(
		ST_SetSRID(ST_MakePoint(78.0035,16.7488),4326)::geography,
		ST_SetSRID(ST_MakePoint(78.0200,16.7600),4326)::geography, 7000)`).Scan(&withinRadius); err != nil {
		t.Fatal(err)
	}
	if !withinRadius {
		t.Fatal("C1 should be within 7 km of the centre restaurant")
	}
}

func TestRateLimiter(t *testing.T) {
	d := testutil.NewMigratedDB(t)
	ctx := context.Background()
	clk := clock.NewFake(time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC))
	lim := ratelimit.New(d.Pool, clk)
	key := "otp:phone:test"
	for i := 1; i <= 3; i++ {
		dec, err := lim.Allow(ctx, key, 3, time.Minute)
		if err != nil || !dec.Allowed {
			t.Fatalf("hit %d should be allowed: %v %+v", i, err, dec)
		}
	}
	dec, err := lim.Allow(ctx, key, 3, time.Minute)
	if err != nil || dec.Allowed {
		t.Fatalf("4th hit must be blocked: %+v", dec)
	}
	if dec.RetryAfter <= 0 {
		t.Fatal("retry-after must be set")
	}
	// Next window resets.
	clk.Advance(time.Minute)
	if dec, _ := lim.Allow(ctx, key, 3, time.Minute); !dec.Allowed {
		t.Fatal("new window should reset the counter")
	}
}

func seedAccount(t *testing.T, d *testutil.DB, cityID uuid.UUID, code, typ, side string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	_, err := d.Pool.Exec(context.Background(), `INSERT INTO ledger_accounts
		(id, city_id, code, account_type, normal_side, owner_type) VALUES ($1,$2,$3,$4,$5,'PLATFORM')`,
		id, cityID, code, typ, side)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
