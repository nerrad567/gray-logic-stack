package device

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Logger defines the logging interface used by the Registry.
// This allows different logging implementations to be used.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// noopLogger is a logger that does nothing.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// Registry provides device management with caching and thread safety.
// It wraps a Repository and adds an in-memory cache for fast lookups.
//
// The cache is populated on startup via RefreshCache() and kept in sync
// by cache-invalidating CRUD operations.
//
// All public methods are thread-safe.
type Registry struct {
	repo    Repository
	cache   map[string]*Device // Cached devices by ID
	cacheMu sync.RWMutex       // Protects cache
	logger  Logger
}

// NewRegistry creates a new device registry.
// The repository is used for persistence; the registry adds caching.
func NewRegistry(repo Repository) *Registry {
	return &Registry{
		repo:   repo,
		cache:  make(map[string]*Device),
		logger: noopLogger{},
	}
}

// SetLogger sets the logger for the registry.
func (r *Registry) SetLogger(logger Logger) {
	r.logger = logger
}

// RefreshCache reloads all devices from the repository into the cache.
// This should be called on application startup.
func (r *Registry) RefreshCache(ctx context.Context) error {
	devices, err := r.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("loading devices: %w", err)
	}

	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	// Clear and rebuild cache with deep copies
	r.cache = make(map[string]*Device, len(devices))
	for i := range devices {
		d := devices[i]
		r.cache[d.ID] = d.DeepCopy()
	}

	r.logger.Info("device cache refreshed", "count", len(devices))
	return nil
}

// GetDevice retrieves a device by ID.
// Returns ErrDeviceNotFound if the device does not exist.
// The returned device is a deep copy; callers can safely modify it.
func (r *Registry) GetDevice(ctx context.Context, id string) (*Device, error) {
	// Try cache first
	r.cacheMu.RLock()
	cached, ok := r.cache[id]
	r.cacheMu.RUnlock()

	if ok {
		// Return a deep copy to prevent external mutation of cache
		return cached.DeepCopy(), nil
	}

	// Fall back to repository (might be a new device not yet cached)
	device, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache for future lookups (store a deep copy)
	r.cacheMu.Lock()
	r.cache[id] = device.DeepCopy()
	r.cacheMu.Unlock()

	return device, nil
}

// ListDevices retrieves all devices.
// The returned devices are deep copies; callers can safely modify them.
func (r *Registry) ListDevices(ctx context.Context) ([]Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	// Return from cache if populated
	if len(r.cache) > 0 {
		devices := make([]Device, 0, len(r.cache))
		for _, d := range r.cache {
			// Deep copy to prevent external mutation of cache
			devices = append(devices, *d.DeepCopy())
		}
		sortDevicesByName(devices)
		return devices, nil
	}

	// Fall back to repository
	return r.repo.List(ctx)
}

// GetDevicesByRoom retrieves all devices in a specific room.
// The returned devices are deep copies; callers can safely modify them.
func (r *Registry) GetDevicesByRoom(ctx context.Context, roomID string) ([]Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	// Filter from cache if populated
	if len(r.cache) > 0 {
		var devices []Device
		for _, d := range r.cache {
			if d.RoomID != nil && *d.RoomID == roomID {
				// Deep copy to prevent external mutation of cache
				devices = append(devices, *d.DeepCopy())
			}
		}
		sortDevicesByName(devices)
		return devices, nil
	}

	return r.repo.ListByRoom(ctx, roomID)
}

// GetDevicesByArea retrieves all devices in a specific area.
// The returned devices are deep copies; callers can safely modify them.
func (r *Registry) GetDevicesByArea(ctx context.Context, areaID string) ([]Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	// Filter from cache if populated
	if len(r.cache) > 0 {
		var devices []Device
		for _, d := range r.cache {
			if d.AreaID != nil && *d.AreaID == areaID {
				// Deep copy to prevent external mutation of cache
				devices = append(devices, *d.DeepCopy())
			}
		}
		return devices, nil
	}

	return r.repo.ListByArea(ctx, areaID)
}

