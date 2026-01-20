package knx

import (
	"encoding/json"
	"fmt"
	"time"
)

// MQTT message types for communication between Gray Logic Core and KNX Bridge.
// These types implement the bridge interface specification (docs/architecture/bridge-interface.md).

// CommandMessage is sent from Core to Bridge to execute a device command.
// Topic: graylogic/command/knx/{address}
type CommandMessage struct {
	// ID uniquely identifies this command for correlation with acknowledgments.
	ID string `json:"id"`

	// Timestamp is when the command was issued (UTC, ISO8601).
	Timestamp time.Time `json:"timestamp"`

	// DeviceID is the Gray Logic device identifier.
	DeviceID string `json:"device_id"`

	// Command is the command name (e.g., "on", "off", "dim", "set_position").
	Command string `json:"command"`

	// Parameters contains command-specific values.
	// Examples:
	//   {"level": 50} for dim
	//   {"position": 75, "tilt": 45} for blinds
	Parameters map[string]any `json:"parameters,omitempty"`

	// Source indicates where the command originated.
	// Values: "api", "automation", "voice", "scene"
	Source string `json:"source"`

	// UserID is the user who triggered the command (if applicable).
	UserID string `json:"user_id,omitempty"`
}

// AckStatus represents the acknowledgment status of a command.
type AckStatus string

const (
	// AckAccepted indicates the command was received and sent to the device.
	AckAccepted AckStatus = "accepted"

	// AckQueued indicates the command was received but waiting to send (device busy).
	AckQueued AckStatus = "queued"

	// AckFailed indicates the command could not be executed.
	AckFailed AckStatus = "failed"

	// AckTimeout indicates the device did not respond within the timeout.
	AckTimeout AckStatus = "timeout"
)

// AckMessage is sent from Bridge to Core to acknowledge a command.
// Topic: graylogic/ack/knx/{address}
type AckMessage struct {
	// CommandID is the ID from the original command.
	CommandID string `json:"command_id"`

	// Timestamp is when the acknowledgment was sent (UTC, ISO8601).
	Timestamp time.Time `json:"timestamp"`

	// DeviceID is the Gray Logic device identifier.
	DeviceID string `json:"device_id"`

	// Status indicates the acknowledgment status.
	Status AckStatus `json:"status"`

	// Protocol is the protocol identifier ("knx").
	Protocol string `json:"protocol"`

	// Address is the protocol-specific address (e.g., "1/2/3").
	Address string `json:"address"`

	// Error contains details if status is "failed" or "timeout".
	Error *AckError `json:"error,omitempty"`
}

// AckError contains error details for failed commands.
type AckError struct {
	// Code is the error code (e.g., "DEVICE_UNREACHABLE", "INVALID_COMMAND").
	Code string `json:"code"`

	// Message is a human-readable error description.
	Message string `json:"message"`

	// Retries is the number of retry attempts made.
	Retries int `json:"retries,omitempty"`
}

// Error codes for command failures.
const (
	ErrCodeDeviceUnreachable = "DEVICE_UNREACHABLE"
	ErrCodeInvalidCommand    = "INVALID_COMMAND"
	ErrCodeInvalidParameters = "INVALID_PARAMETERS"
	ErrCodeProtocolError     = "PROTOCOL_ERROR"
	ErrCodeTimeout           = "TIMEOUT"
	ErrCodeNotConfigured     = "NOT_CONFIGURED"
	ErrCodeBridgeError       = "BRIDGE_ERROR"
)

// StateMessage is sent from Bridge to Core when device state changes.
// Topic: graylogic/state/knx/{address}
// QoS: 1, Retained: Yes
type StateMessage struct {
	// DeviceID is the Gray Logic device identifier.
	DeviceID string `json:"device_id"`

	// Timestamp is when the state was observed (UTC, ISO8601).
	Timestamp time.Time `json:"timestamp"`

	// State contains the current device state.
	// Structure depends on device type:
	//   Light: {"on": true, "level": 50}
	//   Blind: {"position": 50, "tilt": 30}
	//   Sensor: {"temperature": 21.5, "humidity": 45.0}
	State map[string]any `json:"state"`

	// Protocol is the protocol identifier ("knx").
	Protocol string `json:"protocol"`

	// Address is the protocol-specific address (e.g., "1/2/3").
	Address string `json:"address"`
}

