package mqtt

import (
	"fmt"
)

// Subscribe registers a handler for messages on the specified topic.
//
// Topics can include MQTT wildcards:
//   - + (single-level): "graylogic/bridge/+/state/+" matches any bridge and device
//   - # (multi-level): "graylogic/#" matches all Gray Logic topics
//
// The handler is called in a separate goroutine for each received message.
// Handlers should not block for extended periods as this may affect message
// processing throughput.
//
// Subscriptions are automatically restored if the connection is lost and
// reconnected (tracked internally).
//
// Parameters:
//   - topic: The topic pattern to subscribe to
//   - qos: Maximum QoS level for received messages (0, 1, or 2)
//   - handler: Callback function invoked for each message
//
// Returns:
//   - error: nil on success, or wrapped error describing the failure
//
// Example:
//
//	err := client.Subscribe(mqtt.Topics{}.AllBridgeStates(), 1,
//	    func(topic string, payload []byte) error {
//	        log.Printf("Received: %s = %s", topic, payload)
//	        return nil
//	    })
func (c *Client) Subscribe(topic string, qos byte, handler MessageHandler) error {
	// Validate inputs
	if topic == "" {
		return ErrInvalidTopic
	}
	if qos > maxQoS {
		return ErrInvalidQoS
	}
	if handler == nil {
		return fmt.Errorf("%w: handler cannot be nil", ErrSubscribeFailed)
	}

	// Check connection state
	if !c.IsConnected() {
		return ErrNotConnected
	}

	// Track subscription for reconnection restoration
	c.subMu.Lock()
	c.subscriptions[topic] = subscription{
		topic:   topic,
		qos:     qos,
		handler: handler,
	}
	c.subMu.Unlock()

	// Subscribe with wrapped handler (includes panic recovery)
	token := c.client.Subscribe(topic, qos, c.wrapHandler(handler))
	if !token.WaitTimeout(defaultPublishTimeout) {
		// Remove from tracking since subscription failed
		c.subMu.Lock()
		delete(c.subscriptions, topic)
		c.subMu.Unlock()
		return fmt.Errorf("%w: timeout after %v", ErrSubscribeFailed, defaultPublishTimeout)
	}
	if err := token.Error(); err != nil {
		// Remove from tracking since subscription failed
		c.subMu.Lock()
		delete(c.subscriptions, topic)
		c.subMu.Unlock()
		return fmt.Errorf("%w: %w", ErrSubscribeFailed, err)
	}

	return nil
}

// Unsubscribe removes a subscription and stops receiving messages for a topic.
//
// After unsubscribing, the handler will no longer be called for new messages
// on this topic. Any messages in flight may still be delivered.
//
// Parameters:
//   - topic: The exact topic pattern that was subscribed to
//
// Returns:
//   - error: nil on success, or wrapped error describing the failure
func (c *Client) Unsubscribe(topic string) error {
	// Validate inputs
	if topic == "" {
		return ErrInvalidTopic
	}

	// Check connection state
	if !c.IsConnected() {
		return ErrNotConnected
	}

	// Remove from tracking
	c.subMu.Lock()
	delete(c.subscriptions, topic)
	c.subMu.Unlock()

	// Unsubscribe from broker
	token := c.client.Unsubscribe(topic)
	if !token.WaitTimeout(defaultPublishTimeout) {
		return fmt.Errorf("%w: timeout after %v", ErrUnsubscribeFailed, defaultPublishTimeout)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("%w: %w", ErrUnsubscribeFailed, err)
	}

	return nil
}

// SubscriptionCount returns the number of active subscriptions.
//
// This can be useful for monitoring and debugging.
func (c *Client) SubscriptionCount() int {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return len(c.subscriptions)
}

// HasSubscription checks if a subscription exists for the given topic.
//
// Note: This checks only the exact topic string, not pattern matching.
func (c *Client) HasSubscription(topic string) bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	_, exists := c.subscriptions[topic]
	return exists
}
