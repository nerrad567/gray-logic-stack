# IMP-003: Subscription Tracking for Reconnect

## Status
Accepted

## Date
2026-01-19

## Context

Gray Logic Core connects to MQTT broker with `CleanSession=true` (default). When connection is lost and restored:
- Broker forgets all subscriptions
- Core stops receiving bridge state updates
- Automations fail silently

This is critical because:
- Network interruptions are expected
- Power outages may restart broker
- Core must resume full operation automatically

## Decision

Track all subscriptions in a `map[string]subscription` and restore them automatically on reconnect.

## Implementation

**Tracking structure** ([client.go](file:///home/darren/Development/Projects/gray-logic-stack/code/core/internal/infrastructure/mqtt/client.go#L26-L28)):
```go
type Client struct {
    // ...
    subscriptions map[string]subscription
    subMu         sync.RWMutex
    // ...
}

type subscription struct {
    topic   string
    qos     byte
    handler MessageHandler
}
```

**Subscribe tracks** ([subscribe.go](file:///home/darren/Development/Projects/gray-logic-stack/code/core/internal/infrastructure/mqtt/subscribe.go)):
```go
func (c *Client) Subscribe(topic string, qos byte, handler MessageHandler) error {
    // ... perform subscription ...
    
    // Track for reconnect restoration
    c.subMu.Lock()
    c.subscriptions[topic] = subscription{topic, qos, handler}
    c.subMu.Unlock()
    
    return nil
}
```

**Restore on connect** ([client.go](file:///home/darren/Development/Projects/gray-logic-stack/code/core/internal/infrastructure/mqtt/client.go#L148-L157)):
```go
func (c *Client) restoreSubscriptions() {
    c.subMu.RLock()
    defer c.subMu.RUnlock()

    for _, sub := range c.subscriptions {
        c.client.Subscribe(sub.topic, sub.qos, c.wrapHandler(sub.handler))
    }
}
```

**Triggered by callback:**
```go
opts.SetOnConnectHandler(func(_ pahomqtt.Client) {
    c.handleConnect()  // Calls restoreSubscriptions()
})
```

## Consequences

### Advantages
- **Automatic recovery** — Core resumes normal operation after any disconnect
- **Invisible to callers** — Subscribe once, works forever
- **Handler preservation** — Same handler functions used after reconnect

### Disadvantages
- **Memory overhead** — Subscriptions stored twice (paho + our map)
- **Unsubscribe must track** — Must remove from map on Unsubscribe()

### Risks
- **Handler drift** — If handler is changed without Unsubscribe/Subscribe, old handler restored
  - Mitigation: Document that handlers are immutable per topic
- **Race condition** — Rapid Subscribe/Unsubscribe during reconnect
  - Mitigation: Mutex protection

## Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|----------------|
| CleanSession=false | Broker stores subscriptions, but client ID conflict issues |
| Don't auto-restore | Manual intervention required after every disconnect |
| Persistent queue | Overkill, adds complexity for simple reconnect case |
