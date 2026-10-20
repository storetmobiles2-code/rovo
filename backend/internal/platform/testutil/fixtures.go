package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// NewMigratedDB returns a database cloned from the migrated template (alias of NewDB for tests
// that read intent from the name).
func NewMigratedDB(t testing.TB) *DB { return NewDB(t) }

// SeedCity inserts Mahabubnagar and returns its id. Reference seeds (cuisines, app_config, tax
// rules, reason codes) are already present from the template migrations.
func SeedCity(t testing.TB, d *DB) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	_, err := d.Pool.Exec(context.Background(), `INSERT INTO cities
		(id, code, slug, name, name_i18n, state_name, gst_state_code, timezone, centroid, status)
		VALUES ($1,'MBNR','mahabubnagar','Mahabubnagar','{"te":"మహబూబ్‌నగర్"}','Telangana','36','Asia/Kolkata',
		        ST_SetSRID(ST_MakePoint(78.0035,16.7488),4326)::geography, 'LIVE')`, id)
	if err != nil {
		t.Fatalf("seed city: %v", err)
	}
	return id
}

// SeedUser inserts a MEMBER user with a phone and returns its id.
func SeedUser(t testing.TB, d *DB, phone string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	_, err := d.Pool.Exec(context.Background(), `INSERT INTO users (id, kind, phone_e164, status)
		VALUES ($1,'MEMBER',$2,'ACTIVE')`, id, phone)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}
