package device

import (
	"context"
	"time"
)

// State history source values.
const (
	StateHistorySourceMQTT    = "mqtt"
	StateHistorySourceCommand = "command"
	StateHistorySourceScene   = "scene"
)

// StateHistoryEntry represents a single device state change record.
//
// Each entry stores a full snapshot of the device state at the time the
// change was observed. This provides a local audit trail even when the
// time-series database is unavailable.
type StateHistoryEntry struct {
	// ID is the auto-incremented primary key for the history row.
	ID int64 `json:"id"`

	// DeviceID is the unique identifier of the device.
	DeviceID string `json:"device_id"`

	// State is the JSON snapshot of the device state.
	State State `json:"state"`

	// Source identifies how the state change was recorded (mqtt, command, scene).
	Source string `json:"source"`

	// CreatedAt is the timestamp of the state change (UTC).
	CreatedAt time.Time `json:"created_at"`
}

// StateHistoryRepository stores and retrieves device state change history.
//
// Implementations must be thread-safe and use UTC timestamps.
type StateHistoryRepository interface {
	// RecordStateChange records a device state change.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - deviceID: Unique device identifier
	//   - state: State snapshot to persist
	//   - source: Origin of the change (mqtt, command, scene)
	//
	// Returns:
	//   - error: nil on success, otherwise the underlying persistence error
	RecordStateChange(ctx context.Context, deviceID string, state State, source string) error

	// GetHistory returns recent state change history for the device.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - deviceID: Unique device identifier
	//   - limit: Maximum entries to return (implementation may clamp bounds)
	//
	// Returns:
	//   - []StateHistoryEntry: Ordered newest-first history entries (may be empty)
	//   - error: nil on success, otherwise the underlying query error
	GetHistory(ctx context.Context, deviceID string, limit int) ([]StateHistoryEntry, error)
}
