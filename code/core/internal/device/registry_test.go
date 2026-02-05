package device

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// MockRepository is a test implementation of Repository.
type MockRepository struct {
	mu      sync.Mutex
	devices map[string]*Device
	// For testing error paths
	createErr       error
	updateErr       error
	deleteErr       error
	updateStateErr  error
	updateHealthErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		devices: make(map[string]*Device),
	}
}

func (m *MockRepository) GetByID(_ context.Context, id string) (*Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if d, ok := m.devices[id]; ok {
		copy := *d
		return &copy, nil
	}
	return nil, ErrDeviceNotFound
}

func (m *MockRepository) List(_ context.Context) ([]Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	devices := make([]Device, 0, len(m.devices))
	for _, d := range m.devices {
		devices = append(devices, *d)
	}
	return devices, nil
}

func (m *MockRepository) ListByRoom(_ context.Context, roomID string) ([]Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var devices []Device
	for _, d := range m.devices {
		if d.RoomID != nil && *d.RoomID == roomID {
			devices = append(devices, *d)
		}
	}
	return devices, nil
}

func (m *MockRepository) ListByArea(_ context.Context, areaID string) ([]Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var devices []Device
	for _, d := range m.devices {
		if d.AreaID != nil && *d.AreaID == areaID {
			devices = append(devices, *d)
		}
	}
	return devices, nil
}

func (m *MockRepository) ListByDomain(_ context.Context, domain Domain) ([]Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var devices []Device
	for _, d := range m.devices {
		if d.Domain == domain {
			devices = append(devices, *d)
		}
	}
	return devices, nil
}

func (m *MockRepository) ListByProtocol(_ context.Context, protocol Protocol) ([]Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var devices []Device
	for _, d := range m.devices {
		if d.Protocol == protocol {
			devices = append(devices, *d)
		}
	}
	return devices, nil
}

