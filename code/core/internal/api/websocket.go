package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/nerrad567/gray-logic-core/internal/auth"
	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
)

// WebSocket constants.
const (
	WSTypeSubscribe   = "subscribe"
	WSTypeUnsubscribe = "unsubscribe"
	WSTypePing        = "ping"
	WSTypePong        = "pong"
	WSTypeEvent       = "event"
	WSTypeResponse    = "response"
	WSTypeError       = "error"

	// wsSendBufferSize is the per-client outbound message buffer size.
	wsSendBufferSize = 256
)

// WSMessage represents a message sent to/from a WebSocket client.
type WSMessage struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	EventType string `json:"event_type,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

// WSSubscribePayload is the payload for subscribe/unsubscribe messages.
type WSSubscribePayload struct {
	Channels []string `json:"channels"`
}

// Hub manages WebSocket connections and broadcasts events.
type Hub struct {
	cfg     config.WebSocketConfig
	logger  *logging.Logger
	clients map[*WSClient]struct{}
	mu      sync.RWMutex
}

// WSClient represents a connected WebSocket client.
type WSClient struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]struct{}
	mu            sync.RWMutex
	// Identity fields propagated from the WebSocket ticket.
	userID  string    // non-empty for user connections
	role    auth.Role // role of the authenticated caller
	panelID string    // non-empty for panel connections
}

// upgrader configures the WebSocket upgrader.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		// Origin checking is handled by CORS middleware
		return true
	},
}

// NewHub creates a new WebSocket hub.
func NewHub(cfg config.WebSocketConfig, logger *logging.Logger) *Hub {
	return &Hub{
		cfg:     cfg,
		logger:  logger,
		clients: make(map[*WSClient]struct{}),
	}
}

// Run starts the hub's main loop. It blocks until the context is cancelled.
func (h *Hub) Run(ctx context.Context) {
	<-ctx.Done()
	h.closeAll()
}

// Register adds a client to the hub.
func (h *Hub) Register(client *WSClient) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
	h.logger.Debug("websocket client connected", "clients", h.ClientCount())
}

// Unregister removes a client from the hub.
// Only the goroutine that successfully removes the client from the map
// closes the send channel, preventing double-close panics during shutdown.
func (h *Hub) Unregister(client *WSClient) {
	h.mu.Lock()
	_, existed := h.clients[client]
	delete(h.clients, client)
	h.mu.Unlock()

	if existed {
		close(client.send)
	}
	h.logger.Debug("websocket client disconnected", "clients", h.ClientCount())
}

// Broadcast sends an event to all clients subscribed to the given channel.
// Lock ordering: hub lock is acquired first, then released before per-client
// subscription checks. This avoids holding both hub and client locks simultaneously.
func (h *Hub) Broadcast(channel string, payload any) {
	msg := WSMessage{
		Type:      WSTypeEvent,
		EventType: channel,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal broadcast message", "error", err)
		return
	}

	// Snapshot client list under hub lock, then release before sending
	h.mu.RLock()
	clients := make([]*WSClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	sentCount := 0
	for _, client := range clients {
		if client.isSubscribed(channel) {
			client.trySend(data)
			sentCount++
		}
	}
	if sentCount > 0 {
		h.logger.Debug("broadcast sent", "channel", channel, "recipients", sentCount)
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// closeAll disconnects all clients and closes their send channels
// so writePump goroutines can exit cleanly.
func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
		if client.conn != nil {
			client.conn.Close()
		}
		delete(h.clients, client)
	}
}

// subscribeStateUpdates subscribes to MQTT device state topics and broadcasts
// changes to WebSocket clients subscribed to "device.state_changed".
func (s *Server) subscribeStateUpdates() error { //nolint:gocognit // MQTT subscription: message parsing + registry + TSDB pipeline
	if s.mqtt == nil {
		return nil // MQTT not configured; WebSocket broadcast disabled
	}
	// Subscribe to all bridge state publications: graylogic/state/{protocol}/{ga}
	topic := "graylogic/state/+/+"
	s.logger.Info("subscribing to state updates for WebSocket relay", "topic", topic)
	return s.mqtt.Subscribe(topic, 1, func(t string, payload []byte) error {
		if s.hub == nil {
			return nil // Hub not yet initialised
		}

		// Parse the state message and broadcast to WebSocket clients
		var stateMsg map[string]any
		if err := json.Unmarshal(payload, &stateMsg); err != nil {
			s.logger.Warn("failed to parse state message for WebSocket broadcast", "error", err)
			return nil
		}

		s.logger.Debug("broadcasting state to WebSocket", "topic", t, "device_id", stateMsg["device_id"])
		s.hub.Broadcast("device.state_changed", stateMsg)

		// Update device registry with state from bus.
		// Write-through commands are already handled by the bridge; this covers
		// incoming bus telegrams (physical switch presses, sensor updates).
		deviceID, _ := stateMsg["device_id"].(string)     //nolint:errcheck // type assertion checked via empty string test below
		stateMap, _ := stateMsg["state"].(map[string]any) //nolint:errcheck // type assertion checked via nil test below

		if deviceID != "" && stateMap != nil {
			devState := make(device.State, len(stateMap))
			for k, v := range stateMap {
				devState[k] = v
			}
			if err := s.registry.SetDeviceState(context.Background(), deviceID, devState); err != nil {
				s.logger.Debug("state update to registry failed", "device_id", deviceID, "error", err)
			}

			// Write numeric state fields to time-series DB for telemetry
			if s.tsdb != nil {
				for field, val := range stateMap {
					switch v := val.(type) {
					case float64:
						s.tsdb.WriteDeviceMetric(deviceID, field, v)
					case bool:
						boolVal := 0.0
						if v {
							boolVal = 1.0
						}
						s.tsdb.WriteDeviceMetric(deviceID, field, boolVal)
					}
				}
			}

			// Record state change to local audit trail for history queries.
			if s.stateHistory != nil {
				if err := s.stateHistory.RecordStateChange(context.Background(), deviceID, devState, device.StateHistorySourceMQTT); err != nil {
					s.logger.Debug("state history write failed", "device_id", deviceID, "error", err)
				}
			}
		}

		return nil
	})
}

// handleWebSocket upgrades the HTTP connection to a WebSocket connection.
// Authentication is via ticket query parameter (obtained from POST /auth/ws-ticket).
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Validate ticket-based auth — ticket is required
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		writeUnauthorized(w, "ticket query parameter is required")
		return
	}
	entry, ok := s.validateTicket(ticket)
	if !ok {
		writeUnauthorized(w, "invalid or expired ticket")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	client := &WSClient{
		hub:           s.hub,
		conn:          conn,
		send:          make(chan []byte, wsSendBufferSize),
		subscriptions: make(map[string]struct{}),
		userID:        entry.userID,
		role:          entry.role,
		panelID:       entry.panelID,
	}

	s.hub.Register(client)

	// Start read/write pumps
	go client.writePump(s.wsCfg)
	go client.readPump(s.wsCfg)
}

// readPump reads messages from the WebSocket connection.
func (c *WSClient) readPump(cfg config.WebSocketConfig) {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(int64(cfg.MaxMessageSize))
	pingInterval := time.Duration(cfg.PingInterval) * time.Second
	pongWait := time.Duration(cfg.PongTimeout) * time.Second
	//nolint:errcheck // Best-effort deadline on connection setup
	c.conn.SetReadDeadline(time.Now().Add(pingInterval + pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pingInterval + pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.hub.logger.Warn("websocket read error", "error", err)
			} else {
				c.hub.logger.Debug("websocket closed", "error", err)
			}
			return
		}
		// Any client message resets the read deadline (keeps connection alive
		// even if browser doesn't respond to protocol-level pings).
		//nolint:errcheck // Best-effort deadline reset
		c.conn.SetReadDeadline(time.Now().Add(pingInterval + pongWait))
		c.handleMessage(message)
	}
}

// writePump writes messages to the WebSocket connection.
func (c *WSClient) writePump(cfg config.WebSocketConfig) {
	pingInterval := time.Duration(cfg.PingInterval) * time.Second
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	pongWait := time.Duration(cfg.PongTimeout) * time.Second

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Hub closed the channel
				//nolint:errcheck // Best-effort close message
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			//nolint:errcheck // Best-effort deadline; write error caught below
			c.conn.SetWriteDeadline(time.Now().Add(pongWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			//nolint:errcheck // Best-effort deadline; ping error caught below
			c.conn.SetWriteDeadline(time.Now().Add(pongWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes an incoming WebSocket message.
func (c *WSClient) handleMessage(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("", "invalid JSON message")
		return
	}

	switch msg.Type {
	case WSTypeSubscribe:
		c.handleSubscribe(msg)
	case WSTypeUnsubscribe:
		c.handleUnsubscribe(msg)
	case WSTypePing:
		c.sendResponse(msg.ID, WSTypePong, nil)
	default:
		c.sendError(msg.ID, "unknown message type: "+msg.Type)
	}
}

// handleSubscribe adds channels to the client's subscription list.
func (c *WSClient) handleSubscribe(msg WSMessage) {
	// Parse payload to get channels
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		c.sendError(msg.ID, "invalid payload")
		return
	}

	var sub WSSubscribePayload
	if err := json.Unmarshal(payloadBytes, &sub); err != nil {
		c.sendError(msg.ID, "invalid subscribe payload")
		return
	}

	c.mu.Lock()
	for _, ch := range sub.Channels {
		c.subscriptions[ch] = struct{}{}
	}
	c.mu.Unlock()

	c.hub.logger.Info("websocket client subscribed", "channels", sub.Channels)

	c.sendResponse(msg.ID, WSTypeResponse, map[string]any{
		"subscribed": sub.Channels,
	})
}

// handleUnsubscribe removes channels from the client's subscription list.
func (c *WSClient) handleUnsubscribe(msg WSMessage) {
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		c.sendError(msg.ID, "invalid payload")
		return
	}

	var sub WSSubscribePayload
	if err := json.Unmarshal(payloadBytes, &sub); err != nil {
		c.sendError(msg.ID, "invalid unsubscribe payload")
		return
	}

	c.mu.Lock()
	for _, ch := range sub.Channels {
		delete(c.subscriptions, ch)
	}
	c.mu.Unlock()

	c.sendResponse(msg.ID, WSTypeResponse, map[string]any{
		"unsubscribed": sub.Channels,
	})
}

// trySend attempts to send data to the client's send channel.
// It silently handles closed channels (client disconnected during broadcast)
// and full buffers (slow client).
func (c *WSClient) trySend(data []byte) {
	defer func() {
		recover() //nolint:errcheck // Absorb send-on-closed-channel panic
	}()

	select {
	case c.send <- data:
	default:
		// Client buffer full, skip
	}
}

// isSubscribed checks if the client is subscribed to a channel.
func (c *WSClient) isSubscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.subscriptions[channel]
	return ok
}

// sendResponse sends a response message to the client.
// Routes through trySend to safely handle closed channels during shutdown.
func (c *WSClient) sendResponse(id, msgType string, payload any) {
	msg := WSMessage{
		Type:      msgType,
		ID:        id,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.trySend(data)
}

// sendError sends an error message to the client.
func (c *WSClient) sendError(id, message string) {
	c.sendResponse(id, WSTypeError, map[string]string{"message": message})
}
