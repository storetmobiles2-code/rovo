// Package migrations embeds the goose SQL migrations so the rovo binary can apply them
// without access to the source tree.
package migrations

import "embed"

// FS holds every *.sql migration in version order (goose sorts by the numeric prefix).
//
//go:embed *.sql
var FS embed.FS