// HealthStatus represents the operational status of the bridge.
type HealthStatus string

const (
	// HealthHealthy indicates the bridge is operating normally.
	HealthHealthy HealthStatus = "healthy"

	// HealthDegraded indicates the bridge is operating with issues.
	HealthDegraded HealthStatus = "degraded"

	// HealthUnhealthy indicates the bridge is not operating correctly.
	HealthUnhealthy HealthStatus = "unhealthy"

	// HealthOffline indicates the bridge is not connected (from LWT).
	HealthOffline HealthStatus = "offline"

	// HealthStarting indicates the bridge is starting up.
	HealthStarting HealthStatus = "starting"

	// HealthStopping indicates the bridge is shutting down.
	HealthStopping HealthStatus = "stopping"
)

// HealthMessage is sent from Bridge to Core to report operational status.
// Topic: graylogic/health/knx
// QoS: 1, Retained: Yes
// Interval: Every 30 seconds
type HealthMessage struct {
	// Bridge is the bridge identifier (e.g., "knx").
	Bridge string `json:"bridge"`

	// Timestamp is when the health status was generated (UTC, ISO8601).
	Timestamp time.Time `json:"timestamp"`

	// Status indicates the current operational status.
	Status HealthStatus `json:"status"`

	// Version is the bridge software version.
	Version string `json:"version"`

	// UptimeSeconds is how long the bridge has been running.
	UptimeSeconds int64 `json:"uptime_seconds"`

	// Connection contains knxd connection details.
	Connection *ConnectionStatus `json:"connection,omitempty"`

	// Statistics contains operational metrics.
	Statistics *BridgeStatistics `json:"statistics,omitempty"`

	// DevicesManaged is the number of configured devices.
	DevicesManaged int `json:"devices_managed"`

	// Reason explains the status (especially for offline/degraded).
	Reason string `json:"reason,omitempty"`
}

// ConnectionStatus describes the knxd connection state.
type ConnectionStatus struct {
	// Status is the connection status ("connected", "disconnected", "connecting").
	Status string `json:"status"`

	// Address is the knxd connection address.
	Address string `json:"address"`

	// ConnectedSince is when the connection was established.
	ConnectedSince *time.Time `json:"connected_since,omitempty"`
}

// BridgeStatistics contains operational metrics.
type BridgeStatistics struct {
	// MessagesReceived is the total number of KNX telegrams received.
	MessagesReceived uint64 `json:"messages_received"`

	// MessagesSent is the total number of KNX telegrams sent.
	MessagesSent uint64 `json:"messages_sent"`

	// Errors is the total number of errors encountered.
	Errors uint64 `json:"errors"`
}

// RequestMessage is sent from Core to Bridge for request/response operations.
// Topic: graylogic/request/knx/{request_id}
type RequestMessage struct {
	// RequestID uniquely identifies this request for correlation.
	RequestID string `json:"request_id"`

	// Timestamp is when the request was issued (UTC, ISO8601).
	Timestamp time.Time `json:"timestamp"`

	// Action is the requested operation.
	// Values: "read_state", "read_all", "discover", "reconfigure", "restart"
	Action string `json:"action"`

	// DeviceID is the target device (for device-specific actions).
	DeviceID string `json:"device_id,omitempty"`

	// Parameters contains action-specific values.
	Parameters map[string]any `json:"parameters,omitempty"`
}

// ResponseMessage is sent from Bridge to Core in response to a request.
// Topic: graylogic/response/knx/{request_id}
type ResponseMessage struct {
	// RequestID is the ID from the original request.
	RequestID string `json:"request_id"`

	// Timestamp is when the response was generated (UTC, ISO8601).
	Timestamp time.Time `json:"timestamp"`

	// Success indicates whether the request succeeded.
	Success bool `json:"success"`

	// Data contains the response payload (if successful).
	Data map[string]any `json:"data,omitempty"`

	// Error contains error details (if failed).
	Error *ResponseError `json:"error,omitempty"`
}

// ResponseError contains error details for failed requests.
type ResponseError struct {
	// Code is the error code.
	Code string `json:"code"`

	// Message is a human-readable error description.
	Message string `json:"message"`

	// Details contains additional error context.
	Details map[string]any `json:"details,omitempty"`
}

