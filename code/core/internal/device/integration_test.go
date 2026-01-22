package device_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nerrad567/gray-logic-core/internal/device"
)

// setupIntegrationDB creates an in-memory SQLite database with the full devices schema.
// This mirrors the production migration (20260118_200000_initial_schema.up.sql).
func setupIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			room_id TEXT,
			area_id TEXT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			type TEXT NOT NULL,
			domain TEXT NOT NULL,
			protocol TEXT NOT NULL,
			address TEXT NOT NULL,
			gateway_id TEXT,
			capabilities TEXT NOT NULL DEFAULT '[]',
			config TEXT NOT NULL DEFAULT '{}',
			state TEXT NOT NULL DEFAULT '{}',
			state_updated_at TEXT,
			health_status TEXT NOT NULL DEFAULT 'unknown',
			health_last_seen TEXT,
			phm_enabled INTEGER NOT NULL DEFAULT 0,
			phm_baseline TEXT,
			manufacturer TEXT,
			model TEXT,
			firmware_version TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		) STRICT;
		CREATE INDEX idx_devices_room_id ON devices(room_id);
		CREATE INDEX idx_devices_area_id ON devices(area_id);
		CREATE INDEX idx_devices_domain ON devices(domain);
		CREATE INDEX idx_devices_protocol ON devices(protocol);
		CREATE INDEX idx_devices_type ON devices(type);
		CREATE INDEX idx_devices_health ON devices(health_status);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// TestIntegration_FullDeviceLifecycle exercises the complete path:
// SQLiteRepository → Registry → cache → state/health updates → delete.
// This is the flow that main.go relies on at startup.
func TestIntegration_FullDeviceLifecycle(t *testing.T) {
	db := setupIntegrationDB(t)
	ctx := context.Background()

	// Wire up exactly as main.go does
	repo := device.NewSQLiteRepository(db)
	registry := device.NewRegistry(repo)

	// RefreshCache on empty database should succeed
	if err := registry.RefreshCache(ctx); err != nil {
		t.Fatalf("RefreshCache() on empty DB: %v", err)
	}
	if registry.GetDeviceCount() != 0 {
		t.Fatalf("expected 0 devices after refresh, got %d", registry.GetDeviceCount())
	}

	// Create a KNX dimmer
	dev := &device.Device{
		Name:         "Living Room Dimmer",
		Type:         device.DeviceTypeLightDimmer,
		Domain:       device.DomainLighting,
		Protocol:     device.ProtocolKNX,
		Address:      device.Address{"group_address": "1/2/3"},
		Capabilities: []device.Capability{device.CapOnOff, device.CapDim},
		Config:       device.Config{"transition_time": 500},
		State:        device.State{},
		HealthStatus: device.HealthStatusUnknown,
	}

	if err := registry.CreateDevice(ctx, dev); err != nil {
		t.Fatalf("CreateDevice() error: %v", err)
	}
	if dev.ID == "" {
		t.Fatal("expected ID to be generated")
	}
	if dev.Slug != "living-room-dimmer" {
		t.Errorf("Slug = %q, want %q", dev.Slug, "living-room-dimmer")
	}

	deviceID := dev.ID

	// Verify in-cache retrieval
	got, err := registry.GetDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("GetDevice() error: %v", err)
	}
	if got.Name != "Living Room Dimmer" {
		t.Errorf("Name = %q, want %q", got.Name, "Living Room Dimmer")
	}

	// Simulate what the KNX bridge does: state + health updates
	newState := device.State{"on": true, "level": 75.0}
	if stateErr := registry.SetDeviceState(ctx, deviceID, newState); stateErr != nil {
		t.Fatalf("SetDeviceState() error: %v", stateErr)
	}
	if healthErr := registry.SetDeviceHealth(ctx, deviceID, device.HealthStatusOnline); healthErr != nil {
		t.Fatalf("SetDeviceHealth() error: %v", healthErr)
	}

	// Verify state persisted through cache
	got, _ = registry.GetDevice(ctx, deviceID)
	if on, ok := got.State["on"]; !ok || on != true {
		t.Errorf("State[on] = %v, want true", got.State["on"])
	}
	if got.HealthStatus != device.HealthStatusOnline {
		t.Errorf("HealthStatus = %q, want %q", got.HealthStatus, device.HealthStatusOnline)
	}
	if got.StateUpdatedAt == nil {
		t.Error("StateUpdatedAt should be set after SetDeviceState")
	}
	if got.HealthLastSeen == nil {
		t.Error("HealthLastSeen should be set after SetDeviceHealth")
	}

	// Verify persistence: create a new registry from the same DB and RefreshCache
	registry2 := device.NewRegistry(repo)
	if refreshErr := registry2.RefreshCache(ctx); refreshErr != nil {
		t.Fatalf("RefreshCache() on second registry: %v", refreshErr)
	}
	if registry2.GetDeviceCount() != 1 {
		t.Fatalf("expected 1 device after refresh, got %d", registry2.GetDeviceCount())
	}

	got2, err := registry2.GetDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("GetDevice() from second registry: %v", err)
	}
	if got2.Name != "Living Room Dimmer" {
		t.Errorf("persisted Name = %q, want %q", got2.Name, "Living Room Dimmer")
	}
	if got2.HealthStatus != device.HealthStatusOnline {
		t.Errorf("persisted HealthStatus = %q, want %q", got2.HealthStatus, device.HealthStatusOnline)
	}

	// Update device name
	got.Name = "Lounge Dimmer"
	if updateErr := registry.UpdateDevice(ctx, got); updateErr != nil {
		t.Fatalf("UpdateDevice() error: %v", updateErr)
	}
	updated, _ := registry.GetDevice(ctx, deviceID)
	if updated.Name != "Lounge Dimmer" {
		t.Errorf("updated Name = %q, want %q", updated.Name, "Lounge Dimmer")
	}

	// Delete device
	if delErr := registry.DeleteDevice(ctx, deviceID); delErr != nil {
		t.Fatalf("DeleteDevice() error: %v", delErr)
	}
	if registry.GetDeviceCount() != 0 {
		t.Errorf("expected 0 devices after delete, got %d", registry.GetDeviceCount())
	}

	// Verify deletion persisted
	_, err = registry.GetDevice(ctx, deviceID)
	if !errors.Is(err, device.ErrDeviceNotFound) {
		t.Errorf("expected ErrDeviceNotFound after delete, got: %v", err)
	}
}

