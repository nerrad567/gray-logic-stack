package device

import "time"

// GroupType defines how a device group resolves its members.
type GroupType string

const (
	// GroupTypeStatic resolves from explicit member list only.
	GroupTypeStatic GroupType = "static"
	// GroupTypeDynamic resolves from filter rules only (location plus device filters).
	GroupTypeDynamic GroupType = "dynamic"
	// GroupTypeHybrid resolves from filter rules unioned with explicit members.
	GroupTypeHybrid GroupType = "hybrid"
)

// FilterRules defines dynamic resolution criteria for a device group.
// All non-empty fields are ANDed together. Empty fields are ignored.
type FilterRules struct {
	// Location scope
	ScopeType string `json:"scope_type,omitempty"` // "site", "area", or "room"
	ScopeID   string `json:"scope_id,omitempty"`   // area or room ID (empty for site scope)

	// Device filters (all ANDed)
	Domains      []string `json:"domains,omitempty"`      // e.g., ["lighting", "audio"]
	Capabilities []string `json:"capabilities,omitempty"` // e.g., ["dim", "on_off"]
	Tags         []string `json:"tags,omitempty"`         // include devices with ANY of these tags
	ExcludeTags  []string `json:"exclude_tags,omitempty"` // exclude devices with ANY of these tags
	DeviceTypes  []string `json:"device_types,omitempty"` // e.g., ["light_dimmer", "light_rgb"]
}

// DeviceGroup is a named, resolvable collection of devices.
type DeviceGroup struct { //nolint:revive // device.DeviceGroup is clearer than device.Group in calling code
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Description *string      `json:"description,omitempty"`
	Type        GroupType    `json:"type"`
	FilterRules *FilterRules `json:"filter_rules,omitempty"` // nil for static groups
	Icon        *string      `json:"icon,omitempty"`
	Colour      *string      `json:"colour,omitempty"`
	SortOrder   int          `json:"sort_order"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// GroupMember represents an explicit device in a static or hybrid group.
type GroupMember struct {
	GroupID   string    `json:"group_id"`
	DeviceID  string    `json:"device_id"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}