// GetDevicesByDomain retrieves all devices in a specific domain.
// The returned devices are deep copies; callers can safely modify them.
func (r *Registry) GetDevicesByDomain(ctx context.Context, domain Domain) ([]Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	// Filter from cache if populated
	if len(r.cache) > 0 {
		var devices []Device
		for _, d := range r.cache {
			if d.Domain == domain {
				// Deep copy to prevent external mutation of cache
				devices = append(devices, *d.DeepCopy())
			}
		}
		return devices, nil
	}

	return r.repo.ListByDomain(ctx, domain)
}

// GetDevicesByProtocol retrieves all devices using a specific protocol.
// The returned devices are deep copies; callers can safely modify them.
func (r *Registry) GetDevicesByProtocol(ctx context.Context, protocol Protocol) ([]Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	// Filter from cache if populated
	if len(r.cache) > 0 {
		var devices []Device
		for _, d := range r.cache {
			if d.Protocol == protocol {
				// Deep copy to prevent external mutation of cache
				devices = append(devices, *d.DeepCopy())
			}
		}
		return devices, nil
	}

	return r.repo.ListByProtocol(ctx, protocol)
}

// sortDevicesByName sorts a device slice by name for deterministic ordering.
func sortDevicesByName(devices []Device) {
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Name < devices[j].Name
	})
}

// CreateDevice creates a new device.
// It validates the device, generates ID and slug if needed, and persists it.
func (r *Registry) CreateDevice(ctx context.Context, device *Device) error {
	// Generate ID if not provided
	if device.ID == "" {
		device.ID = GenerateID()
	}

	// Generate slug if not provided
	if device.Slug == "" {
		device.Slug = GenerateSlug(device.Name)
	}

	// Validate
	if err := ValidateDevice(device); err != nil {
		return err
	}

	// Persist
	if err := r.repo.Create(ctx, device); err != nil {
		return err
	}

	// Update cache (store a deep copy to prevent external modification)
	r.cacheMu.Lock()
	r.cache[device.ID] = device.DeepCopy()
	r.cacheMu.Unlock()

	r.logger.Info("device created", "id", device.ID, "name", device.Name)
	return nil
}

// UpdateDevice updates an existing device.
// It validates the device and persists the changes.
func (r *Registry) UpdateDevice(ctx context.Context, device *Device) error {
	// Regenerate slug if name changed and slug wasn't explicitly set
	existing, err := r.GetDevice(ctx, device.ID)
	if err != nil {
		return err
	}
	if device.Name != existing.Name && device.Slug == existing.Slug {
		device.Slug = GenerateSlug(device.Name)
	}

	// Validate
	if err := ValidateDevice(device); err != nil {
		return err
	}

	// Persist
	if err := r.repo.Update(ctx, device); err != nil {
		return err
	}

	// Update cache (store a deep copy to prevent external modification)
	r.cacheMu.Lock()
	r.cache[device.ID] = device.DeepCopy()
	r.cacheMu.Unlock()

	r.logger.Info("device updated", "id", device.ID, "name", device.Name)
	return nil
}

// DeleteDevice removes a device.
func (r *Registry) DeleteDevice(ctx context.Context, id string) error {
	if err := r.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Update cache
	r.cacheMu.Lock()
	delete(r.cache, id)
	r.cacheMu.Unlock()

	r.logger.Info("device deleted", "id", id)
	return nil
}

// SetDeviceState merges the given state fields into the device's existing state.
// This is optimised for frequent updates from protocol bridges.
func (r *Registry) SetDeviceState(ctx context.Context, id string, state State) error {
	if err := r.repo.UpdateState(ctx, id, state); err != nil {
		return err
	}

	// Update cache by merging new state into existing state
	r.cacheMu.Lock()
	if cached, ok := r.cache[id]; ok {
		// Create a deep copy and merge new state into existing
		updated := cached.DeepCopy()
		if updated.State == nil {
			updated.State = make(State)
		}
		// Merge: copy new keys into existing state
		for k, v := range state {
			updated.State[k] = v
		}
		now := time.Now().UTC()
		updated.StateUpdatedAt = &now
		r.cache[id] = updated
	}
	r.cacheMu.Unlock()

	r.logger.Debug("device state updated", "id", id)
	return nil
}