// TestIntegration_MultipleDevicesAndQueries tests batch operations across
// multiple devices with different domains, protocols, and rooms.
func TestIntegration_MultipleDevicesAndQueries(t *testing.T) {
	db := setupIntegrationDB(t)
	ctx := context.Background()

	repo := device.NewSQLiteRepository(db)
	registry := device.NewRegistry(repo)
	registry.RefreshCache(ctx)

	roomLiving := "room-living"
	roomBedroom := "room-bedroom"

	devices := []*device.Device{
		{
			Name:         "Living Light",
			Type:         device.DeviceTypeLightDimmer,
			Domain:       device.DomainLighting,
			Protocol:     device.ProtocolKNX,
			Address:      device.Address{"group_address": "1/0/1"},
			Capabilities: []device.Capability{device.CapOnOff, device.CapDim},
			RoomID:       &roomLiving,
		},
		{
			Name:         "Living Thermostat",
			Type:         device.DeviceTypeThermostat,
			Domain:       device.DomainClimate,
			Protocol:     device.ProtocolModbusTCP,
			Address:      device.Address{"host": "192.168.1.50", "unit_id": 1},
			Capabilities: []device.Capability{device.CapTemperatureRead, device.CapTemperatureSet},
			RoomID:       &roomLiving,
		},
		{
			Name:     "Bedroom Blind",
			Type:     device.DeviceTypeBlindPosition,
			Domain:   device.DomainBlinds,
			Protocol: device.ProtocolKNX,
			Address:  device.Address{"group_address": "2/0/1"},
			RoomID:   &roomBedroom,
		},
	}

	for _, d := range devices {
		if err := registry.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice(%s) error: %v", d.Name, err)
		}
	}

	if registry.GetDeviceCount() != 3 {
		t.Fatalf("expected 3 devices, got %d", registry.GetDeviceCount())
	}

	// Query by room
	livingDevices, err := registry.GetDevicesByRoom(ctx, roomLiving)
	if err != nil {
		t.Fatalf("GetDevicesByRoom() error: %v", err)
	}
	if len(livingDevices) != 2 {
		t.Errorf("living room devices = %d, want 2", len(livingDevices))
	}

	// Query by domain
	lightingDevices, err := registry.GetDevicesByDomain(ctx, device.DomainLighting)
	if err != nil {
		t.Fatalf("GetDevicesByDomain() error: %v", err)
	}
	if len(lightingDevices) != 1 {
		t.Errorf("lighting devices = %d, want 1", len(lightingDevices))
	}

	// Query by protocol
	knxDevices, err := registry.GetDevicesByProtocol(ctx, device.ProtocolKNX)
	if err != nil {
		t.Fatalf("GetDevicesByProtocol() error: %v", err)
	}
	if len(knxDevices) != 2 {
		t.Errorf("KNX devices = %d, want 2", len(knxDevices))
	}

	// Query by slug
	blind, err := registry.GetDeviceBySlug(ctx, "bedroom-blind")
	if err != nil {
		t.Fatalf("GetDeviceBySlug() error: %v", err)
	}
	if blind.Domain != device.DomainBlinds {
		t.Errorf("blind domain = %q, want %q", blind.Domain, device.DomainBlinds)
	}

	// Stats
	stats := registry.GetStats()
	if stats.TotalDevices != 3 {
		t.Errorf("stats.TotalDevices = %d, want 3", stats.TotalDevices)
	}
	if stats.ByDomain[device.DomainLighting] != 1 {
		t.Errorf("stats.ByDomain[lighting] = %d, want 1", stats.ByDomain[device.DomainLighting])
	}
	if stats.ByProtocol[device.ProtocolKNX] != 2 {
		t.Errorf("stats.ByProtocol[knx] = %d, want 2", stats.ByProtocol[device.ProtocolKNX])
	}
}