func (m *MockRepository) Create(_ context.Context, device *Device) error {
	if m.createErr != nil {
		return m.createErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.devices[device.ID]; exists {
		return ErrDeviceExists
	}

	copy := *device
	m.devices[device.ID] = &copy
	return nil
}

func (m *MockRepository) Update(_ context.Context, device *Device) error {
	if m.updateErr != nil {
		return m.updateErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.devices[device.ID]; !exists {
		return ErrDeviceNotFound
	}

	copy := *device
	m.devices[device.ID] = &copy
	return nil
}

func (m *MockRepository) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.devices[id]; !exists {
		return ErrDeviceNotFound
	}

	delete(m.devices, id)
	return nil
}

func (m *MockRepository) UpdateState(_ context.Context, id string, state State) error {
	if m.updateStateErr != nil {
		return m.updateStateErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	d, exists := m.devices[id]
	if !exists {
		return ErrDeviceNotFound
	}

	d.State = state
	now := time.Now().UTC()
	d.StateUpdatedAt = &now
	return nil
}

func (m *MockRepository) UpdateHealth(_ context.Context, id string, status HealthStatus, lastSeen time.Time) error {
	if m.updateHealthErr != nil {
		return m.updateHealthErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	d, exists := m.devices[id]
	if !exists {
		return ErrDeviceNotFound
	}

	d.HealthStatus = status
	d.HealthLastSeen = &lastSeen
	return nil
}

// addDevice adds a device directly to the mock for test setup.
func (m *MockRepository) addDevice(d *Device) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *d
	m.devices[d.ID] = &copy
}

func TestRegistry_RefreshCache(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Add devices to mock repo
	repo.addDevice(testDevice("dev-1", "Device 1"))
	repo.addDevice(testDevice("dev-2", "Device 2"))

	if err := registry.RefreshCache(ctx); err != nil {
		t.Fatalf("RefreshCache() error = %v", err)
	}

	if registry.GetDeviceCount() != 2 {
		t.Errorf("GetDeviceCount() = %d, want 2", registry.GetDeviceCount())
	}
}

func TestRegistry_GetDevice(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Add device to mock repo
	device := testDevice("dev-get", "Test Device")
	repo.addDevice(device)
	registry.RefreshCache(ctx)

	t.Run("returns device from cache", func(t *testing.T) {
		got, err := registry.GetDevice(ctx, "dev-get")
		if err != nil {
			t.Fatalf("GetDevice() error = %v", err)
		}
		if got.ID != "dev-get" {
			t.Errorf("ID = %q, want %q", got.ID, "dev-get")
		}
	})

	t.Run("returns ErrDeviceNotFound for nonexistent", func(t *testing.T) {
		_, err := registry.GetDevice(ctx, "nonexistent")
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("GetDevice() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestRegistry_CreateDevice(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	t.Run("creates device with generated ID and slug", func(t *testing.T) {
		device := &Device{
			Name:     "New Device",
			Type:     DeviceTypeLightDimmer,
			Domain:   DomainLighting,
			Protocol: ProtocolKNX,
			Address: Address{"functions": map[string]any{
				"switch": map[string]any{"ga": "1/2/3", "dpt": "1.001", "flags": []any{"write"}},
			}},
			Capabilities: []Capability{CapOnOff},
		}

		if err := registry.CreateDevice(ctx, device); err != nil {
			t.Fatalf("CreateDevice() error = %v", err)
		}

		// ID should be generated
		if device.ID == "" {
			t.Error("ID was not generated")
		}

		// Slug should be generated
		if device.Slug != "new-device" {
			t.Errorf("Slug = %q, want %q", device.Slug, "new-device")
		}

		// Should be in cache
		got, err := registry.GetDevice(ctx, device.ID)
		if err != nil {
			t.Fatalf("GetDevice() error = %v", err)
		}
		if got.Name != "New Device" {
			t.Errorf("Name = %q, want %q", got.Name, "New Device")
		}
	})

	t.Run("validates device before creating", func(t *testing.T) {
		device := &Device{
			Name: "", // Invalid: empty name
		}

		err := registry.CreateDevice(ctx, device)
		if !errors.Is(err, ErrInvalidName) {
			t.Errorf("CreateDevice() error = %v, want ErrInvalidName", err)
		}
	})

	t.Run("returns error for duplicate ID", func(t *testing.T) {
		device1 := testDevice("dup-id", "First")
		if err := registry.CreateDevice(ctx, device1); err != nil {
			t.Fatalf("first CreateDevice() error = %v", err)
		}

		device2 := testDevice("dup-id", "Second")
		err := registry.CreateDevice(ctx, device2)
		if !errors.Is(err, ErrDeviceExists) {
			t.Errorf("CreateDevice() error = %v, want ErrDeviceExists", err)
		}
	})
}

func TestRegistry_UpdateDevice(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create initial device
	device := testDevice("dev-update", "Original")
	if err := registry.CreateDevice(ctx, device); err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	t.Run("updates device successfully", func(t *testing.T) {
		device.Name = "Updated"
		device.HealthStatus = HealthStatusOnline

		if err := registry.UpdateDevice(ctx, device); err != nil {
			t.Fatalf("UpdateDevice() error = %v", err)
		}

		got, _ := registry.GetDevice(ctx, "dev-update")
		if got.Name != "Updated" {
			t.Errorf("Name = %q, want %q", got.Name, "Updated")
		}
		// Slug should be regenerated when name changes
		if got.Slug != "updated" {
			t.Errorf("Slug = %q, want %q", got.Slug, "updated")
		}
	})

	t.Run("returns ErrDeviceNotFound for nonexistent", func(t *testing.T) {
		nonexistent := testDevice("nonexistent", "Ghost")
		err := registry.UpdateDevice(ctx, nonexistent)
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("UpdateDevice() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestRegistry_DeleteDevice(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create device
	device := testDevice("dev-delete", "To Delete")
	if err := registry.CreateDevice(ctx, device); err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	t.Run("deletes device from cache and repo", func(t *testing.T) {
		if err := registry.DeleteDevice(ctx, "dev-delete"); err != nil {
			t.Fatalf("DeleteDevice() error = %v", err)
		}

		_, err := registry.GetDevice(ctx, "dev-delete")
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("GetDevice() error = %v, want ErrDeviceNotFound", err)
		}
	})

	t.Run("returns ErrDeviceNotFound for nonexistent", func(t *testing.T) {
		err := registry.DeleteDevice(ctx, "nonexistent")
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("DeleteDevice() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestRegistry_SetDeviceState(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create device
	device := testDevice("dev-state", "Stateful")
	if err := registry.CreateDevice(ctx, device); err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	t.Run("updates state in cache and repo", func(t *testing.T) {
		newState := State{"on": true, "level": 75}
		if err := registry.SetDeviceState(ctx, "dev-state", newState); err != nil {
			t.Fatalf("SetDeviceState() error = %v", err)
		}

		got, _ := registry.GetDevice(ctx, "dev-state")
		if on, ok := got.State["on"]; !ok || on != true {
			t.Errorf("State[on] = %v, want true", on)
		}
		if got.StateUpdatedAt == nil {
			t.Error("StateUpdatedAt = nil, want non-nil")
		}
	})

	t.Run("returns ErrDeviceNotFound for nonexistent", func(t *testing.T) {
		err := registry.SetDeviceState(ctx, "nonexistent", State{})
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("SetDeviceState() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestRegistry_SetDeviceHealth(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create device
	device := testDevice("dev-health", "Healthy")
	if err := registry.CreateDevice(ctx, device); err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	t.Run("updates health in cache and repo", func(t *testing.T) {
		if err := registry.SetDeviceHealth(ctx, "dev-health", HealthStatusOnline); err != nil {
			t.Fatalf("SetDeviceHealth() error = %v", err)
		}

		got, _ := registry.GetDevice(ctx, "dev-health")
		if got.HealthStatus != HealthStatusOnline {
			t.Errorf("HealthStatus = %q, want %q", got.HealthStatus, HealthStatusOnline)
		}
		if got.HealthLastSeen == nil {
			t.Error("HealthLastSeen = nil, want non-nil")
		}
	})
}

func TestRegistry_GetDevicesByDomain(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create devices in different domains
	light := testDevice("light-1", "Light")
	light.Domain = DomainLighting

	thermo := testDevice("thermo-1", "Thermostat")
	thermo.Domain = DomainClimate
	thermo.Type = DeviceTypeThermostat

	for _, d := range []*Device{light, thermo} {
		if err := registry.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice() error = %v", err)
		}
	}

	devices, err := registry.GetDevicesByDomain(ctx, DomainLighting)
	if err != nil {
		t.Fatalf("GetDevicesByDomain() error = %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("GetDevicesByDomain() returned %d devices, want 1", len(devices))
	}
	if devices[0].Domain != DomainLighting {
		t.Errorf("Domain = %q, want %q", devices[0].Domain, DomainLighting)
	}
}

func TestRegistry_GetDevicesByProtocol(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create devices with different protocols
	knx := testDevice("knx-1", "KNX Device")
	knx.Protocol = ProtocolKNX

	dali := testDevice("dali-1", "DALI Device")
	dali.Protocol = ProtocolDALI
	dali.Address = Address{"gateway": "gw", "short_address": 1}

	for _, d := range []*Device{knx, dali} {
		if err := registry.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice() error = %v", err)
		}
	}

	devices, err := registry.GetDevicesByProtocol(ctx, ProtocolKNX)
	if err != nil {
		t.Fatalf("GetDevicesByProtocol() error = %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("GetDevicesByProtocol() returned %d devices, want 1", len(devices))
	}
}

func TestRegistry_GetDeviceBySlug(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	device := testDevice("dev-slug", "Living Room Light")
	if err := registry.CreateDevice(ctx, device); err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	t.Run("finds device by slug", func(t *testing.T) {
		got, err := registry.GetDeviceBySlug(ctx, "living-room-light")
		if err != nil {
			t.Fatalf("GetDeviceBySlug() error = %v", err)
		}
		if got.ID != "dev-slug" {
			t.Errorf("ID = %q, want %q", got.ID, "dev-slug")
		}
	})

	t.Run("returns ErrDeviceNotFound for unknown slug", func(t *testing.T) {
		_, err := registry.GetDeviceBySlug(ctx, "nonexistent")
		if !errors.Is(err, ErrDeviceNotFound) {
			t.Errorf("GetDeviceBySlug() error = %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestRegistry_GetDevicesByCapability(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create devices with different capabilities
	dimmer := testDevice("dimmer-1", "Dimmer")
	dimmer.Capabilities = []Capability{CapOnOff, CapDim}

	switcher := testDevice("switch-1", "Switch")
	switcher.Capabilities = []Capability{CapOnOff}
	switcher.Type = DeviceTypeLightSwitch

	for _, d := range []*Device{dimmer, switcher} {
		if err := registry.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice() error = %v", err)
		}
	}

	// Both should have on_off
	devices, _ := registry.GetDevicesByCapability(ctx, CapOnOff)
	if len(devices) != 2 {
		t.Errorf("GetDevicesByCapability(on_off) returned %d devices, want 2", len(devices))
	}

	// Only dimmer has dim
	devices, _ = registry.GetDevicesByCapability(ctx, CapDim)
	if len(devices) != 1 {
		t.Errorf("GetDevicesByCapability(dim) returned %d devices, want 1", len(devices))
	}
}

func TestRegistry_GetStats(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create diverse devices
	light := testDevice("light", "Light")
	light.Domain = DomainLighting
	light.Protocol = ProtocolKNX
	light.HealthStatus = HealthStatusOnline

	thermo := testDevice("thermo", "Thermostat")
	thermo.Domain = DomainClimate
	thermo.Protocol = ProtocolModbusTCP
	thermo.Type = DeviceTypeThermostat
	thermo.Address = Address{"host": "1.1.1.1", "unit_id": 1}
	thermo.HealthStatus = HealthStatusOffline

	for _, d := range []*Device{light, thermo} {
		if err := registry.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice() error = %v", err)
		}
	}

	stats := registry.GetStats()

	if stats.TotalDevices != 2 {
		t.Errorf("TotalDevices = %d, want 2", stats.TotalDevices)
	}
	if stats.ByDomain[DomainLighting] != 1 {
		t.Errorf("ByDomain[lighting] = %d, want 1", stats.ByDomain[DomainLighting])
	}
	if stats.ByProtocol[ProtocolKNX] != 1 {
		t.Errorf("ByProtocol[knx] = %d, want 1", stats.ByProtocol[ProtocolKNX])
	}
	if stats.ByHealthStatus[HealthStatusOnline] != 1 {
		t.Errorf("ByHealthStatus[online] = %d, want 1", stats.ByHealthStatus[HealthStatusOnline])
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create initial device
	device := testDevice("concurrent", "Concurrent Device")
	if err := registry.CreateDevice(ctx, device); err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)

		// Concurrent reads
		go func() {
			defer wg.Done()
			registry.GetDevice(ctx, "concurrent")
		}()

		// Concurrent state updates
		go func(n int) {
			defer wg.Done()
			registry.SetDeviceState(ctx, "concurrent", State{"count": n})
		}(i)

		// Concurrent health updates
		go func() {
			defer wg.Done()
			registry.SetDeviceHealth(ctx, "concurrent", HealthStatusOnline)
		}()
	}

	wg.Wait()

	// Should still be accessible
	_, err := registry.GetDevice(ctx, "concurrent")
	if err != nil {
		t.Errorf("GetDevice() after concurrent access error = %v", err)
	}
}

func TestRegistry_ListDevices(t *testing.T) { //nolint:gocognit // comprehensive table-driven test
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	t.Run("returns empty list when no devices", func(t *testing.T) {
		devices, err := registry.ListDevices(ctx)
		if err != nil {
			t.Fatalf("ListDevices() error = %v", err)
		}
		if len(devices) != 0 {
			t.Errorf("ListDevices() returned %d devices, want 0", len(devices))
		}
	})

	t.Run("returns all devices sorted by name", func(t *testing.T) {
		// Create devices in non-alphabetical order
		deviceC := testDevice("dev-c", "Charlie Light")
		deviceA := testDevice("dev-a", "Alpha Light")
		deviceB := testDevice("dev-b", "Bravo Light")

		for _, d := range []*Device{deviceC, deviceA, deviceB} {
			if err := registry.CreateDevice(ctx, d); err != nil {
				t.Fatalf("CreateDevice() error = %v", err)
			}
		}

		devices, err := registry.ListDevices(ctx)
		if err != nil {
			t.Fatalf("ListDevices() error = %v", err)
		}
		if len(devices) != 3 {
			t.Errorf("ListDevices() returned %d devices, want 3", len(devices))
		}

		// Should be sorted by name
		if devices[0].Name != "Alpha Light" {
			t.Errorf("devices[0].Name = %q, want %q", devices[0].Name, "Alpha Light")
		}
		if devices[1].Name != "Bravo Light" {
			t.Errorf("devices[1].Name = %q, want %q", devices[1].Name, "Bravo Light")
		}
		if devices[2].Name != "Charlie Light" {
			t.Errorf("devices[2].Name = %q, want %q", devices[2].Name, "Charlie Light")
		}
	})

	t.Run("returns deep copies", func(t *testing.T) {
		devices, _ := registry.ListDevices(ctx)
		if len(devices) == 0 {
			t.Skip("no devices to test")
		}

		// Modify returned device
		devices[0].Name = "Modified"

		// Original should be unchanged
		original, _ := registry.GetDevice(ctx, devices[0].ID)
		if original.Name == "Modified" {
			t.Error("ListDevices() did not return deep copies")
		}
	})
}

func TestRegistry_GetDevicesByArea(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create devices in different areas
	areaA := "area-living"
	areaB := "area-kitchen"

	deviceInA := testDevice("dev-living", "Living Room Light")
	deviceInA.AreaID = &areaA

	deviceInB := testDevice("dev-kitchen", "Kitchen Light")
	deviceInB.AreaID = &areaB

	deviceNoArea := testDevice("dev-no-area", "Orphan Light")
	// deviceNoArea.AreaID is nil

	for _, d := range []*Device{deviceInA, deviceInB, deviceNoArea} {
		if err := registry.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice() error = %v", err)
		}
	}

	t.Run("returns devices in area", func(t *testing.T) {
		devices, err := registry.GetDevicesByArea(ctx, areaA)
		if err != nil {
			t.Fatalf("GetDevicesByArea() error = %v", err)
		}
		if len(devices) != 1 {
			t.Errorf("GetDevicesByArea() returned %d devices, want 1", len(devices))
		}
		if devices[0].ID != "dev-living" {
			t.Errorf("device ID = %q, want %q", devices[0].ID, "dev-living")
		}
	})

	t.Run("returns empty list for unknown area", func(t *testing.T) {
		devices, err := registry.GetDevicesByArea(ctx, "nonexistent-area")
		if err != nil {
			t.Fatalf("GetDevicesByArea() error = %v", err)
		}
		if len(devices) != 0 {
			t.Errorf("GetDevicesByArea() returned %d devices, want 0", len(devices))
		}
	})
}

func TestRegistry_GetDevicesByHealthStatus(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create devices with different health statuses
	onlineDevice := testDevice("dev-online", "Online Light")
	onlineDevice.HealthStatus = HealthStatusOnline

	offlineDevice := testDevice("dev-offline", "Offline Light")
	offlineDevice.HealthStatus = HealthStatusOffline

	unknownDevice := testDevice("dev-unknown", "Unknown Light")
	unknownDevice.HealthStatus = HealthStatusUnknown

	for _, d := range []*Device{onlineDevice, offlineDevice, unknownDevice} {
		if err := registry.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice() error = %v", err)
		}
	}

	t.Run("returns devices with online status", func(t *testing.T) {
		devices, err := registry.GetDevicesByHealthStatus(ctx, HealthStatusOnline)
		if err != nil {
			t.Fatalf("GetDevicesByHealthStatus() error = %v", err)
		}
		if len(devices) != 1 {
			t.Errorf("GetDevicesByHealthStatus(online) returned %d devices, want 1", len(devices))
		}
		if devices[0].ID != "dev-online" {
			t.Errorf("device ID = %q, want %q", devices[0].ID, "dev-online")
		}
	})

	t.Run("returns devices with offline status", func(t *testing.T) {
		devices, err := registry.GetDevicesByHealthStatus(ctx, HealthStatusOffline)
		if err != nil {
			t.Fatalf("GetDevicesByHealthStatus() error = %v", err)
		}
		if len(devices) != 1 {
			t.Errorf("GetDevicesByHealthStatus(offline) returned %d devices, want 1", len(devices))
		}
	})

	t.Run("returns empty list for status with no devices", func(t *testing.T) {
		devices, err := registry.GetDevicesByHealthStatus(ctx, HealthStatusDegraded)
		if err != nil {
			t.Fatalf("GetDevicesByHealthStatus() error = %v", err)
		}
		if len(devices) != 0 {
			t.Errorf("GetDevicesByHealthStatus(degraded) returned %d devices, want 0", len(devices))
		}
	})
}

func TestRegistry_GetDevicesByGateway(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Create devices with different gateways
	gatewayA := "knx-main"
	gatewayB := "dali-main"

	deviceOnGwA := testDevice("dev-knx", "KNX Light")
	deviceOnGwA.GatewayID = &gatewayA

	deviceOnGwB := testDevice("dev-dali", "DALI Light")
	deviceOnGwB.GatewayID = &gatewayB
	deviceOnGwB.Protocol = ProtocolDALI
	deviceOnGwB.Address = Address{"gateway": "gw", "short_address": 1}

	deviceNoGw := testDevice("dev-no-gw", "Local Light")
	// deviceNoGw.GatewayID is nil

	for _, d := range []*Device{deviceOnGwA, deviceOnGwB, deviceNoGw} {
		if err := registry.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice() error = %v", err)
		}
	}

	t.Run("returns devices on gateway", func(t *testing.T) {
		devices, err := registry.GetDevicesByGateway(ctx, gatewayA)
		if err != nil {
			t.Fatalf("GetDevicesByGateway() error = %v", err)
		}
		if len(devices) != 1 {
			t.Errorf("GetDevicesByGateway() returned %d devices, want 1", len(devices))
		}
		if devices[0].ID != "dev-knx" {
			t.Errorf("device ID = %q, want %q", devices[0].ID, "dev-knx")
		}
	})

	t.Run("returns empty list for unknown gateway", func(t *testing.T) {
		devices, err := registry.GetDevicesByGateway(ctx, "nonexistent-gateway")
		if err != nil {
			t.Fatalf("GetDevicesByGateway() error = %v", err)
		}
		if len(devices) != 0 {
			t.Errorf("GetDevicesByGateway() returned %d devices, want 0", len(devices))
		}
	})
}

func TestRegistry_SetLogger(t *testing.T) {
	repo := NewMockRepository()
	registry := NewRegistry(repo)

	// Create a mock logger that tracks calls
	var logged bool
	mockLogger := &testLogger{onInfo: func(string, ...any) { logged = true }}

	registry.SetLogger(mockLogger)

	// Trigger a log (RefreshCache logs Info on success)
	ctx := context.Background()
	registry.RefreshCache(ctx)

	if !logged {
		t.Error("SetLogger() did not set the logger correctly")
	}
}

// testLogger is a minimal Logger implementation for testing SetLogger.
type testLogger struct {
	onInfo func(string, ...any)
}

func (l *testLogger) Debug(string, ...any) {}
func (l *testLogger) Info(msg string, args ...any) {
	if l.onInfo != nil {
		l.onInfo(msg, args...)
	}
}
func (l *testLogger) Warn(string, ...any)  {}
func (l *testLogger) Error(string, ...any) {}
