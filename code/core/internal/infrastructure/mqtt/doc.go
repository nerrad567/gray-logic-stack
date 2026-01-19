// Package mqtt provides MQTT client connectivity for Gray Logic Core.
//
// This package manages:
//   - Connection to Mosquitto broker with auto-reconnect
//   - Message publishing with QoS guarantees
//   - Topic subscriptions with wildcard support
//   - Last Will and Testament (LWT) for offline detection
//   - Connection health monitoring
//
// # Architecture
//
// Gray Logic uses MQTT as the internal message bus connecting the Core
// to protocol bridges (KNX, DALI, Modbus, etc.). The broker (Mosquitto)
// decouples Core from protocol-specific implementations.
//
//	Gray Logic Core ↔ MQTT Broker ↔ Protocol Bridges
//
// # Security Considerations
//
//   - TLS is required for production deployments (cfg.Broker.TLS=true)
//   - Credentials are validated against broker ACL
//   - Anonymous access is only for local development
//   - Message payloads are not encrypted beyond TLS transport
//
// # Performance Characteristics
//
//   - Connection: <1 second to local broker
//   - Publish latency: <10ms for QoS 1 to local broker
//   - Reconnect: Exponential backoff 1s-60s with jitter
//   - Message throughput: Broker-limited (typically 10K+ msg/sec)
//
// # Usage
//
//	client, err := mqtt.Connect(cfg.MQTT)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Subscribe to all bridge state updates
//	err = client.Subscribe(mqtt.Topics{}.AllBridgeStates(), 1,
//	    func(topic string, payload []byte) error {
//	        log.Printf("Received: %s = %s", topic, payload)
//	        return nil
//	    })
//
//	// Publish command
//	topic := mqtt.Topics{}.BridgeCommand("knx-bridge-01", "light-living")
//	client.Publish(topic, []byte(`{"on":true}`), 1, false)
//
// # Related Documents
//
//   - docs/protocols/mqtt.md — Topic structure and message formats
//   - docs/architecture/mqtt-resilience.md — Persistence and recovery
//   - docs/architecture/decisions/002-mqtt-internal-bus.md — Why MQTT
package mqtt
