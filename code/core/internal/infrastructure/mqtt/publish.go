package mqtt

import (
	"fmt"
)

// Maximum payload size for MQTT messages (1MB).
// This prevents resource exhaustion and aligns with typical broker limits.
const maxPayloadSize = 1 << 20 // 1MB

// Publish sends a message to the specified MQTT topic.
//
// Parameters:
//   - topic: The topic to publish to (e.g., "graylogic/command/knx/light-living")
//   - payload: The message payload (typically JSON, max 1MB)
//   - qos: Quality of Service level (0, 1, or 2)
//   - retained: Whether the broker should retain the message for new subscribers
//
// QoS Levels:
//   - 0: At most once (fire and forget)
//   - 1: At least once (guaranteed delivery, may duplicate)
//   - 2: Exactly once (guaranteed, no duplicates, higher overhead)
//
// Retained Messages:
//   - When true, broker stores the last message for each topic
//   - New subscribers immediately receive the retained message
//   - Use for state topics (device status, system status)
//   - Don't use for commands or events
//
// Returns:
//   - error: nil on success, or wrapped error describing the failure
//
// Example:
//
//	topic := mqtt.Topics{}.BridgeCommand("knx", "light-living")
//	err := client.Publish(topic, []byte(`{"on":true}`), 1, false)
func (c *Client) Publish(topic string, payload []byte, qos byte, retained bool) error {
	// Validate inputs
	if topic == "" {
		return ErrInvalidTopic
	}
	if qos > maxQoS {
		return ErrInvalidQoS
	}
	if len(payload) > maxPayloadSize {
		return fmt.Errorf("%w: payload size %d exceeds maximum %d bytes", ErrPublishFailed, len(payload), maxPayloadSize)
	}

	// Check connection state
	if !c.IsConnected() {
		return ErrNotConnected
	}

	// Publish with timeout
	token := c.client.Publish(topic, qos, retained, payload)
	if !token.WaitTimeout(defaultPublishTimeout) {
		return fmt.Errorf("%w: timeout after %v", ErrPublishFailed, defaultPublishTimeout)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("%w: %w", ErrPublishFailed, err)
	}

	return nil
}

// PublishString is a convenience method that publishes a string payload.
//
// This is equivalent to calling Publish with []byte(payload).
func (c *Client) PublishString(topic string, payload string, qos byte, retained bool) error {
	return c.Publish(topic, []byte(payload), qos, retained)
}

// PublishRetained publishes a retained message with the configured default QoS.
//
// Use for state updates where new subscribers should receive the current state.
func (c *Client) PublishRetained(topic string, payload []byte) error {
	return c.Publish(topic, payload, byte(c.cfg.QoS), true)
}
