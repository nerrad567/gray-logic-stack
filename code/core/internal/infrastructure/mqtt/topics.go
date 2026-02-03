package mqtt

import "fmt"

// Topic prefixes per Gray Logic MQTT specification.
// See docs/protocols/mqtt.md for complete topic hierarchy.
//
// All bridge topics use the flat scheme: graylogic/{category}/{protocol}/{address}
// This matches the KNX bridge's messages.go and all runtime subscribers.
const (
	// TopicPrefixBridge is the base for all bridge topics.
	// Flat scheme: graylogic/{category}/{protocol}/{address_or_id}
	TopicPrefixBridge = "graylogic"

	// TopicPrefixCore is the base for all core topics.
	TopicPrefixCore = "graylogic/core"

	// TopicPrefixSystem is the base for system topics.
	TopicPrefixSystem = "graylogic/system"

	// TopicPrefixUI is the base for UI-specific topics.
	TopicPrefixUI = "graylogic/ui"
)

// Topics provides builders for Gray Logic MQTT topics.
// Using these helpers ensures consistent topic naming across the codebase.
//
// Bridge topics use the flat scheme matching the KNX bridge's messages.go:
//
//	topics := mqtt.Topics{}
//	stateTopic := topics.BridgeState("knx", "light-living-main")
//	// Returns: "graylogic/state/knx/light-living-main"
type Topics struct{}

// =============================================================================
// Bridge Topics
// =============================================================================

// BridgeState returns the topic for device state updates from a bridge.
//
// Example: graylogic/state/knx/light-living-main
func (Topics) BridgeState(protocol, address string) string {
	return fmt.Sprintf("%s/state/%s/%s", TopicPrefixBridge, protocol, address)
}

// BridgeCommand returns the topic for commands to a bridge.
//
// Example: graylogic/command/knx/light-living-main
func (Topics) BridgeCommand(protocol, address string) string {
	return fmt.Sprintf("%s/command/%s/%s", TopicPrefixBridge, protocol, address)
}

// BridgeAck returns the topic for command acknowledgements from a bridge.
//
// Example: graylogic/ack/knx/light-living-main
func (Topics) BridgeAck(protocol, address string) string {
	return fmt.Sprintf("%s/ack/%s/%s", TopicPrefixBridge, protocol, address)
}

// BridgeResponse returns the topic for request responses from a bridge.
//
// Example: graylogic/response/knx/req-abc123
func (Topics) BridgeResponse(protocol, requestID string) string {
	return fmt.Sprintf("%s/response/%s/%s", TopicPrefixBridge, protocol, requestID)
}

// BridgeRequest returns the topic for requests to a bridge.
//
// Example: graylogic/request/knx/req-abc123
func (Topics) BridgeRequest(protocol, requestID string) string {
	return fmt.Sprintf("%s/request/%s/%s", TopicPrefixBridge, protocol, requestID)
}

// BridgeHealth returns the topic for bridge health status.
//
// Example: graylogic/health/knx
func (Topics) BridgeHealth(protocol string) string {
	return fmt.Sprintf("%s/health/%s", TopicPrefixBridge, protocol)
}

// BridgeDiscovery returns the topic for device discovery from a bridge.
//
// Example: graylogic/discovery/knx
func (Topics) BridgeDiscovery(protocol string) string {
	return fmt.Sprintf("%s/discovery/%s", TopicPrefixBridge, protocol)
}

// BridgeConfig returns the topic for configuration updates to a bridge.
//
// Example: graylogic/config/knx
func (Topics) BridgeConfig(protocol string) string {
	return fmt.Sprintf("%s/config/%s", TopicPrefixBridge, protocol)
}

// =============================================================================
// Core Topics
// =============================================================================

// CoreDeviceState returns the canonical device state topic.
// This is the authoritative state published by Core after processing bridge updates.
//
// Example: graylogic/core/device/light-living-main/state
func (Topics) CoreDeviceState(deviceID string) string {
	return fmt.Sprintf("%s/device/%s/state", TopicPrefixCore, deviceID)
}

// CoreEvent returns the topic for system events.
//
// Example: graylogic/core/event/device_state_changed
func (Topics) CoreEvent(eventType string) string {
	return fmt.Sprintf("%s/event/%s", TopicPrefixCore, eventType)
}

// CoreSceneActivated returns the topic for scene activation events.
//
// Example: graylogic/core/scene/cinema-mode/activated
func (Topics) CoreSceneActivated(sceneID string) string {
	return fmt.Sprintf("%s/scene/%s/activated", TopicPrefixCore, sceneID)
}

// CoreSceneProgress returns the topic for scene execution progress.
//
// Example: graylogic/core/scene/cinema-mode/progress
func (Topics) CoreSceneProgress(sceneID string) string {
	return fmt.Sprintf("%s/scene/%s/progress", TopicPrefixCore, sceneID)
}

