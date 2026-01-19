// Package database provides SQLite database connectivity for Gray Logic Core.
//
// This package manages:
//   - Database connection with WAL mode for concurrent access
//   - Schema migrations (additive-only per database-schema.md)
//   - Connection pooling and lifecycle management
//   - STRICT mode enforcement for type safety
//
// Security Considerations:
//   - All queries use parameterised statements (no SQL injection)
//   - Database file permissions are set to 0600 (owner read/write only)
//   - Sensitive data (passwords, tokens) should be encrypted at rest
//
// Performance Characteristics:
//   - WAL mode allows concurrent reads during writes
//   - Busy timeout prevents lock contention errors
//   - Connection pooling reduces overhead
//
// Usage:
//
//	db, err := database.Open(cfg.Database)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Run migrations
//	if err := db.Migrate(); err != nil {
//	    log.Fatal(err)
//	}
//
// Migration Strategy:
//
// Migrations are additive-only to support safe rollbacks:
//   - New columns must be NULLABLE or have DEFAULT values
//   - Never DROP or RENAME columns (until v2.0 major release)
//   - Each migration file has both .up.sql and .down.sql
//
// Related Documents:
//   - docs/development/database-schema.md — Schema strategy
//   - docs/data-model/entities.md — Entity definitions
//   - docs/development/CODING-STANDARDS.md — Migration file format
package database