// SetDeviceHealth updates the health status of a device.
func (r *Registry) SetDeviceHealth(ctx context.Context, id string, status HealthStatus) error {
	now := time.Now().UTC()
	if err := r.repo.UpdateHealth(ctx, id, status, now); err != nil {
		return err
	}

	// Update cache using deep copy to prevent race conditions
	r.cacheMu.Lock()
	if cached, ok := r.cache[id]; ok {
		// Create a deep copy with updated health (atomic replacement)
		updated := cached.DeepCopy()
		updated.HealthStatus = status
		updated.HealthLastSeen = &now
		r.cache[id] = updated
	}
	r.cacheMu.Unlock()

	r.logger.Debug("device health updated", "id", id, "status", status)
	return nil
}

// GetDeviceCount returns the number of cached devices.
func (r *Registry) GetDeviceCount() int {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	return len(r.cache)
}

// GetDevicesByHealthStatus retrieves all devices with a specific health status.
// The returned devices are deep copies; callers can safely modify them.
func (r *Registry) GetDevicesByHealthStatus(ctx context.Context, status HealthStatus) ([]Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	var devices []Device
	for _, d := range r.cache {
		if d.HealthStatus == status {
			// Deep copy to prevent external mutation of cache
			devices = append(devices, *d.DeepCopy())
		}
	}
	return devices, nil
}

// GetDeviceBySlug retrieves a device by its URL-safe slug.
// The returned device is a deep copy; callers can safely modify it.
func (r *Registry) GetDeviceBySlug(ctx context.Context, slug string) (*Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	for _, d := range r.cache {
		if d.Slug == slug {
			// Return a deep copy to prevent external mutation of cache
			return d.DeepCopy(), nil
		}
	}
	return nil, ErrDeviceNotFound
}

// GetDevicesByGateway retrieves all devices connected through a specific gateway.
// The returned devices are deep copies; callers can safely modify them.
func (r *Registry) GetDevicesByGateway(ctx context.Context, gatewayID string) ([]Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	var devices []Device
	for _, d := range r.cache {
		if d.GatewayID != nil && *d.GatewayID == gatewayID {
			// Deep copy to prevent external mutation of cache
			devices = append(devices, *d.DeepCopy())
		}
	}
	return devices, nil
}

// GetDevicesByCapability retrieves all devices that have a specific capability.
// The returned devices are deep copies; callers can safely modify them.
func (r *Registry) GetDevicesByCapability(ctx context.Context, capability Capability) ([]Device, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	var devices []Device
	for _, d := range r.cache {
		for _, cap := range d.Capabilities {
			if cap == capability {
				// Deep copy to prevent external mutation of cache
				devices = append(devices, *d.DeepCopy())
				break
			}
		}
	}
	return devices, nil
}

// Stats returns registry statistics for monitoring.
type Stats struct {
	TotalDevices   int
	ByDomain       map[Domain]int
	ByProtocol     map[Protocol]int
	ByHealthStatus map[HealthStatus]int
}

// GetStats returns current registry statistics.
func (r *Registry) GetStats() Stats {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	stats := Stats{
		TotalDevices:   len(r.cache),
		ByDomain:       make(map[Domain]int),
		ByProtocol:     make(map[Protocol]int),
		ByHealthStatus: make(map[HealthStatus]int),
	}

	for _, d := range r.cache {
		stats.ByDomain[d.Domain]++
		stats.ByProtocol[d.Protocol]++
		stats.ByHealthStatus[d.HealthStatus]++
	}

	return stats
}