// CoreAutomationFired returns the topic for automation rule triggers.
//
// Example: graylogic/core/automation/rule-sunrise-blinds/fired
func (Topics) CoreAutomationFired(ruleID string) string {
	return fmt.Sprintf("%s/automation/%s/fired", TopicPrefixCore, ruleID)
}

// CoreAlert returns the topic for system alerts.
//
// Example: graylogic/core/alert/alert-dali-offline
func (Topics) CoreAlert(alertID string) string {
	return fmt.Sprintf("%s/alert/%s", TopicPrefixCore, alertID)
}

// CoreMode returns the topic for mode changes.
//
// Example: graylogic/core/mode
func (Topics) CoreMode() string {
	return fmt.Sprintf("%s/mode", TopicPrefixCore)
}

// =============================================================================
// System Topics
// =============================================================================

// SystemStatus returns the system status topic.
//
// Example: graylogic/system/status
func (Topics) SystemStatus() string {
	return fmt.Sprintf("%s/status", TopicPrefixSystem)
}

// SystemTime returns the time sync topic.
//
// Example: graylogic/system/time
func (Topics) SystemTime() string {
	return fmt.Sprintf("%s/time", TopicPrefixSystem)
}

// SystemShutdown returns the shutdown signal topic.
//
// Example: graylogic/system/shutdown
func (Topics) SystemShutdown() string {
	return fmt.Sprintf("%s/shutdown", TopicPrefixSystem)
}

// =============================================================================
// UI Topics
// =============================================================================

// UINotification returns the notification topic for a specific UI client.
//
// Example: graylogic/ui/panel-kitchen/notification
func (Topics) UINotification(clientID string) string {
	return fmt.Sprintf("%s/%s/notification", TopicPrefixUI, clientID)
}

// UIPresence returns the presence topic for a specific UI client.
//
// Example: graylogic/ui/panel-kitchen/presence
func (Topics) UIPresence(clientID string) string {
	return fmt.Sprintf("%s/%s/presence", TopicPrefixUI, clientID)
}

// =============================================================================
// Wildcard Patterns for Subscriptions
// =============================================================================

// AllBridgeStates returns a pattern matching all bridge state updates.
//
// Pattern: graylogic/state/+/+
func (Topics) AllBridgeStates() string {
	return fmt.Sprintf("%s/state/+/+", TopicPrefixBridge)
}

// AllBridgeCommands returns a pattern matching all commands to bridges.
//
// Pattern: graylogic/command/+/+
func (Topics) AllBridgeCommands() string {
	return fmt.Sprintf("%s/command/+/+", TopicPrefixBridge)
}

// AllBridgeAcks returns a pattern matching all bridge acknowledgements.
//
// Pattern: graylogic/ack/+/+
func (Topics) AllBridgeAcks() string {
	return fmt.Sprintf("%s/ack/+/+", TopicPrefixBridge)
}

// AllBridgeHealth returns a pattern matching all bridge health updates.
//
// Pattern: graylogic/health/+
func (Topics) AllBridgeHealth() string {
	return fmt.Sprintf("%s/health/+", TopicPrefixBridge)
}

// AllBridgeDiscovery returns a pattern matching all bridge discovery topics.
//
// Pattern: graylogic/discovery/+
func (Topics) AllBridgeDiscovery() string {
	return fmt.Sprintf("%s/discovery/+", TopicPrefixBridge)
}

// AllBridgeRequests returns a pattern matching all bridge request topics.
//
// Pattern: graylogic/request/+/+
func (Topics) AllBridgeRequests() string {
	return fmt.Sprintf("%s/request/+/+", TopicPrefixBridge)
}

// AllBridgeResponses returns a pattern matching all bridge response topics.
//
// Pattern: graylogic/response/+/+
func (Topics) AllBridgeResponses() string {
	return fmt.Sprintf("%s/response/+/+", TopicPrefixBridge)
}

// AllBridgeConfigs returns a pattern matching all bridge config topics.
//
// Pattern: graylogic/config/+
func (Topics) AllBridgeConfigs() string {
	return fmt.Sprintf("%s/config/+", TopicPrefixBridge)
}

// AllCoreDeviceStates returns a pattern matching all canonical device states.
//
// Pattern: graylogic/core/device/+/state
func (Topics) AllCoreDeviceStates() string {
	return fmt.Sprintf("%s/device/+/state", TopicPrefixCore)
}

// AllCoreEvents returns a pattern matching all core events.
//
// Pattern: graylogic/core/event/+
func (Topics) AllCoreEvents() string {
	return fmt.Sprintf("%s/event/+", TopicPrefixCore)
}

// AllCoreAlerts returns a pattern matching all alerts.
//
// Pattern: graylogic/core/alert/+
func (Topics) AllCoreAlerts() string {
	return fmt.Sprintf("%s/alert/+", TopicPrefixCore)
}

// AllTopics returns a pattern matching all Gray Logic topics.
// Use with caution - this receives ALL traffic.
//
// Pattern: graylogic/#
func (Topics) AllTopics() string {
	return "graylogic/#"
}
