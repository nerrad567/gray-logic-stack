package api

import (
	"database/sql"
	"net/http"
	"time"
)

// DiscoveryData represents passively discovered KNX bus data.
type DiscoveryData struct {
	GroupAddresses []DiscoveredGA     `json:"group_addresses"`
	Devices        []DiscoveredDevice `json:"devices"`
	Summary        DiscoverySummary   `json:"summary"`
}

// DiscoveredGA represents a group address seen on the KNX bus.
type DiscoveredGA struct {
	GroupAddress    string `json:"group_address"`
	LastSeen        string `json:"last_seen"`
	LastSeenAgo     string `json:"last_seen_ago"`
	MessageCount    int64  `json:"message_count"`
	HasReadResponse bool   `json:"has_read_response"`
}

// DiscoveredDevice represents a KNX device (individual address) seen on the bus.
type DiscoveredDevice struct {
	IndividualAddress string `json:"individual_address"`
	LastSeen          string `json:"last_seen"`
	LastSeenAgo       string `json:"last_seen_ago"`
	MessageCount      int64  `json:"message_count"`
}

// DiscoverySummary provides aggregate statistics about discovered data.
type DiscoverySummary struct {
	TotalGroupAddresses int `json:"total_group_addresses"`
	TotalDevices        int `json:"total_devices"`
	RespondingAddresses int `json:"responding_addresses"`
	ActiveLast5Min      int `json:"active_last_5min"`
	ActiveLast1Hour     int `json:"active_last_1hour"`
}

// handleListDiscovery returns passively discovered KNX bus data.
func (s *Server) handleListDiscovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get DB handle
	db := s.getDB()
	if db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "database not available",
		})
		return
	}

	now := time.Now()
	fiveMinAgo := now.Add(-5 * time.Minute).Unix()
	oneHourAgo := now.Add(-1 * time.Hour).Unix()

	// Query group addresses
	gaRows, err := db.QueryContext(ctx, `
		SELECT group_address, last_seen, message_count, has_read_response
		FROM knx_group_addresses
		ORDER BY last_seen DESC
		LIMIT 500
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to query group addresses",
		})
		return
	}
	defer gaRows.Close()

	var groupAddresses []DiscoveredGA
	var respondingCount int
	var activeLast5MinGA int
	var activeLast1HourGA int

	for gaRows.Next() {
		var ga DiscoveredGA
		var lastSeenUnix int64
		var hasReadResponse int

		if err := gaRows.Scan(&ga.GroupAddress, &lastSeenUnix, &ga.MessageCount, &hasReadResponse); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
			continue
		}

		lastSeenTime := time.Unix(lastSeenUnix, 0)
		ga.LastSeen = lastSeenTime.Format(time.RFC3339)
		ga.LastSeenAgo = formatDuration(now.Sub(lastSeenTime))
		ga.HasReadResponse = hasReadResponse == 1

		if ga.HasReadResponse {
			respondingCount++
		}
		if lastSeenUnix >= fiveMinAgo {
			activeLast5MinGA++
		}
		if lastSeenUnix >= oneHourAgo {
			activeLast1HourGA++
		}

		groupAddresses = append(groupAddresses, ga)
	}

	// Query devices (individual addresses)
	deviceRows, err := db.QueryContext(ctx, `
		SELECT individual_address, last_seen, message_count
		FROM knx_devices
		ORDER BY last_seen DESC
		LIMIT 500
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to query devices",
		})
		return
	}
	defer deviceRows.Close()

	var devices []DiscoveredDevice
	var activeLast5MinDev int

	for deviceRows.Next() {
		var dev DiscoveredDevice
		var lastSeenUnix int64

		if err := deviceRows.Scan(&dev.IndividualAddress, &lastSeenUnix, &dev.MessageCount); err != nil {
			continue
		}

		lastSeenTime := time.Unix(lastSeenUnix, 0)
		dev.LastSeen = lastSeenTime.Format(time.RFC3339)
		dev.LastSeenAgo = formatDuration(now.Sub(lastSeenTime))

		if lastSeenUnix >= fiveMinAgo {
			activeLast5MinDev++
		}

		devices = append(devices, dev)
	}

	// Build response
	data := DiscoveryData{
		GroupAddresses: groupAddresses,
		Devices:        devices,
		Summary: DiscoverySummary{
			TotalGroupAddresses: len(groupAddresses),
			TotalDevices:        len(devices),
			RespondingAddresses: respondingCount,
			ActiveLast5Min:      activeLast5MinGA + activeLast5MinDev,
			ActiveLast1Hour:     activeLast1HourGA,
		},
	}

	// Handle empty slices for clean JSON
	if data.GroupAddresses == nil {
		data.GroupAddresses = []DiscoveredGA{}
	}
	if data.Devices == nil {
		data.Devices = []DiscoveredDevice{}
	}

	writeJSON(w, http.StatusOK, data)
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return formatInt(mins) + " mins ago"
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return formatInt(hours) + " hours ago"
	}
	days := int(d.Hours() / 24) //nolint:mnd // 24 hours per day
	if days == 1 {
		return "1 day ago"
	}
	return formatInt(days) + " days ago"
}

// formatInt converts an int to string without importing strconv.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatInt(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// getDB returns the database handle, checking if it implements the interface.
func (s *Server) getDB() *sql.DB {
	if s.db == nil {
		return nil
	}
	return s.db.SQLDB()
}
