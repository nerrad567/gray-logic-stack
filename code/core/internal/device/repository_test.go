package device

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with the devices table.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create devices table matching the schema
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
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// testDevice creates a device for testing.
func testDevice(id, name string) *Device {
	return &Device{
		ID:       id,
		Name:     name,
		Slug:     GenerateSlug(name),
		Type:     DeviceTypeLightDimmer,
		Domain:   DomainLighting,
		Protocol: ProtocolKNX,
		Address: Address{"functions": map[string]any{
			"switch":     map[string]any{"ga": "1/2/3", "dpt": "1.001", "flags": []any{"write"}},
			"brightness": map[string]any{"ga": "1/2/4", "dpt": "5.001", "flags": []any{"write"}},
		}},
		Capabilities: []Capability{CapOnOff, CapDim},
		Config:       Config{},
		State:        State{},
		HealthStatus: HealthStatusUnknown,
	}
}

func TestSQLiteRepository_Create(t *testing.T) { //nolint:gocognit // comprehensive table-driven test
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	t.Run("creates device successfully", func(t *testing.T) {
		device := testDevice("dev-001", "Living Room Light")

		err := repo.Create(ctx, device)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Verify it was created
		got, err := repo.GetByID(ctx, "dev-001")
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.Name != "Living Room Light" {
			t.Errorf("Name = %q, want %q", got.Name, "Living Room Light")
		}
		if got.Domain != DomainLighting {
			t.Errorf("Domain = %q, want %q", got.Domain, DomainLighting)
		}
	})

	t.Run("returns error for duplicate ID", func(t *testing.T) {
		device := testDevice("dev-duplicate", "First Device")
		if err := repo.Create(ctx, device); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		device2 := testDevice("dev-duplicate", "Second Device")
		err := repo.Create(ctx, device2)
		if !errors.Is(err, ErrDeviceExists) {
			t.Errorf("Create() error = %v, want ErrDeviceExists", err)
		}
	})

	t.Run("stores all fields correctly", func(t *testing.T) {
		roomID := "room-001"
		areaID := "area-001"
		gatewayID := "gw-001"
		manufacturer := "ACME"
		model := "Dimmer Pro"
		firmware := "1.2.3"
		stateTime := time.Now().UTC().Truncate(time.Second)
		healthTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

		device := &Device{
			ID:              "dev-full",
			RoomID:          &roomID,
			AreaID:          &areaID,
			Name:            "Full Device",
			Slug:            "full-device",
			Type:            DeviceTypeLightRGBW,
			Domain:          DomainLighting,
			Protocol:        ProtocolDALI,
			Address:         Address{"gateway": "dali-gw", "short_address": 15},
			GatewayID:       &gatewayID,
			Capabilities:    []Capability{CapOnOff, CapDim, CapColorRGB},
			Config:          Config{"transition_time": 500},
			State:           State{"on": true, "level": 75},
			StateUpdatedAt:  &stateTime,
			HealthStatus:    HealthStatusOnline,
			HealthLastSeen:  &healthTime,
			PHMEnabled:      true,
			PHMBaseline:     &PHMBaseline{"avg_power": 25.5},
			Manufacturer:    &manufacturer,
			Model:           &model,
			FirmwareVersion: &firmware,
		}

		if err := repo.Create(ctx, device); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repo.GetByID(ctx, "dev-full")
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		// Verify all fields
		if got.RoomID == nil || *got.RoomID != roomID {
			t.Errorf("RoomID = %v, want %q", got.RoomID, roomID)
		}
		if got.AreaID == nil || *got.AreaID != areaID {
			t.Errorf("AreaID = %v, want %q", got.AreaID, areaID)
		}
		if got.GatewayID == nil || *got.GatewayID != gatewayID {
			t.Errorf("GatewayID = %v, want %q", got.GatewayID, gatewayID)
		}
		if got.Type != DeviceTypeLightRGBW {
			t.Errorf("Type = %q, want %q", got.Type, DeviceTypeLightRGBW)
		}
		if got.Protocol != ProtocolDALI {
			t.Errorf("Protocol = %q, want %q", got.Protocol, ProtocolDALI)
		}
		if len(got.Capabilities) != 3 {
			t.Errorf("Capabilities count = %d, want 3", len(got.Capabilities))
		}
		if got.HealthStatus != HealthStatusOnline {
			t.Errorf("HealthStatus = %q, want %q", got.HealthStatus, HealthStatusOnline)
		}
		if !got.PHMEnabled {
			t.Error("PHMEnabled = false, want true")
		}
		if got.PHMBaseline == nil {
			t.Error("PHMBaseline = nil, want non-nil")
		}
		if got.Manufacturer == nil || *got.Manufacturer != manufacturer {
			t.Errorf("Manufacturer = %v, want %q", got.Manufacturer, manufacturer)
		}

		// Check address was stored
		if ga, ok := got.Address["gateway"]; !ok || ga != "dali-gw" {
			t.Errorf("Address[gateway] = %v, want %q", ga, "dali-gw")
		}

		// Check state was stored
		if on, ok := got.State["on"]; !ok || on != true {
			t.Errorf("State[on] = %v, want true", on)
		}
	})
}

func TestSQLiteRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a test device
	device := testDevice("dev-get", "Test Device")
	if err := repo.Create(ctx, device); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("returns device when found", func(t *testing.T) {
		got, err := repo.GetByID(ctx, "dev-get")
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.ID != "dev-get" {
			t.Errorf("ID = %q, want %q", got.ID, "dev-get")
		}
	})

	t.Run("returns ErrDeviceNotFound when not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "nonexistent")
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("GetByID() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestSQLiteRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	t.Run("returns empty list when no devices", func(t *testing.T) {
		devices, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(devices) != 0 {
			t.Errorf("List() returned %d devices, want 0", len(devices))
		}
	})

	// Create test devices
	for i := 1; i <= 3; i++ {
		device := testDevice(
			GenerateID(),
			[]string{"Alpha Light", "Beta Light", "Gamma Light"}[i-1],
		)
		if err := repo.Create(ctx, device); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	t.Run("returns all devices ordered by name", func(t *testing.T) {
		devices, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(devices) != 3 {
			t.Fatalf("List() returned %d devices, want 3", len(devices))
		}
		// Should be alphabetically sorted
		if devices[0].Name != "Alpha Light" {
			t.Errorf("First device = %q, want %q", devices[0].Name, "Alpha Light")
		}
		if devices[1].Name != "Beta Light" {
			t.Errorf("Second device = %q, want %q", devices[1].Name, "Beta Light")
		}
		if devices[2].Name != "Gamma Light" {
			t.Errorf("Third device = %q, want %q", devices[2].Name, "Gamma Light")
		}
	})
}

