package device

import (
	"context"
	"errors"
	"fmt"
)

// ResolveGroup expands a DeviceGroup into a concrete list of devices.
//
// It combines explicit members (static or hybrid) with dynamic filter results,
// then applies exclude_tags to remove exceptions. Results are deduplicated by
// device ID and sorted by name.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - group: Group definition to resolve
//   - registry: Device registry used for cached device access
//   - tagRepo: Tag repository used for fallback lookups when cache tags are missing
//   - groupRepo: Group repository used for explicit member lookups
//
// Returns:
//   - []Device: Resolved device list ordered by name
//   - error: nil on success, otherwise the underlying error
//
// Security: Uses cached device data and parameterised repository calls only.
// Example:
//
//	devices, err := device.ResolveGroup(ctx, group, registry, tagRepo, groupRepo)
func ResolveGroup(
	ctx context.Context,
	group *DeviceGroup,
	registry *Registry,
	tagRepo TagRepository,
	groupRepo GroupRepository,
) ([]Device, error) {
	if group == nil {
		return nil, fmt.Errorf("group is required")
	}
	if registry == nil {
		return nil, fmt.Errorf("registry is required")
	}

	explicitDevices, err := resolveExplicitMembers(ctx, group, registry, groupRepo)
	if err != nil {
		return nil, err
	}

	dynamicDevices, err := resolveDynamicMembers(ctx, group, registry, tagRepo)
	if err != nil {
		return nil, err
	}

	merged := mergeDevices(explicitDevices, dynamicDevices)
	filtered, err := applyExcludeTags(ctx, merged, group.FilterRules, tagRepo)
	if err != nil {
		return nil, err
	}

	sortDevicesByName(filtered)
	if len(filtered) == 0 {
		return []Device{}, nil
	}

	return filtered, nil
}

// resolveExplicitMembers resolves static or hybrid group members.
func resolveExplicitMembers(ctx context.Context, group *DeviceGroup, registry *Registry, groupRepo GroupRepository) ([]Device, error) {
	if group.Type != GroupTypeStatic && group.Type != GroupTypeHybrid {
		return []Device{}, nil
	}
	if groupRepo == nil {
		return nil, fmt.Errorf("group repository is required for explicit members")
	}

	memberIDs, err := groupRepo.GetMemberDeviceIDs(ctx, group.ID)
	if err != nil {
		return nil, fmt.Errorf("loading group members: %w", err)
	}

	explicitDevices := make([]Device, 0, len(memberIDs))
	for _, id := range memberIDs {
		device, err := registry.GetDevice(ctx, id)
		if err != nil {
			if errors.Is(err, ErrDeviceNotFound) {
				continue
			}
			return nil, fmt.Errorf("loading device %s: %w", id, err)
		}
		explicitDevices = append(explicitDevices, *device)
	}

	return explicitDevices, nil
}

// resolveDynamicMembers resolves dynamic or hybrid group members.
func resolveDynamicMembers(ctx context.Context, group *DeviceGroup, registry *Registry, tagRepo TagRepository) ([]Device, error) {
	if group.Type != GroupTypeDynamic && group.Type != GroupTypeHybrid {
		return []Device{}, nil
	}

	baseDevices, err := resolveDynamicBase(ctx, registry, group.FilterRules)
	if err != nil {
		return nil, err
	}

	return applyDynamicFilters(ctx, baseDevices, group.FilterRules, tagRepo)
}

// resolveDynamicBase returns devices for the requested scope.
func resolveDynamicBase(ctx context.Context, registry *Registry, rules *FilterRules) ([]Device, error) {
	if rules == nil || rules.ScopeType == "" || rules.ScopeType == "site" {
		return registry.ListDevices(ctx)
	}

	switch rules.ScopeType {
	case "area":
		if rules.ScopeID == "" {
			return []Device{}, nil
		}
		return registry.GetDevicesByArea(ctx, rules.ScopeID)
	case "room":
		if rules.ScopeID == "" {
			return []Device{}, nil
		}
		return registry.GetDevicesByRoom(ctx, rules.ScopeID)
	default:
		return nil, fmt.Errorf("unknown scope_type: %s", rules.ScopeType)
	}
}

// applyDynamicFilters applies domain, type, capability, and tag filters.
func applyDynamicFilters(ctx context.Context, devices []Device, rules *FilterRules, tagRepo TagRepository) ([]Device, error) {
	if rules == nil {
		return devices, nil
	}

	filtered := devices
	filtered = filterByDomains(filtered, rules.Domains)
	filtered = filterByDeviceTypes(filtered, rules.DeviceTypes)
	filtered = filterByCapabilities(filtered, rules.Capabilities)

	var err error
	filtered, err = filterByTags(ctx, filtered, rules.Tags, tagRepo)
	if err != nil {
		return nil, err
	}

	return filtered, nil
}

