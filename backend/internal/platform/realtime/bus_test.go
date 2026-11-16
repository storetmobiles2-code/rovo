package realtime_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/realtime"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/testutil"
)

func TestBusTransactionalNotify(t *testing.T) {
	d := testutil.NewMigratedDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	bus := realtime.NewBus(d.Pool, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ready := make(chan bool, 4)
	bus.OnStateChange(func(up bool) {
		select {
		case ready <- up:
		default:
		}
	})
	go bus.Run(ctx)
	// Wait for LISTEN to connect.
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("bus did not connect")
	}
	sub := bus.Hub().Subscribe([]string{"order:42"})
	defer sub.Close()

	// A rolled-back transaction must NOT deliver (transactional NOTIFY).
	_ = db.WithTx(ctx, d.Pool, func(tx pgx.Tx) error {
		_ = bus.NotifyTx(ctx, tx, "order:42", map[string]any{"status": "PLACED"})
		return context.Canceled // force rollback
	})
	select {
	case f := <-sub.C:
		t.Fatalf("rolled-back notify must not deliver, got %s", f.Data)
	case <-time.After(500 * time.Millisecond):
	}

	// A committed transaction delivers.
	if err := db.WithTx(ctx, d.Pool, func(tx pgx.Tx) error {
		return bus.NotifyTx(ctx, tx, "order:42", map[string]any{"status": "ACCEPTED"})
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case f := <-sub.C:
		var m map[string]string
		_ = json.Unmarshal(f.Data, &m)
		if m["status"] != "ACCEPTED" {
			t.Fatalf("payload %s", f.Data)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("committed notify should deliver")
	}
}
