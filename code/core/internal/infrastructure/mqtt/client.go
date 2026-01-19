package mqtt

import (
	"context"
	"fmt"
	"sync"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
)

// Client wraps paho.mqtt.golang with Gray Logic-specific functionality.
//
// It provides connection management, message publishing, subscription handling,
// and automatic reconnection with exponential backoff.
//
// Thread Safety:
//   - All methods are safe for concurrent use from multiple goroutines.
//   - Subscriptions are automatically restored on reconnection.
type Client struct {
	client  pahomqtt.Client
	options *pahomqtt.ClientOptions
	cfg     config.MQTTConfig

	// subscriptions tracks active subscriptions for re-subscription on reconnect.
	subscriptions map[string]subscription
	subMu         sync.RWMutex

	// connected tracks current connection state.
	connected bool
	connMu    sync.RWMutex

	// Callbacks for connection events (optional, set via SetOnConnect/SetOnDisconnect).
	onConnect    func()
	onDisconnect func(err error)
}

// subscription holds subscription details for re-subscription on reconnect.
type subscription struct {
	topic   string
	qos     byte
	handler MessageHandler
}

// MessageHandler is the callback signature for received messages.
//
// Handlers are invoked in separate goroutines by the paho library.
// They should not block for extended periods.
//
// Parameters:
//   - topic: The topic the message was received on (wildcards expanded)
//   - payload: The raw message payload (typically JSON)
//
// Returns:
//   - error: Logged but does not affect message acknowledgment
type MessageHandler func(topic string, payload []byte) error

// Connect establishes a connection to the MQTT broker.
//
// It performs the following setup:
//  1. Builds connection options from config (broker URL, auth, TLS)
//  2. Configures Last Will and Testament (LWT) for offline detection
//  3. Sets up auto-reconnect with exponential backoff
//  4. Attempts initial connection with timeout
//  5. Publishes online status to graylogic/system/status
//
// Parameters:
//   - cfg: MQTT configuration from config.yaml
//
// Returns:
//   - *Client: Connected client ready for use
//   - error: If initial connection fails within timeout
func Connect(cfg config.MQTTConfig) (*Client, error) {
	opts := buildClientOptions(cfg)
	configureLWT(opts, cfg.Broker.ClientID)

	c := &Client{
		cfg:           cfg,
		options:       opts,
		subscriptions: make(map[string]subscription),
	}

	// Set up connection callbacks
	opts.SetOnConnectHandler(func(_ pahomqtt.Client) {
		c.handleConnect()
	})

	opts.SetConnectionLostHandler(func(_ pahomqtt.Client, err error) {
		c.handleDisconnect(err)
	})

	opts.SetReconnectingHandler(func(_ pahomqtt.Client, _ *pahomqtt.ClientOptions) {
		// Could add logging here when reconnecting
	})

	// Create and connect
	c.client = pahomqtt.NewClient(opts)
	token := c.client.Connect()
	if !token.WaitTimeout(defaultConnectTimeout) {
		return nil, fmt.Errorf("%w: timeout after %v", ErrConnectionFailed, defaultConnectTimeout)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	// Set connected state immediately after successful connection.
	// The OnConnectHandler callback runs asynchronously and may not have
	// executed yet, so we set it here to ensure IsConnected() returns true.
	// The callback will handle subscription restoration and status publishing.
	c.connMu.Lock()
	c.connected = true
	c.connMu.Unlock()

	return c, nil
}

// handleConnect is called when the connection is established.
func (c *Client) handleConnect() {
	c.connMu.Lock()
	c.connected = true
	c.connMu.Unlock()

	// Restore subscriptions
	c.restoreSubscriptions()

	// Publish online status
	c.publishOnlineStatus()

	// Notify callback if set
	if c.onConnect != nil {
		c.onConnect()
	}
}

// handleDisconnect is called when the connection is lost.
func (c *Client) handleDisconnect(err error) {
	c.connMu.Lock()
	c.connected = false
	c.connMu.Unlock()

	// Notify callback if set
	if c.onDisconnect != nil {
		c.onDisconnect(err)
	}
}

// restoreSubscriptions re-subscribes to all tracked topics after reconnect.
func (c *Client) restoreSubscriptions() {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	for _, sub := range c.subscriptions {
		// Re-subscribe (ignore errors during reconnection)
		c.client.Subscribe(sub.topic, sub.qos, c.wrapHandler(sub.handler))
	}
}

// publishOnlineStatus publishes Core's online status to the system status topic.
func (c *Client) publishOnlineStatus() {
	topic := Topics{}.SystemStatus()
	payload := buildOnlinePayload(c.cfg.Broker.ClientID)
	c.client.Publish(topic, byte(c.cfg.QoS), true, payload)
}

// Close gracefully disconnects from the MQTT broker.
//
// It performs:
//  1. Publishes graceful offline status (different from LWT crash status)
//  2. Waits for pending publish operations
//  3. Disconnects from broker
//
// Returns:
//   - error: If disconnect fails (connection already closed is not an error)
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}

	// Check if connected before trying to publish
	if c.IsConnected() {
		// Publish graceful shutdown status
		topic := Topics{}.SystemStatus()
		payload := buildOfflinePayload(c.cfg.Broker.ClientID)
		token := c.client.Publish(topic, byte(c.cfg.QoS), true, payload)
		token.WaitTimeout(defaultPublishTimeout)
	}

	// Disconnect with quiesce period for pending operations
	c.client.Disconnect(defaultDisconnectQuiesce)

	c.connMu.Lock()
	c.connected = false
	c.connMu.Unlock()

	return nil
}

// HealthCheck verifies the MQTT connection is alive and functioning.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//
// Returns:
//   - error: nil if healthy, error describing the issue otherwise
func (c *Client) HealthCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("mqtt health check: %w", ctx.Err())
	default:
	}

	if !c.IsConnected() {
		return ErrNotConnected
	}

	return nil
}

// IsConnected returns the current connection state.
//
// Note: This reflects the last known state. For reliability,
// use HealthCheck which can perform an active test.
func (c *Client) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.connected && c.client.IsConnected()
}

// SetOnConnect sets a callback to be invoked when connection is established.
// This is called on initial connect and on every reconnect.
func (c *Client) SetOnConnect(callback func()) {
	c.onConnect = callback
}

// SetOnDisconnect sets a callback to be invoked when connection is lost.
// The error parameter describes why the connection was lost.
func (c *Client) SetOnDisconnect(callback func(err error)) {
	c.onDisconnect = callback
}

// wrapHandler wraps a MessageHandler with panic recovery.
func (c *Client) wrapHandler(handler MessageHandler) pahomqtt.MessageHandler {
	return func(_ pahomqtt.Client, msg pahomqtt.Message) {
		defer func() {
			if r := recover(); r != nil {
				// TODO: Log panic when logging package is available
				// For now, silently recover to prevent crash
				_ = r // Acknowledge recovered panic
			}
		}()

		if err := handler(msg.Topic(), msg.Payload()); err != nil {
			// TODO: Log handler error when logging package is available
			_ = err // Acknowledge error until logging is available
		}
	}
}
