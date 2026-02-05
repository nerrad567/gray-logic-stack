package location

import "time"

// ZoneDomain represents the infrastructure domain a zone belongs to.
type ZoneDomain string

const (
	ZoneDomainClimate  ZoneDomain = "climate"
	ZoneDomainAudio    ZoneDomain = "audio"
	ZoneDomainLighting ZoneDomain = "lighting"
	ZoneDomainPower    ZoneDomain = "power"
	ZoneDomainSecurity ZoneDomain = "security"
	ZoneDomainVideo    ZoneDomain = "video"
)

// AllZoneDomains returns all valid zone domain values.
func AllZoneDomains() []ZoneDomain {
	return []ZoneDomain{
		ZoneDomainClimate, ZoneDomainAudio, ZoneDomainLighting,
		ZoneDomainPower, ZoneDomainSecurity, ZoneDomainVideo,
	}
}

// validZoneDomains is a pre-computed set for O(1) domain validation.
var validZoneDomains = func() map[ZoneDomain]struct{} {
	m := make(map[ZoneDomain]struct{}, len(AllZoneDomains()))
	for _, d := range AllZoneDomains() {
		m[d] = struct{}{}
	}
	return m
}()

// ValidZoneDomain checks whether the given string is a valid zone domain.
func ValidZoneDomain(s string) bool {
	_, ok := validZoneDomains[ZoneDomain(s)]
	return ok
}

// InfrastructureZone represents a physical resource grouping that spans one or more rooms.
// All domains use this same table; domain-specific config lives in Settings.
//
// Examples:
//   - Climate zone: rooms sharing a thermostat. Settings: {"thermostat_id": "dev-123"}
//   - Audio zone: rooms sharing an amplifier channel. Settings: {"matrix_zone": 4}
//   - Lighting zone: rooms on a DALI bus. Settings: {"gateway_id": "dev-456", "dali_line": 1}
//   - Security zone: rooms in an alarm partition. Settings: {"panel_id": "dev-789", "partition": 1}
type InfrastructureZone struct {
	ID        string     `json:"id"`
	SiteID    string     `json:"site_id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Domain    ZoneDomain `json:"domain"`
	Settings  Settings   `json:"settings"`
	SortOrder int        `json:"sort_order"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
