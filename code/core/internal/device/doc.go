// Package device provides the Device Registry for Gray Logic Core.
//
// The Device Registry is the central catalogue of all controllable and
// monitorable entities in a Gray Logic installation. It manages device
// lifecycle, state, and provides query operations for the REST API and
// automation engine.
//
// # Architecture
//
//	┌─────────────────────────────────────────────────────────────────────────┐
//	│                          Device Registry                                 │
//	│                                                                          │
//	│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐   │
//	│  │     Registry     │    │    Repository    │    │    Validation    │   │
//	│  │   (registry.go)  │───▶│  (repository.go) │    │ (validation.go)  │   │
//	│  │                  │    │                  │    │                  │   │
//	│  │ • CRUD ops       │    │ • SQLite queries │    │ • Device checks  │   │
//	│  │ • In-memory cache│    │ • JSON marshal   │    │ • Address valid. │   │
//	│  │ • Thread safety  │    │ • Transactions   │    │ • Slug generation│   │
//	│  └──────────────────┘    └──────────────────┘    └──────────────────┘   │
//	│           │                       │                                      │
//	└───────────│───────────────────────│──────────────────────────────────────┘
//	            │                       │
//	            ▼                       ▼
//	┌──────────────────────┐   ┌──────────────────────┐
//	│    REST API (M1.4)   │   │   SQLite Database    │
//	│  • GET /devices      │   │   (devices table)    │
//	│  • POST /devices     │   └──────────────────────┘
//	│  • WebSocket state   │
//	└──────────────────────┘
//
// # Key Types
//
//   - Device: The core entity representing a controllable/monitorable device
//   - Domain: Functional area (lighting, climate, blinds, etc.)
//   - Protocol: Communication protocol (KNX, DALI, Modbus, etc.)
//   - DeviceType: Specific device classification (light_dimmer, thermostat, etc.)
//   - Capability: What a device can do (on_off, dim, temperature_read, etc.)
//
// # Usage
//
//	// Create repository and registry
//	repo := device.NewSQLiteRepository(db)
//	registry := device.NewRegistry(repo)
//	registry.SetLogger(log)
//
//	// Load devices into cache on startup
//	if err := registry.RefreshCache(ctx); err != nil {
//	    return err
//	}
//
//	// Create a new device
//	dev := &device.Device{
//	    Name:     "Living Room Dimmer",
//	    Type:     device.DeviceTypeLightDimmer,
//	    Domain:   device.DomainLighting,
//	    Protocol: device.ProtocolKNX,
//	    Address:  device.Address{"functions": map[string]any{
//	        "switch":     map[string]any{"ga": "1/2/3", "dpt": "1.001", "flags": []any{"write"}},
//	        "brightness": map[string]any{"ga": "1/2/4", "dpt": "5.001", "flags": []any{"write"}},
//	    }},
//	    Capabilities: []device.Capability{
//	        device.CapOnOff,
//	        device.CapDim,
//	    },
//	}
//	if err := registry.CreateDevice(ctx, dev); err != nil {
//	    return err
//	}
//
//	// Query devices
//	devices, _ := registry.GetDevicesByDomain(ctx, device.DomainLighting)
//	device, _ := registry.GetDevice(ctx, "device-uuid")
//
//	// Update state (from protocol bridge)
//	registry.SetDeviceState(ctx, id, device.State{"on": true, "level": 75})
//
// # Thread Safety
//
// The Registry is safe for concurrent use. All operations are protected by
// a read-write mutex. The Repository implementation must also be thread-safe.
//
// # Related Documentation
//
//   - docs/data-model/entities.md — Device entity specification
//   - docs/architecture/core-internals.md — Core architecture overview
//   - migrations/20260118_200000_initial_schema.up.sql — Database schema
package device
