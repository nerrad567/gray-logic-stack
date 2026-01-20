package knx

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// HealthReporter manages periodic health status reporting.
// It publishes health messages to MQTT at regular intervals.
type HealthReporter struct {
	bridgeID   string
	version    string
	startTime  time.Time
	interval   time.Duration
	publisher  HealthPublisher
	knxdClient Connector

	// Device count (updated externally)
	deviceCount   int
	deviceCountMu sync.RWMutex

	// Shutdown coordination (stopOnce prevents double-close panics)
	done     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once

	// Logger (optional)
	logger   Logger
	loggerMu sync.RWMutex
}

// HealthPublisher is the interface for publishing health messages.
// This is typically implemented by an MQTT client.
type HealthPublisher interface {
	// Publish sends a message to a topic with the specified QoS and retention.
	Publish(topic string, payload []byte, qos byte, retained bool) error

	// IsConnected returns true if the publisher is connected.
	IsConnected() bool
}

// HealthReporterConfig holds configuration for the health reporter.
type HealthReporterConfig struct {
	// BridgeID is the bridge identifier for health messages.
	BridgeID string

	// Version is the bridge software version.
	Version string

	// Interval is how often to publish health status.
	// Default: 30 seconds.
	Interval time.Duration

	// Publisher is the MQTT client for publishing messages.
	Publisher HealthPublisher

	// KNXDClient provides connection statistics.
	KNXDClient Connector
}

// NewHealthReporter creates a new health reporter.
//
// Parameters:
//   - cfg: Configuration for the health reporter
//
// Returns:
//   - *HealthReporter: Ready to start (call Start to begin reporting)
func NewHealthReporter(cfg HealthReporterConfig) *HealthReporter {
	interval := cfg.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &HealthReporter{
		bridgeID:   cfg.BridgeID,
		version:    cfg.Version,
		startTime:  time.Now(),
		interval:   interval,
		publisher:  cfg.Publisher,
		knxdClient: cfg.KNXDClient,
		done:       make(chan struct{}),
	}
}

// Start begins periodic health reporting.
// Must be called after creation. Call Stop to shut down.
//
// Parameters:
//   - ctx: Context for cancellation (will stop reporting when cancelled)
func (h *HealthReporter) Start(ctx context.Context) {
	h.wg.Add(1)
	go h.reportLoop(ctx)
}

// Stop gracefully stops health reporting.
// Publishes a final "stopping" status before returning.
// Safe to call multiple times (uses sync.Once).
func (h *HealthReporter) Stop() {
	h.stopOnce.Do(func() {
		// Signal shutdown
		close(h.done)

		// Wait for report loop to finish
		h.wg.Wait()

		// Publish final stopping status (best-effort, ignore errors)
		//nolint:errcheck // Best-effort during shutdown, nothing we can do if it fails
		h.publishStatus(HealthStopping, "")
	})
}

// SetDeviceCount updates the managed device count.
// This is called when device configuration changes.
func (h *HealthReporter) SetDeviceCount(count int) {
	h.deviceCountMu.Lock()
	h.deviceCount = count
	h.deviceCountMu.Unlock()
}

// SetLogger sets the logger for this reporter.
func (h *HealthReporter) SetLogger(logger Logger) {
	h.loggerMu.Lock()
	h.logger = logger
	h.loggerMu.Unlock()
}

// PublishStarting publishes a "starting" status.
// Called during bridge initialization.
func (h *HealthReporter) PublishStarting() error {
	return h.publishStatus(HealthStarting, "bridge starting")
}

// PublishNow publishes the current health status immediately.
// Useful for forcing an update after a significant event.
func (h *HealthReporter) PublishNow() error {
	status, reason := h.determineStatus()
	return h.publishStatus(status, reason)
}

// GetLWTPayload returns the Last Will and Testament message payload.
// This should be set as the MQTT will message during connection.
func (h *HealthReporter) GetLWTPayload() ([]byte, error) {
	msg := NewLWTMessage(h.bridgeID)
	return json.Marshal(msg)
}

// GetLWTTopic returns the topic for the Last Will and Testament.
func (h *HealthReporter) GetLWTTopic() string {
	return HealthTopic()
}

// reportLoop runs the periodic health reporting.
func (h *HealthReporter) reportLoop(ctx context.Context) {
	defer h.wg.Done()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	// Publish initial status
	if err := h.PublishNow(); err != nil {
		h.logError("failed to publish initial health", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.done:
			return
		case <-ticker.C:
			if err := h.PublishNow(); err != nil {
				h.logError("failed to publish health", err)
			}
		}
	}
}

// determineStatus evaluates the current bridge status.
func (h *HealthReporter) determineStatus() (HealthStatus, string) {
	// Check MQTT connection
	if h.publisher == nil || !h.publisher.IsConnected() {
		return HealthDegraded, "MQTT disconnected"
	}

	// Check knxd connection
	if h.knxdClient == nil || !h.knxdClient.IsConnected() {
		return HealthDegraded, "knxd disconnected"
	}

	// All good
	return HealthHealthy, ""
}

// publishStatus publishes a health status message.
func (h *HealthReporter) publishStatus(status HealthStatus, reason string) error {
	if h.publisher == nil {
		return nil // No publisher configured
	}

	// Get device count
	h.deviceCountMu.RLock()
	deviceCount := h.deviceCount
	h.deviceCountMu.RUnlock()

	// Get knxd stats
	var stats KNXDStats
	if h.knxdClient != nil {
		stats = h.knxdClient.Stats()
	}

	// Build message
	msg := NewHealthMessage(h.bridgeID, h.version, status, stats, deviceCount, h.startTime)
	if reason != "" {
		msg.Reason = reason
	}

	// Set connection address
	if msg.Connection != nil {
		msg.Connection.Address = h.getKNXDAddress()
	}

	// Serialise to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish (QoS 1, retained)
	return h.publisher.Publish(HealthTopic(), payload, 1, true)
}

// getKNXDAddress returns the knxd connection address.
func (h *HealthReporter) getKNXDAddress() string {
	// This would come from config in the full implementation
	return DefaultKNXDConnection
}

// logError logs an error if logger is set.
func (h *HealthReporter) logError(msg string, err error) {
	h.loggerMu.RLock()
	logger := h.logger
	h.loggerMu.RUnlock()

	if logger != nil {
		logger.Error(msg, "error", err)
	}
}
