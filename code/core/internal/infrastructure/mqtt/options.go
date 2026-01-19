package mqtt

import (
	"crypto/tls"
	"fmt"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
)

// Connection constants.
const (
	// defaultConnectTimeout is the maximum time to wait for initial connection.
	defaultConnectTimeout = 10 * time.Second

	// defaultPublishTimeout is the maximum time to wait for publish acknowledgment.
	defaultPublishTimeout = 5 * time.Second

	// defaultDisconnectQuiesce is the time to wait for pending operations on disconnect.
	defaultDisconnectQuiesce = 1000 // milliseconds

	// defaultKeepAlive is the keepalive interval for the connection.
	defaultKeepAlive = 60 * time.Second

	// maxQoS is the maximum QoS level supported.
	maxQoS = 2

	// tlsMinVersion is the minimum TLS version for secure connections.
	tlsMinVersion = tls.VersionTLS12
)

// buildClientOptions creates paho MQTT options from Gray Logic config.
//
// This configures:
//   - Broker URL (tcp:// or ssl:// based on TLS setting)
//   - Client ID for identification
//   - Authentication credentials (if provided)
//   - Auto-reconnect with exponential backoff
//   - TLS configuration (if enabled)
//   - Clean session mode
func buildClientOptions(cfg config.MQTTConfig) *pahomqtt.ClientOptions {
	opts := pahomqtt.NewClientOptions()

	// Broker URL
	scheme := "tcp"
	if cfg.Broker.TLS {
		scheme = "ssl"
	}
	brokerURL := fmt.Sprintf("%s://%s:%d", scheme, cfg.Broker.Host, cfg.Broker.Port)
	opts.AddBroker(brokerURL)

	// Client identification
	opts.SetClientID(cfg.Broker.ClientID)

	// Authentication (if credentials provided)
	if cfg.Auth.Username != "" {
		opts.SetUsername(cfg.Auth.Username)
		opts.SetPassword(cfg.Auth.Password)
	}

	// Clean session - start fresh on connect (no persistent session on broker)
	opts.SetCleanSession(true)

	// Auto-reconnect with exponential backoff
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(time.Duration(cfg.Reconnect.InitialDelay) * time.Second)
	opts.SetMaxReconnectInterval(time.Duration(cfg.Reconnect.MaxDelay) * time.Second)

	// Connection timeout
	opts.SetConnectTimeout(defaultConnectTimeout)

	// Keepalive - broker sends PINGs to detect dead connections
	opts.SetKeepAlive(defaultKeepAlive)

	// TLS configuration if enabled
	if cfg.Broker.TLS {
		tlsConfig := &tls.Config{
			MinVersion: tlsMinVersion,
		}
		opts.SetTLSConfig(tlsConfig)
	}

	return opts
}

// configureLWT sets up Last Will and Testament for offline detection.
//
// The LWT message is published by the broker if the client disconnects
// unexpectedly (crash, network failure, etc.). This allows other services
// to detect when Core goes offline.
//
// Topic: graylogic/system/status
// QoS: 1 (guaranteed delivery)
// Retained: true (new subscribers see last status)
func configureLWT(opts *pahomqtt.ClientOptions, clientID string) {
	willTopic := Topics{}.SystemStatus()
	willPayload := fmt.Sprintf(
		`{"status":"offline","client_id":"%s","reason":"unexpected_disconnect","timestamp":"%s"}`,
		clientID,
		time.Now().UTC().Format(time.RFC3339),
	)

	opts.SetWill(willTopic, willPayload, 1, true)
}

// buildOnlinePayload creates the JSON payload for online status messages.
func buildOnlinePayload(clientID string) string {
	return fmt.Sprintf(
		`{"status":"online","client_id":"%s","timestamp":"%s"}`,
		clientID,
		time.Now().UTC().Format(time.RFC3339),
	)
}

// buildOfflinePayload creates the JSON payload for graceful offline status.
func buildOfflinePayload(clientID string) string {
	return fmt.Sprintf(
		`{"status":"offline","client_id":"%s","reason":"graceful_shutdown","timestamp":"%s"}`,
		clientID,
		time.Now().UTC().Format(time.RFC3339),
	)
}
