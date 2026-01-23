// Package automation provides the scene engine for Gray Logic Core.
//
// Scenes are named collections of device commands that execute together.
// Each scene contains ordered actions that can run in parallel or
// sequentially, with optional delays and fade transitions.
//
// Architecture:
//
//	┌───────────────────────────────────────────────────────┐
//	│                  Engine (engine.go)                    │
//	│  Orchestrates scene execution with parallel support    │
//	│  ┌──────────────┐    ┌──────────────┐                │
//	│  │   Registry   │───▶│  Repository  │                │
//	│  │(registry.go) │    │(repository.go)│               │
//	│  └──────────────┘    └──────────────┘                │
//	│        │                                              │
//	│        ▼                                              │
//	│  ┌──────────────────────────────────────────────┐    │
//	│  │  Execution Pipeline                           │    │
//	│  │  1. Load scene (cached)                       │    │
//	│  │  2. Group actions by parallel flag            │    │
//	│  │  3. Execute groups: goroutines + WaitGroup    │    │
//	│  │  4. Publish MQTT commands to bridges          │    │
//	│  │  5. Log execution result                      │    │
//	│  │  6. Broadcast WebSocket event                 │    │
//	│  └──────────────────────────────────────────────┘    │
//	└───────────────────────────────────────────────────────┘
//
// # Key Types
//
//   - Scene: Named collection of device actions with metadata
//   - SceneAction: Individual device command (device_id, command, parameters)
//   - SceneExecution: Audit record of a scene activation
//   - Engine: Orchestrator that activates scenes via MQTT
//   - Registry: Thread-safe in-memory cache wrapping Repository
//
// # Thread Safety
//
// Registry and Engine are safe for concurrent use from multiple goroutines.
// All public methods use appropriate synchronisation.
//
// # Usage
//
//	repo := automation.NewSQLiteRepository(db)
//	registry := automation.NewRegistry(repo)
//	registry.SetLogger(log)
//
//	if err := registry.RefreshCache(ctx); err != nil {
//	    return err
//	}
//
//	engine := automation.NewEngine(registry, devices, mqtt, hub, repo, log)
//	executionID, err := engine.ActivateScene(ctx, "cinema-mode", "manual", "api")
//
// See docs/automation/automation.md for the full automation specification.
package automation
