//go:build !testhooks

package testhooks

import (
	"net/http"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
)

// Enabled reports whether the test-control surface is compiled in.
const Enabled = false

// Mount is a no-op in production builds: there is no /_test/* surface.
func Mount(_ *http.ServeMux, _ *Config, _ *app.Deps) {}

// Config holds the testhooks listener settings (empty in production).
type Config struct {
	Addr   string
	Secret string
}
