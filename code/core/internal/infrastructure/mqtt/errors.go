package mqtt

import "errors"

// Domain-specific errors for MQTT operations.
// Use errors.Is() to check for these errors in calling code.
var (
	// ErrNotConnected is returned when attempting operations on a disconnected client.
	ErrNotConnected = errors.New("mqtt: client not connected")

	// ErrConnectionFailed is returned when the initial connection attempt fails.
	ErrConnectionFailed = errors.New("mqtt: connection failed")

	// ErrPublishFailed is returned when a publish operation fails.
	ErrPublishFailed = errors.New("mqtt: publish failed")

	// ErrSubscribeFailed is returned when a subscribe operation fails.
	ErrSubscribeFailed = errors.New("mqtt: subscribe failed")

	// ErrUnsubscribeFailed is returned when an unsubscribe operation fails.
	ErrUnsubscribeFailed = errors.New("mqtt: unsubscribe failed")

	// ErrInvalidQoS is returned when an invalid QoS level is specified.
	// Valid QoS levels are 0, 1, or 2.
	ErrInvalidQoS = errors.New("mqtt: invalid QoS level (must be 0, 1, or 2)")

	// ErrInvalidTopic is returned when an empty or invalid topic is provided.
	ErrInvalidTopic = errors.New("mqtt: topic cannot be empty")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("mqtt: operation timed out")
)
