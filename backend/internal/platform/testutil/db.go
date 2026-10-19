// Package testutil provides integration-test helpers: a fresh, fully migrated PostGIS database
// per test (cloned from a template), fixtures, HTTP helpers and the OpenAPI contract validator.
//
// Integration tests need ROVO_TEST_DATABASE_URL pointing at a PostGIS server where the user may
// CREATE DATABASE; without it they are skipped (CI always sets it).
package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/idgen"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/migrate"
	"github.com/storetmobiles2-code/rovo/backend/migrations"
)

// EnvDatabaseURL names the admin connection used to create test databases.
const EnvDatabaseURL = "ROVO_TEST_DATABASE_URL"

var (
	templateOnce sync.Once
	templateName string
	templateErr  error
)

// DB is a throwaway database for one test.
type DB struct {
	Pool *pgxpool.Pool
	URL  string
	Name string
}

// NewDB returns a pool on a new database cloned from the migrated template. The database is
// dropped when the test ends.
func NewDB(t testing.TB) *DB {
	t.Helper()
	admin := os.Getenv(EnvDatabaseURL)
	if admin == "" {
		t.Skipf("%s not set; skipping integration test", EnvDatabaseURL)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	templateOnce.Do(func() { templateName, templateErr = ensureTemplate(ctx, admin) })
	if templateErr != nil {
		t.Fatalf("testutil: template database: %v", templateErr)
	}
	name := "rovo_t_" + strings.ToLower(idgen.RandomCrockford(10))
	if err := adminExec(ctx, admin, fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s`, pgx.Identifier{name}.Sanitize(),
		pgx.Identifier{templateName}.Sanitize())); err != nil {
		t.Fatalf("testutil: create database: %v", err)
	}
	dbURL := withDatabase(admin, name)
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("testutil: connect: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		dctx, dcancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dcancel()
		_ = adminExec(dctx, admin, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, pgx.Identifier{name}.Sanitize()))
	})
	return &DB{Pool: pool, URL: dbURL, Name: name}
}

// NewEmptyDB returns an unmigrated database (for migration up/down tests).
func NewEmptyDB(t testing.TB) *DB {
	t.Helper()
	admin := os.Getenv(EnvDatabaseURL)
	if admin == "" {
		t.Skipf("%s not set; skipping integration test", EnvDatabaseURL)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	name := "rovo_e_" + strings.ToLower(idgen.RandomCrockford(10))
	if err := adminExec(ctx, admin, fmt.Sprintf(`CREATE DATABASE %s`, pgx.Identifier{name}.Sanitize())); err != nil {
		t.Fatalf("testutil: create database: %v", err)
	}
	dbURL := withDatabase(admin, name)
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("testutil: connect: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		dctx, dcancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dcancel()
		_ = adminExec(dctx, admin, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, pgx.Identifier{name}.Sanitize()))
	})
	return &DB{Pool: pool, URL: dbURL, Name: name}
}

// ensureTemplate creates (once per migration set) a migrated template database. The name is a
// hash of the embedded migrations, so changing a migration produces a new template. A
// cluster-wide advisory lock serialises creation across parallel test packages.
func ensureTemplate(ctx context.Context, admin string) (string, error) {
	sum, err := migrationsHash()
	if err != nil {
		return "", err
	}
	name := "rovo_tpl_" + sum[:12]
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(74_726_576)`); err != nil {
		return "", err
	}
	defer func() { _, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock(74_726_576)`) }()
	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1 AND datistemplate)`, name).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return name, nil
	}
	_, _ = conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, pgx.Identifier{name}.Sanitize()))
	if _, err := conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, pgx.Identifier{name}.Sanitize())); err != nil {
		return "", err
	}
	if err := migrate.Up(ctx, withDatabase(admin, name), nil); err != nil {
		return "", fmt.Errorf("migrate template: %w", err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf(`ALTER DATABASE %s WITH IS_TEMPLATE true ALLOW_CONNECTIONS false`,
		pgx.Identifier{name}.Sanitize())); err != nil {
		return "", err
	}
	return name, nil
}

func migrationsHash() (string, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		b, err := fs.ReadFile(migrations.FS, n)
		if err != nil {
			return "", err
		}
		h.Write([]byte(n))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func adminExec(ctx context.Context, admin, sql string) error {
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, sql)
	return err
}

func withDatabase(raw, name string) string {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	u.Path = "/" + name
	return u.String()
}