// filterByDomains keeps devices in the allowed domain set.
func filterByDomains(devices []Device, domains []string) []Device {
	if len(domains) == 0 {
		return devices
	}

	allowed := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		if d == "" {
			continue
		}
		allowed[stringsToLower(d)] = struct{}{}
	}

	var filtered []Device
	for _, device := range devices {
		if _, ok := allowed[stringsToLower(string(device.Domain))]; ok {
			filtered = append(filtered, device)
		}
	}
	return filtered
}

// filterByDeviceTypes keeps devices in the allowed type set.
func filterByDeviceTypes(devices []Device, types []string) []Device {
	if len(types) == 0 {
		return devices
	}

	allowed := make(map[string]struct{}, len(types))
	for _, t := range types {
		if t == "" {
			continue
		}
		allowed[stringsToLower(t)] = struct{}{}
	}

	var filtered []Device
	for _, device := range devices {
		if _, ok := allowed[stringsToLower(string(device.Type))]; ok {
			filtered = append(filtered, device)
		}
	}
	return filtered
}

// filterByCapabilities keeps devices that have all required capabilities.
func filterByCapabilities(devices []Device, required []string) []Device {
	if len(required) == 0 {
		return devices
	}

	req := make([]string, 0, len(required))
	for _, cap := range required {
		if cap != "" {
			req = append(req, stringsToLower(cap))
		}
	}
	if len(req) == 0 {
		return devices
	}

	var filtered []Device
	for _, device := range devices {
		if deviceHasCapabilities(device, req) {
			filtered = append(filtered, device)
		}
	}
	return filtered
}

// filterByTags keeps devices that match any of the provided tags.
func filterByTags(ctx context.Context, devices []Device, tags []string, tagRepo TagRepository) ([]Device, error) {
	if len(tags) == 0 {
		return devices, nil
	}

	wanted := normaliseTags(tags)
	if len(wanted) == 0 {
		return devices, nil
	}

	var filtered []Device
	for _, device := range devices {
		match, err := deviceHasAnyTag(ctx, device, wanted, tagRepo)
		if err != nil {
			return nil, err
		}
		if match {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

// applyExcludeTags removes devices that match any excluded tag.
func applyExcludeTags(ctx context.Context, devices []Device, rules *FilterRules, tagRepo TagRepository) ([]Device, error) {
	if rules == nil || len(rules.ExcludeTags) == 0 {
		return devices, nil
	}

	excluded := normaliseTags(rules.ExcludeTags)
	if len(excluded) == 0 {
		return devices, nil
	}

	var filtered []Device
	for _, device := range devices {
		match, err := deviceHasAnyTag(ctx, device, excluded, tagRepo)
		if err != nil {
			return nil, err
		}
		if !match {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

// deviceHasCapabilities checks if the device has all required capabilities.
func deviceHasCapabilities(device Device, required []string) bool {
	if len(required) == 0 {
		return true
	}

	available := make(map[string]struct{}, len(device.Capabilities))
	for _, cap := range device.Capabilities {
		available[stringsToLower(string(cap))] = struct{}{}
	}

	for _, cap := range required {
		if _, ok := available[stringsToLower(cap)]; !ok {
			return false
		}
	}
	return true
}

// deviceHasAnyTag checks if the device has any of the provided tags.
func deviceHasAnyTag(ctx context.Context, device Device, tags []string, tagRepo TagRepository) (bool, error) {
	deviceTags, err := resolveDeviceTags(ctx, device, tagRepo)
	if err != nil {
		return false, err
	}

	for _, tag := range tags {
		if _, ok := deviceTags[tag]; ok {
			return true, nil
		}
	}

	return false, nil
}

// resolveDeviceTags returns a normalised tag set for a device.
func resolveDeviceTags(ctx context.Context, device Device, tagRepo TagRepository) (map[string]struct{}, error) {
	tags := device.Tags
	if len(tags) == 0 && tagRepo != nil {
		fetched, err := tagRepo.GetTags(ctx, device.ID)
		if err != nil {
			return nil, fmt.Errorf("loading tags for device %s: %w", device.ID, err)
		}
		tags = fetched
	}

	set := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		normalised := normaliseTag(tag)
		if normalised == "" {
			continue
		}
		set[normalised] = struct{}{}
	}

	return set, nil
}

// mergeDevices combines device slices and removes duplicates by ID.
func mergeDevices(a []Device, b []Device) []Device {
	merged := make(map[string]Device)
	for _, device := range a {
		merged[device.ID] = device
	}
	for _, device := range b {
		merged[device.ID] = device
	}

	result := make([]Device, 0, len(merged))
	for _, device := range merged {
		result = append(result, device)
	}

	return result
}

// stringsToLower trims and lowercases a string for comparisons.
func stringsToLower(value string) string {
	return normaliseTag(value)
}
