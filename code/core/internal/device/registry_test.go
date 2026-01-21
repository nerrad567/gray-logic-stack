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
			Name:         "New Device",
			Type:         DeviceTypeLightDimmer,
			Domain:       DomainLighting,
			Protocol:     ProtocolKNX,
			Address:      Address{"group_address": "1/2/3"},
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