// TestIntegration_CacheConsistencyAfterRestart simulates what happens when
// the application restarts: devices from a previous session are loaded from
// the database into a fresh registry cache.
func TestIntegration_CacheConsistencyAfterRestart(t *testing.T) {
	db := setupIntegrationDB(t)
	ctx := context.Background()

	repo := device.NewSQLiteRepository(db)

	// Session 1: Create devices and update state
	r1 := device.NewRegistry(repo)
	r1.RefreshCache(ctx)

	dev := &device.Device{
		Name:     "Persistent Device",
		Type:     device.DeviceTypeLightSwitch,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address:  device.Address{"group_address": "1/0/0"},
	}
	if err := r1.CreateDevice(ctx, dev); err != nil {
		t.Fatalf("CreateDevice() error: %v", err)
	}
	deviceID := dev.ID

	// Simulate runtime state changes
	r1.SetDeviceState(ctx, deviceID, device.State{"on": true})
	r1.SetDeviceHealth(ctx, deviceID, device.HealthStatusOnline)

	// Session 2: Fresh registry from same database (simulates restart)
	r2 := device.NewRegistry(repo)
	if err := r2.RefreshCache(ctx); err != nil {
		t.Fatalf("RefreshCache() session 2: %v", err)
	}

	got, err := r2.GetDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("GetDevice() session 2: %v", err)
	}

	// State should be persisted
	if on, ok := got.State["on"]; !ok || on != true {
		t.Errorf("persisted State[on] = %v, want true", got.State["on"])
	}
	if got.HealthStatus != device.HealthStatusOnline {
		t.Errorf("persisted HealthStatus = %q, want %q", got.HealthStatus, device.HealthStatusOnline)
	}
}

// TestIntegration_RapidStateUpdates simulates a busy KNX bus where the bridge
// sends many state updates in quick succession (as happens with dimmer ramps).
func TestIntegration_RapidStateUpdates(t *testing.T) {
	db := setupIntegrationDB(t)
	ctx := context.Background()

	repo := device.NewSQLiteRepository(db)
	registry := device.NewRegistry(repo)
	registry.RefreshCache(ctx)

	dev := &device.Device{
		Name:     "Ramping Dimmer",
		Type:     device.DeviceTypeLightDimmer,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address:  device.Address{"group_address": "1/1/1"},
	}
	if err := registry.CreateDevice(ctx, dev); err != nil {
		t.Fatalf("CreateDevice() error: %v", err)
	}

	// Simulate a dimmer ramping from 0 to 100
	for i := 0; i <= 100; i += 5 {
		state := device.State{"on": true, "level": float64(i)}
		if err := registry.SetDeviceState(ctx, dev.ID, state); err != nil {
			t.Fatalf("SetDeviceState(level=%d) error: %v", i, err)
		}
	}

	// Final state should be level=100
	got, _ := registry.GetDevice(ctx, dev.ID)
	level, ok := got.State["level"].(float64)
	if !ok || level != 100 {
		t.Errorf("final level = %v, want 100", got.State["level"])
	}

	// Verify last update time is recent
	if got.StateUpdatedAt == nil {
		t.Fatal("StateUpdatedAt should be set")
	}
	if time.Since(*got.StateUpdatedAt) > 5*time.Second {
		t.Error("StateUpdatedAt seems too old")
	}
}
