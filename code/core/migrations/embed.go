// Package migrations embeds SQL migration files into the binary.
//
// This allows Gray Logic to run migrations without needing the SQL files
// present on the filesystem - they're compiled into the executable.
package migrations

import (
	"embed"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/database"
)

//go:embed *.sql
var migrationsFS embed.FS

func init() {
	// Register embedded migrations with the database package.
	// The embed directive above captures all .sql files in this directory.
	database.MigrationsFS = migrationsFS
	database.MigrationsDir = "." // Files are at root of embedded FS
}
