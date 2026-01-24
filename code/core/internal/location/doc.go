// Package location provides the area and room hierarchy for sites.
//
// It defines the spatial model used by Gray Logic: Sites contain Areas
// (floors, wings, buildings), which contain Rooms (physical spaces).
// Each room can reference climate zones, audio zones, and per-room settings.
//
// The package provides a Repository interface with a SQLite implementation
// for querying areas and rooms by site or area membership.
//
// # Thread Safety
//
// SQLiteRepository is safe for concurrent use from multiple goroutines
// (SQLite WAL mode + connection pooling).
package location
