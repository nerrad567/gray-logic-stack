package api

import (
	"net/http"
	"runtime"
	"time"
)

// SystemMetrics represents the complete system metrics response.
type SystemMetrics struct {
	Timestamp     string          `json:"timestamp"`
	Version       string          `json:"version"`
	UptimeSeconds int64           `json:"uptime_seconds"`
	Runtime       RuntimeMetrics  `json:"runtime"`
	WebSocket     WSMetrics       `json:"websocket"`
	MQTT          MQTTMetrics     `json:"mqtt"`
	KNXBridge     *KNXMetrics     `json:"knx_bridge,omitempty"`
	Devices       DeviceMetrics   `json:"devices"`
	Database      DatabaseMetrics `json:"database"`
}

// RuntimeMetrics contains Go runtime statistics.
type RuntimeMetrics struct {
	Goroutines    int     `json:"goroutines"`
	MemoryAllocMB float64 `json:"memory_alloc_mb"`
	MemoryTotalMB float64 `json:"memory_total_mb"`
	NumGC         uint32  `json:"num_gc"`
}

// WSMetrics contains WebSocket hub statistics.
type WSMetrics struct {
	ConnectedClients int `json:"connected_clients"`
}

// MQTTMetrics contains MQTT client statistics.
type MQTTMetrics struct {
	Connected bool `json:"connected"`
}

// KNXMetrics contains KNX bridge statistics.
type KNXMetrics struct {
	Connected      bool   `json:"connected"`
	Status         string `json:"status"`
	TelegramsTx    uint64 `json:"telegrams_tx"`
	TelegramsRx    uint64 `json:"telegrams_rx"`
	DevicesManaged int    `json:"devices_managed"`
}

// DeviceMetrics contains device registry statistics.
type DeviceMetrics struct {
	Total    int            `json:"total"`
	ByHealth map[string]int `json:"by_health"`
	ByDomain map[string]int `json:"by_domain"`
}

// DatabaseMetrics contains database connection pool statistics.
type DatabaseMetrics struct {
	OpenConnections int   `json:"open_connections"`
	InUse           int   `json:"in_use"`
	Idle            int   `json:"idle"`
	WaitCount       int64 `json:"wait_count"`
}

// handleMetrics returns comprehensive system metrics.
func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	// Collect runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Build metrics response
	metrics := SystemMetrics{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Version:       s.version,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Runtime: RuntimeMetrics{
			Goroutines:    runtime.NumGoroutine(),
			MemoryAllocMB: float64(memStats.Alloc) / 1024 / 1024,
			MemoryTotalMB: float64(memStats.TotalAlloc) / 1024 / 1024,
			NumGC:         memStats.NumGC,
		},
		WebSocket: WSMetrics{
			ConnectedClients: s.hub.ClientCount(),
		},
	}

	// MQTT metrics (if available)
	if s.mqtt != nil {
		metrics.MQTT = MQTTMetrics{
			Connected: s.mqtt.IsConnected(),
		}
	}

	// KNX bridge metrics (if available)
	if s.knxMetricsProvider != nil {
		knxStats := s.knxMetricsProvider.GetMetrics()
		metrics.KNXBridge = &KNXMetrics{
			Connected:      knxStats.Connected,
			Status:         knxStats.Status,
			TelegramsTx:    knxStats.TelegramsTx,
			TelegramsRx:    knxStats.TelegramsRx,
			DevicesManaged: knxStats.DevicesManaged,
		}
	}

	// Device registry stats
	regStats := s.registry.GetStats()
	metrics.Devices = DeviceMetrics{
		Total:    regStats.TotalDevices,
		ByHealth: make(map[string]int),
		ByDomain: make(map[string]int),
	}
	for health, count := range regStats.ByHealthStatus {
		metrics.Devices.ByHealth[string(health)] = count
	}
	for domain, count := range regStats.ByDomain {
		metrics.Devices.ByDomain[string(domain)] = count
	}

	// Database stats (if available)
	if s.db != nil {
		dbStats := s.db.Stats()
		metrics.Database = DatabaseMetrics{
			OpenConnections: dbStats.OpenConnections,
			InUse:           dbStats.InUse,
			Idle:            dbStats.Idle,
			WaitCount:       dbStats.WaitCount,
		}
	}

	writeJSON(w, http.StatusOK, metrics)
}
