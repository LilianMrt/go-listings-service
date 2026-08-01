// Package migrations embeds the SQL migration files so they travel with the
// binary and can be applied identically from the app and from tests.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
