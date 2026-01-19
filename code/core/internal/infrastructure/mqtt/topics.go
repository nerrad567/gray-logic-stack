package mqtt

import "fmt"

// Topic prefixes per Gray Logic MQTT specification.
// See docs/protocols/mqtt.md for complete topic hierarchy.
const (
	// TopicPrefixBridge is the base for all bridge topics.
	TopicPrefixBridge = "graylogic/bridge"

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
// Example:
//
//	topics := mqtt.Topics{}
//	stateTopic := topics.BridgeState("knx-bridge-01", "light-living-main")
//	// Returns: "graylogic/bridge/knx-bridge-01/state/light-living-main"
type Topics struct{}

// =============================================================================
// Bridge Topics
// =============================================================================

// BridgeState returns the topic for device state updates from a bridge.
//
// Example: graylogic/bridge/knx-bridge-01/state/light-living-main
func (Topics) BridgeState(bridgeID, deviceID string) string {
	return fmt.Sprintf("%s/%s/state/%s", TopicPrefixBridge, bridgeID, deviceID)
}

// BridgeCommand returns the topic for commands to a bridge.
//
// Example: graylogic/bridge/knx-bridge-01/command/light-living-main
func (Topics) BridgeCommand(bridgeID, deviceID string) string {
	return fmt.Sprintf("%s/%s/command/%s", TopicPrefixBridge, bridgeID, deviceID)
}

// BridgeResponse returns the topic for command responses from a bridge.
//
// Example: graylogic/bridge/knx-bridge-01/response/req-abc123
func (Topics) BridgeResponse(bridgeID, requestID string) string {
	return fmt.Sprintf("%s/%s/response/%s", TopicPrefixBridge, bridgeID, requestID)
}

// BridgeHealth returns the topic for bridge health status.
//
// Example: graylogic/bridge/knx-bridge-01/health
func (Topics) BridgeHealth(bridgeID string) string {
	return fmt.Sprintf("%s/%s/health", TopicPrefixBridge, bridgeID)
}

// BridgeDiscovery returns the topic for device discovery from a bridge.
//
// Example: graylogic/bridge/knx-bridge-01/discovery
func (Topics) BridgeDiscovery(bridgeID string) string {
	return fmt.Sprintf("%s/%s/discovery", TopicPrefixBridge, bridgeID)
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
// Pattern: graylogic/bridge/+/state/+
func (Topics) AllBridgeStates() string {
	return fmt.Sprintf("%s/+/state/+", TopicPrefixBridge)
}

// AllBridgeCommands returns a pattern matching all commands to bridges.
//
// Pattern: graylogic/bridge/+/command/+
func (Topics) AllBridgeCommands() string {
	return fmt.Sprintf("%s/+/command/+", TopicPrefixBridge)
}

// AllBridgeHealth returns a pattern matching all bridge health updates.
//
// Pattern: graylogic/bridge/+/health
func (Topics) AllBridgeHealth() string {
	return fmt.Sprintf("%s/+/health", TopicPrefixBridge)
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