// DiscoveryMessage is sent from Bridge to Core to announce discovered devices.
// Topic: graylogic/discovery/knx
type DiscoveryMessage struct {
	// Timestamp is when discovery was performed (UTC, ISO8601).
	Timestamp time.Time `json:"timestamp"`

	// Bridge is the bridge identifier.
	Bridge string `json:"bridge"`

	// Devices contains the discovered devices.
	Devices []DiscoveredDevice `json:"devices"`
}

// DiscoveredDevice represents a device found during discovery.
type DiscoveredDevice struct {
	// Protocol is the protocol identifier.
	Protocol string `json:"protocol"`

	// Address is the protocol-specific address.
	Address string `json:"address"`

	// Type is the device type (e.g., "light_dimmer", "blind", "sensor").
	Type string `json:"type"`

	// Capabilities lists the device capabilities (e.g., ["on_off", "dim"]).
	Capabilities []string `json:"capabilities"`

	// Manufacturer is the device manufacturer (if known).
	Manufacturer string `json:"manufacturer,omitempty"`

	// Product is the product model (if known).
	Product string `json:"product,omitempty"`

	// SuggestedName is a suggested display name for the device.
	SuggestedName string `json:"suggested_name,omitempty"`
}

// JSON marshalling helpers

// MarshalJSON marshals a CommandMessage to JSON.
func (m *CommandMessage) MarshalJSON() ([]byte, error) {
	type Alias CommandMessage
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(m),
		Timestamp: m.Timestamp.UTC().Format(time.RFC3339),
	})
}

// UnmarshalJSON unmarshals a CommandMessage from JSON.
func (m *CommandMessage) UnmarshalJSON(data []byte) error {
	type Alias CommandMessage
	aux := &struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return fmt.Errorf("unmarshal command message: %w", err)
	}
	if aux.Timestamp != "" {
		t, err := time.Parse(time.RFC3339, aux.Timestamp)
		if err != nil {
			return fmt.Errorf("parse timestamp: %w", err)
		}
		m.Timestamp = t
	}
	return nil
}

// NewAckMessage creates an acknowledgment message for a command.
func NewAckMessage(cmd CommandMessage, status AckStatus, address string) AckMessage {
	return AckMessage{
		CommandID: cmd.ID,
		Timestamp: time.Now().UTC(),
		DeviceID:  cmd.DeviceID,
		Status:    status,
		Protocol:  "knx",
		Address:   address,
	}
}

// NewAckError creates an acknowledgment with error details.
func NewAckError(cmd CommandMessage, address, code, message string, retries int) AckMessage {
	status := AckFailed
	if code == ErrCodeTimeout {
		status = AckTimeout
	}
	return AckMessage{
		CommandID: cmd.ID,
		Timestamp: time.Now().UTC(),
		DeviceID:  cmd.DeviceID,
		Status:    status,
		Protocol:  "knx",
		Address:   address,
		Error: &AckError{
			Code:    code,
			Message: message,
			Retries: retries,
		},
	}
}

// NewStateMessage creates a state message for a device.
func NewStateMessage(deviceID, address string, state map[string]any) StateMessage {
	return StateMessage{
		DeviceID:  deviceID,
		Timestamp: time.Now().UTC(),
		State:     state,
		Protocol:  "knx",
		Address:   address,
	}
}

// NewHealthMessage creates a health status message.
func NewHealthMessage(bridgeID, version string, status HealthStatus, stats KNXDStats, deviceCount int, startTime time.Time) HealthMessage {
	msg := HealthMessage{
		Bridge:         bridgeID,
		Timestamp:      time.Now().UTC(),
		Status:         status,
		Version:        version,
		UptimeSeconds:  int64(time.Since(startTime).Seconds()),
		DevicesManaged: deviceCount,
	}

	if stats.Connected {
		connectedSince := stats.LastActivity // Approximation
		msg.Connection = &ConnectionStatus{
			Status:         "connected",
			ConnectedSince: &connectedSince,
		}
	} else {
		msg.Connection = &ConnectionStatus{
			Status: "disconnected",
		}
	}

	msg.Statistics = &BridgeStatistics{
		MessagesReceived: stats.TelegramsRx,
		MessagesSent:     stats.TelegramsTx,
		Errors:           stats.ErrorsTotal,
	}

	return msg
}

