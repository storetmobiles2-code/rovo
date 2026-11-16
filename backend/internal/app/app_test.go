package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	platform "github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/config"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load(func(k string) string {
		switch k {
		case "ROVO_ENV":
			return "test"
		case "ROVO_DATABASE_URL":
			if v := os.Getenv("ROVO_TEST_DATABASE_URL"); v != "" {
				return v
			}
			return "postgres://postgres:postgres@localhost:1/none?sslmode=disable"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// TestAllModulesWire asserts every service key has exactly one module and that route and job
// registration runs without panicking (duplicate ServeMux patterns panic).
func TestAllModulesWire(t *testing.T) {
	cfg := testConfig(t)
	// pgxpool connects lazily, so wiring needs no reachable database.
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, worker := range []bool{false, true} {
		a, err := New(cfg, logger, pool, Options{Worker: worker})
		if err != nil {
			t.Fatalf("New(worker=%v): %v", worker, err)
		}
		want := []string{
			platform.ServiceIdentity, platform.ServiceGeo, platform.ServiceUsers, platform.ServiceCatalog,
			platform.ServicePricing, platform.ServicePromotions, platform.ServiceOrdering, platform.ServicePayments,
			platform.ServiceDispatch, platform.ServiceLedger, platform.ServiceRatings, platform.ServiceNotifications,
			platform.ServiceSupport, platform.ServiceAdminAudit, platform.ServiceReporting,
		}
		if len(a.Modules) != len(want) {
			t.Fatalf("modules = %d, want %d", len(a.Modules), len(want))
		}
		for _, k := range want {
			if a.Deps.Service(k) == nil {
				t.Errorf("service %q not registered", k)
			}
		}
		mux := httpx.NewMux()
		if got := a.RegisterRoutes(mux); len(got) != len(want) {
			t.Errorf("RegisterRoutes registered %d modules, want %d", len(got), len(want))
		}
		// An unknown path must still 404 rather than panic.
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/__nope", nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("unknown path = %d, want 404", rec.Code)
		}
		if a.Deps.Queue == nil || a.Deps.Events == nil {
			t.Error("queue/events not wired")
		}
	}
}
