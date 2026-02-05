package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// DeviceInfo holds the minimal device information the engine needs for routing.
// GatewayID identifies the physical gateway for the bridge's internal routing
// but is not used in MQTT topic construction (topics use protocol as the key).
type DeviceInfo struct {
	ID        string
	Protocol  string
	GatewayID *string
}

// DeviceRegistry is the interface the engine needs from the device package.
// It provides device information for MQTT command routing.
type DeviceRegistry interface {
	// GetDevice retrieves device info for routing commands.
	GetDevice(ctx context.Context, id string) (DeviceInfo, error)
}

// MQTTClient is the interface for publishing commands to protocol bridges.
type MQTTClient interface {
	// Publish sends a message to the specified MQTT topic.
	Publish(topic string, payload []byte, qos byte, retained bool) error
}

// WSHub is the interface for broadcasting WebSocket events.
type WSHub interface {
	// Broadcast sends an event to all clients subscribed to the given channel.
	Broadcast(channel string, payload any)
}

// Engine orchestrates scene execution.
//
// It loads scenes from the registry, groups actions by parallel flag,
// executes groups sequentially (with parallel actions within each group),
// publishes MQTT commands to protocol bridges, and logs execution results.
//
// Thread Safety: ActivateScene is safe for concurrent use.
type Engine struct {
	registry *Registry
	devices  DeviceRegistry
	mqtt     MQTTClient
	hub      WSHub
	repo     Repository // For execution logging
	logger   Logger
}

// NewEngine creates a new scene engine.
//
// Parameters:
//   - registry: Scene registry for loading scene definitions
//   - devices: Device registry for routing (protocol/gateway lookup)
//   - mqtt: MQTT client for publishing commands to bridges
//   - hub: WebSocket hub for broadcasting activation events (may be nil)
//   - repo: Repository for persisting execution logs
//   - logger: Logger instance
func NewEngine(registry *Registry, devices DeviceRegistry, mqtt MQTTClient, hub WSHub, repo Repository, logger Logger) *Engine {
	if logger == nil {
		logger = noopLogger{}
	}
	return &Engine{
		registry: registry,
		devices:  devices,
		mqtt:     mqtt,
		hub:      hub,
		repo:     repo,
		logger:   logger,
	}
}

// ActivateScene activates a scene by ID.
//
// It loads the scene, verifies it's enabled, groups actions by parallel flag,
// executes each group (parallel actions via goroutines), and logs the result.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - sceneID: The scene to activate
//   - triggerType: How the scene was triggered (manual, schedule, event, voice, automation)
//   - triggerSource: Where the trigger originated (api, wall_panel, mobile, etc.)
//
// Returns:
//   - string: The execution ID for tracking
//   - error: nil on success, or:
//   - ErrSceneNotFound if scene doesn't exist
//   - ErrSceneDisabled if scene is disabled
//   - ErrMQTTUnavailable if MQTT client is nil
//
// maxSceneExecutionTime is the hard limit for a single scene activation.
// Even complex scenes (10+ devices, sequential groups with delays) should complete
// well within this window. Prevents goroutine accumulation from runaway scenes.
const maxSceneExecutionTime = 60 * time.Second

func (e *Engine) ActivateScene(ctx context.Context, sceneID, triggerType, triggerSource string) (string, error) { //nolint:gocognit,gocyclo // scene activation: validates, executes actions, records execution
	// Apply execution timeout to prevent unbounded goroutine accumulation.
	ctx, cancel := context.WithTimeout(ctx, maxSceneExecutionTime)
	defer cancel()

	// Load scene from registry
	scene, err := e.registry.GetScene(ctx, sceneID)
	if err != nil {
		return "", err
	}

	// Check enabled
	if !scene.Enabled {
		return "", ErrSceneDisabled
	}

	// Check MQTT availability
	if e.mqtt == nil {
		return "", ErrMQTTUnavailable
	}

	// Create execution record
	now := time.Now().UTC()
	exec := &SceneExecution{
		ID:           GenerateID(),
		SceneID:      sceneID,
		TriggeredAt:  now,
		TriggerType:  triggerType,
		Status:       StatusPending,
		ActionsTotal: len(scene.Actions),
	}
	if triggerSource != "" {
		exec.TriggerSource = &triggerSource
	}

	// Persist initial execution record
	if createErr := e.repo.CreateExecution(ctx, exec); createErr != nil {
		e.logger.Error("failed to create execution record", "error", createErr)
		// Continue execution even if logging fails — scene activation is more important
	}

	// Start execution
	started := time.Now().UTC()
	exec.StartedAt = &started
	exec.Status = StatusRunning

	e.logger.Info("scene activation started",
		"scene_id", sceneID,
		"scene_name", scene.Name,
		"execution_id", exec.ID,
		"actions", len(scene.Actions),
	)

	// Group actions by parallel flag and execute
	groups := groupActions(scene.Actions)
	var failures []ActionFailure
	completed := 0
	failed := 0
	skipped := 0
	aborted := false

	for _, group := range groups {
		if aborted {
			skipped += len(group)
			continue
		}

		// Check context cancellation between groups
		select {
		case <-ctx.Done():
			skipped += len(group)
			exec.Status = StatusCancelled
			aborted = true
			continue
		default:
		}

		// Execute group (all actions in parallel)
		groupFailures := e.executeGroup(ctx, scene.ID, exec.ID, group)
		completed += len(group) - len(groupFailures)
		failed += len(groupFailures)
		failures = append(failures, groupFailures...)

		// Check if we should abort (any action with ContinueOnError=false failed)
		for _, gf := range groupFailures {
			// ActionIndex is group-relative, so index directly into the group
			if gf.ActionIndex >= 0 && gf.ActionIndex < len(group) {
				if !group[gf.ActionIndex].ContinueOnError {
					aborted = true
					break
				}
			}
		}
	}

	// Determine final status
	completedAt := time.Now().UTC()
	exec.CompletedAt = &completedAt
	exec.ActionsCompleted = completed
	exec.ActionsFailed = failed
	exec.ActionsSkipped = skipped
	exec.Failures = failures
	duration := int(completedAt.Sub(started).Milliseconds())
	exec.DurationMS = &duration

	switch {
	case exec.Status == StatusCancelled:
		// Already set
	case failed > 0 && aborted:
		exec.Status = StatusFailed
	case failed > 0:
		exec.Status = StatusPartial
	default:
		exec.Status = StatusCompleted
	}

	// Update execution record
	if updateErr := e.repo.UpdateExecution(ctx, exec); updateErr != nil {
		e.logger.Error("failed to update execution record", "error", updateErr)
	}

	e.logger.Info("scene activation complete",
		"scene_id", sceneID,
		"execution_id", exec.ID,
		"status", exec.Status,
		"completed", completed,
		"failed", failed,
		"skipped", skipped,
		"duration_ms", duration,
	)

	// Broadcast WebSocket event
	if e.hub != nil {
		e.hub.Broadcast("scene.activated", map[string]any{
			"scene_id":     sceneID,
			"scene_name":   scene.Name,
			"execution_id": exec.ID,
			"status":       string(exec.Status),
			"duration_ms":  duration,
		})
	}

	return exec.ID, nil
}