// NewLWTMessage creates a Last Will and Testament message for MQTT.
// This message is published by the broker if the bridge disconnects unexpectedly.
func NewLWTMessage(bridgeID string) HealthMessage {
	return HealthMessage{
		Bridge:    bridgeID,
		Timestamp: time.Now().UTC(),
		Status:    HealthOffline,
		Reason:    "unexpected_disconnect",
	}
}

// Topic helpers

const (
	// TopicPrefix is the base topic for all Gray Logic messages.
	TopicPrefix = "graylogic"
)

// CommandTopic returns the MQTT topic for commands to a specific address.
// Example: graylogic/command/knx/1%2F0%2F1
func CommandTopic(address string) string {
	return fmt.Sprintf("%s/command/knx/%s", TopicPrefix, EncodeTopicAddress(address))
}

// AckTopic returns the MQTT topic for command acknowledgments.
// Example: graylogic/ack/knx/1%2F0%2F1
func AckTopic(address string) string {
	return fmt.Sprintf("%s/ack/knx/%s", TopicPrefix, EncodeTopicAddress(address))
}

// StateTopic returns the MQTT topic for state updates.
// Example: graylogic/state/knx/1%2F0%2F1
func StateTopic(address string) string {
	return fmt.Sprintf("%s/state/knx/%s", TopicPrefix, EncodeTopicAddress(address))
}

// HealthTopic returns the MQTT topic for health status.
// Example: graylogic/health/knx
func HealthTopic() string {
	return fmt.Sprintf("%s/health/knx", TopicPrefix)
}

// RequestTopic returns the MQTT topic for requests.
// Example: graylogic/request/knx/req-123
func RequestTopic(requestID string) string {
	return fmt.Sprintf("%s/request/knx/%s", TopicPrefix, requestID)
}

// ResponseTopic returns the MQTT topic for responses.
// Example: graylogic/response/knx/req-123
func ResponseTopic(requestID string) string {
	return fmt.Sprintf("%s/response/knx/%s", TopicPrefix, requestID)
}

// DiscoveryTopic returns the MQTT topic for device discovery.
// Example: graylogic/discovery/knx
func DiscoveryTopic() string {
	return fmt.Sprintf("%s/discovery/knx", TopicPrefix)
}

// CommandSubscribeTopic returns the MQTT subscription pattern for all commands.
// Example: graylogic/command/knx/#
func CommandSubscribeTopic() string {
	return fmt.Sprintf("%s/command/knx/#", TopicPrefix)
}

// RequestSubscribeTopic returns the MQTT subscription pattern for all requests.
// Example: graylogic/request/knx/#
func RequestSubscribeTopic() string {
	return fmt.Sprintf("%s/request/knx/#", TopicPrefix)
}

// ConfigTopic returns the MQTT topic for configuration updates.
// Example: graylogic/config/knx
func ConfigTopic() string {
	return fmt.Sprintf("%s/config/knx", TopicPrefix)
}

// encodedSlashLen is the length of URL-encoded slash (%2F).
const encodedSlashLen = 3

// EncodeTopicAddress URL-encodes an address for use in MQTT topics.
// KNX addresses contain slashes which must be encoded.
// Example: "1/2/3" → "1%2F2%2F3"
func EncodeTopicAddress(address string) string {
	// Simple encoding for KNX addresses (replace / with %2F)
	result := make([]byte, 0, len(address)*encodedSlashLen)
	for i := 0; i < len(address); i++ {
		if address[i] == '/' {
			result = append(result, '%', '2', 'F')
		} else {
			result = append(result, address[i])
		}
	}
	return string(result)
}

// DecodeTopicAddress decodes a URL-encoded address from an MQTT topic.
// Example: "1%2F2%2F3" → "1/2/3"
func DecodeTopicAddress(encoded string) string {
	result := make([]byte, 0, len(encoded))
	for i := 0; i < len(encoded); i++ {
		if i+2 < len(encoded) && encoded[i] == '%' && encoded[i+1] == '2' && encoded[i+2] == 'F' {
			result = append(result, '/')
			i += 2
		} else {
			result = append(result, encoded[i])
		}
	}
	return string(result)
}