// MigrateKNXAddressFormat converts KNX devices from the old flat address format
// to the new structured "functions" format. This is a one-time data migration
// that runs on startup. Devices already in the new format are skipped.
//
// Old format: {"group_address": "1/0/1", "switch": "1/0/1", "switch_status": "1/0/2"}
// New format: {"functions": {"switch": {"ga": "1/0/1", "dpt": "1.001", "flags": ["write"]}, ...}}
func (r *Registry) MigrateKNXAddressFormat(ctx context.Context) error {
	devices, err := r.GetDevicesByProtocol(ctx, ProtocolKNX)
	if err != nil {
		return fmt.Errorf("loading KNX devices for migration: %w", err)
	}

	migrated := 0
	for _, dev := range devices {
		// Skip devices already in the new format
		if _, hasFunctions := dev.Address["functions"]; hasFunctions {
			continue
		}

		// Convert old flat format to new structured format
		functions := make(map[string]any)
		for key, val := range dev.Address {
			// Skip metadata keys
			if key == "group_address" || key == "individual_address" || key == "application_program" || key == "" {
				continue
			}
			ga, ok := val.(string)
			if !ok || ga == "" {
				continue
			}

			functions[key] = map[string]any{
				"ga":    ga,
				"dpt":   migrateInferDPT(key),
				"flags": migrateInferFlags(key),
			}
		}

		if len(functions) == 0 {
			continue
		}

		// Build new address map, preserving metadata keys
		newAddr := make(Address)
		newAddr["functions"] = functions
		if v, ok := dev.Address["individual_address"]; ok {
			newAddr["individual_address"] = v
		}
		if v, ok := dev.Address["application_program"]; ok {
			newAddr["application_program"] = v
		}

		dev.Address = newAddr
		if err := r.UpdateDevice(ctx, &dev); err != nil {
			r.logger.Error("migration: failed to update device", "id", dev.ID, "error", err)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		r.logger.Info("migrated KNX devices to structured address format", "count", migrated)
	}
	return nil
}

// migrateInferDPT is a best-effort DPT inference used only during one-time
// data migration. After migration, DPTs come from the registry, not inference.
func migrateInferDPT(fn string) string {
	fn = strings.ToLower(fn)

	switch {
	case strings.Contains(fn, "temperature"), strings.Contains(fn, "setpoint"):
		return "9.001"
	case strings.Contains(fn, "humidity"):
		return "9.007"
	case strings.Contains(fn, "lux"):
		return "9.004"
	case strings.Contains(fn, "brightness"), strings.Contains(fn, "level"),
		strings.Contains(fn, "position"), strings.Contains(fn, "slat"),
		strings.Contains(fn, "valve"), strings.Contains(fn, "heating_output"):
		return "5.001"
	case strings.Contains(fn, "switch"), strings.Contains(fn, "on"),
		strings.Contains(fn, "button"), strings.Contains(fn, "led"):
		return "1.001"
	case strings.Contains(fn, "presence"), strings.Contains(fn, "motion"):
		return "1.018"
	case strings.Contains(fn, "move"):
		return "1.008"
	case strings.Contains(fn, "stop"):
		return "1.007"
	default:
		return ""
	}
}

// migrateInferFlags is a best-effort flag inference used only during one-time
// data migration. After migration, flags come from the registry, not inference.
func migrateInferFlags(fn string) []string {
	fn = strings.ToLower(fn)

	if strings.Contains(fn, "status") || strings.Contains(fn, "feedback") {
		return []string{"read", "transmit"}
	}
	if strings.Contains(fn, "temperature") || strings.Contains(fn, "humidity") ||
		strings.Contains(fn, "lux") || strings.Contains(fn, "presence") ||
		strings.Contains(fn, "co2") || strings.Contains(fn, "wind") ||
		strings.Contains(fn, "rain") {
		return []string{"read", "transmit"}
	}
	if strings.Contains(fn, "switch") || strings.Contains(fn, "valve") ||
		strings.Contains(fn, "brightness") || strings.Contains(fn, "position") ||
		strings.Contains(fn, "slat") || strings.Contains(fn, "move") ||
		strings.Contains(fn, "stop") || strings.Contains(fn, "scene") {
		return []string{"write", "transmit"}
	}
	return []string{"write"}
}