func TestSQLiteRepository_ListByRoom(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	roomID1 := "room-001"
	roomID2 := "room-002"

	// Create devices in different rooms
	dev1 := testDevice("dev-r1-1", "Room 1 Device A")
	dev1.RoomID = &roomID1
	dev2 := testDevice("dev-r1-2", "Room 1 Device B")
	dev2.RoomID = &roomID1
	dev3 := testDevice("dev-r2-1", "Room 2 Device")
	dev3.RoomID = &roomID2

	for _, d := range []*Device{dev1, dev2, dev3} {
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	t.Run("returns devices in room", func(t *testing.T) {
		devices, err := repo.ListByRoom(ctx, roomID1)
		if err != nil {
			t.Fatalf("ListByRoom() error = %v", err)
		}
		if len(devices) != 2 {
			t.Errorf("ListByRoom() returned %d devices, want 2", len(devices))
		}
	})

	t.Run("returns empty for nonexistent room", func(t *testing.T) {
		devices, err := repo.ListByRoom(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("ListByRoom() error = %v", err)
		}
		if len(devices) != 0 {
			t.Errorf("ListByRoom() returned %d devices, want 0", len(devices))
		}
	})
}

func TestSQLiteRepository_ListByDomain(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create devices in different domains
	light := testDevice("dev-light", "Light")
	light.Domain = DomainLighting

	climate := testDevice("dev-climate", "Thermostat")
	climate.Domain = DomainClimate
	climate.Type = DeviceTypeThermostat

	for _, d := range []*Device{light, climate} {
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	t.Run("returns devices in domain", func(t *testing.T) {
		devices, err := repo.ListByDomain(ctx, DomainLighting)
		if err != nil {
			t.Fatalf("ListByDomain() error = %v", err)
		}
		if len(devices) != 1 {
			t.Errorf("ListByDomain() returned %d devices, want 1", len(devices))
		}
		if devices[0].Name != "Light" {
			t.Errorf("Device name = %q, want %q", devices[0].Name, "Light")
		}
	})
}

func TestSQLiteRepository_ListByProtocol(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create devices with different protocols
	knxDevice := testDevice("dev-knx", "KNX Device")
	knxDevice.Protocol = ProtocolKNX

	daliDevice := testDevice("dev-dali", "DALI Device")
	daliDevice.Protocol = ProtocolDALI
	daliDevice.Address = Address{"gateway": "gw1", "short_address": 1}

	for _, d := range []*Device{knxDevice, daliDevice} {
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	t.Run("returns devices by protocol", func(t *testing.T) {
		devices, err := repo.ListByProtocol(ctx, ProtocolKNX)
		if err != nil {
			t.Fatalf("ListByProtocol() error = %v", err)
		}
		if len(devices) != 1 {
			t.Errorf("ListByProtocol() returned %d devices, want 1", len(devices))
		}
		if devices[0].Protocol != ProtocolKNX {
			t.Errorf("Protocol = %q, want %q", devices[0].Protocol, ProtocolKNX)
		}
	})
}

func TestSQLiteRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a device
	device := testDevice("dev-update", "Original Name")
	if err := repo.Create(ctx, device); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("updates device successfully", func(t *testing.T) {
		device.Name = "Updated Name"
		device.HealthStatus = HealthStatusOnline
		device.State = State{"on": true}

		if err := repo.Update(ctx, device); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		got, err := repo.GetByID(ctx, "dev-update")
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.Name != "Updated Name" {
			t.Errorf("Name = %q, want %q", got.Name, "Updated Name")
		}
		if got.HealthStatus != HealthStatusOnline {
			t.Errorf("HealthStatus = %q, want %q", got.HealthStatus, HealthStatusOnline)
		}
		if on, ok := got.State["on"]; !ok || on != true {
			t.Errorf("State[on] = %v, want true", on)
		}
	})

	t.Run("returns ErrDeviceNotFound for nonexistent device", func(t *testing.T) {
		nonexistent := testDevice("nonexistent", "Ghost")
		err := repo.Update(ctx, nonexistent)
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("Update() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestSQLiteRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a device
	device := testDevice("dev-delete", "To Delete")
	if err := repo.Create(ctx, device); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("deletes device successfully", func(t *testing.T) {
		if err := repo.Delete(ctx, "dev-delete"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err := repo.GetByID(ctx, "dev-delete")
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("GetByID() after delete error = %v, want ErrDeviceNotFound", err)
		}
	})

	t.Run("returns ErrDeviceNotFound for nonexistent device", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent")
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("Delete() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestSQLiteRepository_UpdateState(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a device
	device := testDevice("dev-state", "Stateful Device")
	device.State = State{"on": false, "level": 0}
	if err := repo.Create(ctx, device); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("updates state successfully", func(t *testing.T) {
		newState := State{"on": true, "level": 75}
		if err := repo.UpdateState(ctx, "dev-state", newState); err != nil {
			t.Fatalf("UpdateState() error = %v", err)
		}

		got, err := repo.GetByID(ctx, "dev-state")
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if on, ok := got.State["on"]; !ok || on != true {
			t.Errorf("State[on] = %v, want true", on)
		}
		if level, ok := got.State["level"].(float64); !ok || level != 75 {
			t.Errorf("State[level] = %v, want 75", got.State["level"])
		}
		if got.StateUpdatedAt == nil {
			t.Error("StateUpdatedAt = nil, want non-nil")
		}
	})

	t.Run("returns ErrDeviceNotFound for nonexistent device", func(t *testing.T) {
		err := repo.UpdateState(ctx, "nonexistent", State{})
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("UpdateState() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestSQLiteRepository_UpdateHealth(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a device
	device := testDevice("dev-health", "Health Device")
	device.HealthStatus = HealthStatusUnknown
	if err := repo.Create(ctx, device); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("updates health successfully", func(t *testing.T) {
		lastSeen := time.Now().UTC().Truncate(time.Second)
		if err := repo.UpdateHealth(ctx, "dev-health", HealthStatusOnline, lastSeen); err != nil {
			t.Fatalf("UpdateHealth() error = %v", err)
		}

		got, err := repo.GetByID(ctx, "dev-health")
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.HealthStatus != HealthStatusOnline {
			t.Errorf("HealthStatus = %q, want %q", got.HealthStatus, HealthStatusOnline)
		}
		if got.HealthLastSeen == nil {
			t.Error("HealthLastSeen = nil, want non-nil")
		}
	})

	t.Run("returns ErrDeviceNotFound for nonexistent device", func(t *testing.T) {
		err := repo.UpdateHealth(ctx, "nonexistent", HealthStatusOnline, time.Now())
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("UpdateHealth() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestSQLiteRepository_ListByArea(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	areaID := "area-001"

	// Create devices
	dev1 := testDevice("dev-a1", "Area Device")
	dev1.AreaID = &areaID
	if err := repo.Create(ctx, dev1); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	devices, err := repo.ListByArea(ctx, areaID)
	if err != nil {
		t.Fatalf("ListByArea() error = %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("ListByArea() returned %d devices, want 1", len(devices))
	}
}