// executeGroup executes all actions in a group concurrently.
// Returns a slice of failures (empty if all succeeded).
func (e *Engine) executeGroup(ctx context.Context, sceneID, executionID string, actions []SceneAction) []ActionFailure {
	var (
		mu       sync.Mutex
		failures []ActionFailure
		wg       sync.WaitGroup
	)

	for i, action := range actions {
		wg.Add(1)
		go func(idx int, a SceneAction) {
			defer wg.Done()

			if err := e.executeAction(ctx, sceneID, executionID, a); err != nil {
				mu.Lock()
				failures = append(failures, ActionFailure{
					ActionIndex: idx,
					DeviceID:    a.DeviceID,
					Command:     a.Command,
					ErrorCode:   "EXECUTION_FAILED",
					ErrorMsg:    err.Error(),
				})
				mu.Unlock()
			}
		}(i, action)
	}

	wg.Wait()
	return failures
}

// executeAction executes a single scene action.
// It handles delay, device lookup, and MQTT command publishing.
func (e *Engine) executeAction(ctx context.Context, sceneID, executionID string, action SceneAction) error {
	// Handle delay
	if action.DelayMS > 0 {
		select {
		case <-time.After(time.Duration(action.DelayMS) * time.Millisecond):
		case <-ctx.Done():
			return fmt.Errorf("action delayed: %w", ctx.Err())
		}
	}

	// Look up device for routing
	dev, err := e.devices.GetDevice(ctx, action.DeviceID)
	if err != nil {
		return fmt.Errorf("device %q: %w", action.DeviceID, err)
	}

	// Build MQTT command payload
	commandID := GenerateID()
	params := action.Parameters
	if params == nil {
		params = make(map[string]any)
	}

	// Add fade_ms to parameters if set
	if action.FadeMS > 0 {
		// Deep copy to prevent shared mutable state in parallel execution
		paramsCopy := deepCopyMap(params)
		if paramsCopy == nil {
			paramsCopy = make(map[string]any, 1)
		}
		paramsCopy["fade_ms"] = action.FadeMS
		params = paramsCopy
	}

	mqttPayload := map[string]any{
		"id":           commandID,
		"device_id":    action.DeviceID,
		"command":      action.Command,
		"parameters":   params,
		"source":       "scene:" + sceneID,
		"execution_id": executionID,
	}

	payload, marshalErr := json.Marshal(mqttPayload)
	if marshalErr != nil {
		return fmt.Errorf("marshalling command: %w", marshalErr)
	}

	// Publish command using flat topic scheme: graylogic/command/{protocol}/{device_id}
	topic := "graylogic/command/" + dev.Protocol + "/" + action.DeviceID

	if pubErr := e.mqtt.Publish(topic, payload, 1, false); pubErr != nil {
		return fmt.Errorf("publishing to %q: %w", topic, pubErr)
	}

	e.logger.Debug("scene action published",
		"scene_id", sceneID,
		"device_id", action.DeviceID,
		"command", action.Command,
		"topic", topic,
	)

	return nil
}

// groupActions splits actions into sequential groups based on the Parallel flag.
//
// The first action always starts a new group. Subsequent actions with
// Parallel=true join the current group; Parallel=false starts a new group.
//
// Example:
//
//	actions: [A(parallel=false), B(parallel=true), C(parallel=true), D(parallel=false)]
//	groups:  [[A, B, C], [D]]
//
// Group 1 (A, B, C) executes concurrently, then group 2 (D) executes after.
func groupActions(actions []SceneAction) [][]SceneAction {
	if len(actions) == 0 {
		return nil
	}

	var groups [][]SceneAction
	current := []SceneAction{actions[0]}

	for _, action := range actions[1:] {
		if action.Parallel {
			current = append(current, action)
		} else {
			groups = append(groups, current)
			current = []SceneAction{action}
		}
	}
	groups = append(groups, current)
	return groups
}
