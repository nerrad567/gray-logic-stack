package location

import "time"

// Area represents a logical grouping within a site (floor, building, wing).
type Area struct {
	ID        string    `json:"id"`
	SiteID    string    `json:"site_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Type      string    `json:"type"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Room represents a physical space within an area.
type Room struct {
	ID            string    `json:"id"`
	AreaID        string    `json:"area_id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Type          string    `json:"type"`
	SortOrder     int       `json:"sort_order"`
	ClimateZoneID *string   `json:"climate_zone_id,omitempty"`
	AudioZoneID   *string   `json:"audio_zone_id,omitempty"`
	Settings      Settings  `json:"settings"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Settings holds room-specific configuration as a JSON map.
type Settings map[string]any
