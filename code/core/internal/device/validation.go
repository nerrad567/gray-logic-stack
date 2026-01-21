package device

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Validation constants.
const (
	maxNameLength = 100
	maxSlugLength = 50
	slugPattern   = `^[a-z0-9]+(?:-[a-z0-9]+)*$`

	// Size limits for JSON fields to prevent DoS via memory exhaustion.
	// These are generous limits for building automation use cases.
	maxAddressKeys     = 20   // Max keys in address map
	maxConfigKeys      = 50   // Max keys in config map
	maxStateKeys       = 100  // Max keys in state map (devices can have many readings)
	maxCapabilities    = 50   // Max capabilities per device
	maxStringValueLen  = 1024 // Max length for string values in JSON maps
	maxPHMBaselineKeys = 100  // Max keys in PHM baseline
)

var slugRegex = regexp.MustCompile(slugPattern)

// Pre-computed validation sets for O(1) lookups instead of O(n) linear search.
var (
	validDomains      map[Domain]struct{}
	validProtocols    map[Protocol]struct{}
	validDeviceTypes  map[DeviceType]struct{}
	validCapabilities map[Capability]struct{}
	validHealthStatus map[HealthStatus]struct{}
)

func init() {
	// Build validation sets once at startup
	validDomains = make(map[Domain]struct{}, len(AllDomains()))
	for _, d := range AllDomains() {
		validDomains[d] = struct{}{}
	}

	validProtocols = make(map[Protocol]struct{}, len(AllProtocols()))
	for _, p := range AllProtocols() {
		validProtocols[p] = struct{}{}
	}

	validDeviceTypes = make(map[DeviceType]struct{}, len(AllDeviceTypes()))
	for _, t := range AllDeviceTypes() {
		validDeviceTypes[t] = struct{}{}
	}

	validCapabilities = make(map[Capability]struct{}, len(AllCapabilities()))
	for _, c := range AllCapabilities() {
		validCapabilities[c] = struct{}{}
	}

	validHealthStatus = make(map[HealthStatus]struct{}, len(AllHealthStatuses()))
	for _, s := range AllHealthStatuses() {
		validHealthStatus[s] = struct{}{}
	}
}

// ValidateDevice performs comprehensive validation on a device.
// Returns an error describing the first validation failure found.
// Includes size limits to prevent DoS via memory exhaustion.
func ValidateDevice(d *Device) error {
	if d == nil {
		return ErrInvalidDevice
	}

	// Validate name
	if err := ValidateName(d.Name); err != nil {
		return err
	}

	// Validate slug if provided (empty slug will be generated)
	if d.Slug != "" {
		if err := ValidateSlug(d.Slug); err != nil {
			return err
		}
	}

	// Validate type
	if err := ValidateDeviceType(d.Type); err != nil {
		return err
	}

	// Validate domain
	if err := ValidateDomain(d.Domain); err != nil {
		return err
	}

	// Validate protocol
	if err := ValidateProtocol(d.Protocol); err != nil {
		return err
	}

	// Validate address is not empty and within size limits
	if len(d.Address) == 0 {
		return fmt.Errorf("%w: address is required", ErrInvalidAddress)
	}
	if len(d.Address) > maxAddressKeys {
		return fmt.Errorf("%w: address exceeds max keys (%d)", ErrInvalidAddress, maxAddressKeys)
	}
	if err := validateMapSize(d.Address, "address"); err != nil {
		return err
	}

	// Validate protocol-specific address
	if err := ValidateAddress(d.Protocol, d.Address); err != nil {
		return err
	}

	// Validate config size if provided
	if len(d.Config) > maxConfigKeys {
		return fmt.Errorf("%w: config exceeds max keys (%d)", ErrInvalidDevice, maxConfigKeys)
	}
	if err := validateMapSize(d.Config, "config"); err != nil {
		return err
	}

	// Validate state size if provided
	if len(d.State) > maxStateKeys {
		return fmt.Errorf("%w: state exceeds max keys (%d)", ErrInvalidState, maxStateKeys)
	}
	if err := validateMapSize(d.State, "state"); err != nil {
		return err
	}

	// Validate capabilities if provided
	if len(d.Capabilities) > 0 {
		if err := ValidateCapabilities(d.Capabilities); err != nil {
			return err
		}
	}

	// Validate health status if set
	if d.HealthStatus != "" {
		if err := ValidateHealthStatus(d.HealthStatus); err != nil {
			return err
		}
	}

	// Validate PHM baseline size if provided
	if d.PHMBaseline != nil {
		if len(*d.PHMBaseline) > maxPHMBaselineKeys {
			return fmt.Errorf("%w: PHM baseline exceeds max keys (%d)", ErrInvalidDevice, maxPHMBaselineKeys)
		}
		if err := validateMapSize(map[string]any(*d.PHMBaseline), "phm_baseline"); err != nil {
			return err
		}
	}

	return nil
}

// validateMapSize checks that all values in a map don't exceed size limits.
// This recursively validates nested maps and slices to prevent DoS attacks.
func validateMapSize(m map[string]any, fieldName string) error {
	return validateMapSizeRecursive(m, fieldName, 0)
}

// maxNestingDepth prevents stack overflow from deeply nested structures.
const maxNestingDepth = 10

// validateMapSizeRecursive recursively validates map values with depth tracking.
func validateMapSizeRecursive(m map[string]any, fieldName string, depth int) error {
	if depth > maxNestingDepth {
		return fmt.Errorf("%w: %s exceeds maximum nesting depth", ErrInvalidDevice, fieldName)
	}

	for k, v := range m {
		// Check key length
		if len(k) > maxStringValueLen {
			return fmt.Errorf("%w: %s key too long", ErrInvalidDevice, fieldName)
		}
		// Recursively validate values
		if err := validateValueSize(v, fieldName, depth); err != nil {
			return err
		}
	}
	return nil
}

// validateValueSize recursively validates a value's size.
func validateValueSize(v any, fieldName string, depth int) error {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case string:
		if len(val) > maxStringValueLen {
			return fmt.Errorf("%w: %s string value too long", ErrInvalidDevice, fieldName)
		}
	case map[string]any:
		if len(val) > maxConfigKeys { // Use config limit for nested maps
			return fmt.Errorf("%w: %s nested map too large", ErrInvalidDevice, fieldName)
		}
		return validateMapSizeRecursive(val, fieldName, depth+1)
	case []any:
		if len(val) > maxCapabilities { // Reasonable limit for arrays
			return fmt.Errorf("%w: %s array too large", ErrInvalidDevice, fieldName)
		}
		for _, elem := range val {
			if err := validateValueSize(elem, fieldName, depth+1); err != nil {
				return err
			}
		}
	}
	// Primitives (bool, int, float64, etc.) are safe
	return nil
}

// ValidateName checks if a device name is valid.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidName)
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("%w: name exceeds %d characters", ErrInvalidName, maxNameLength)
	}
	return nil
}

// ValidateSlug checks if a slug format is valid.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("%w: slug cannot be empty", ErrInvalidSlug)
	}
	if len(slug) > maxSlugLength {
		return fmt.Errorf("%w: slug exceeds %d characters", ErrInvalidSlug, maxSlugLength)
	}
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("%w: slug must be lowercase alphanumeric with hyphens", ErrInvalidSlug)
	}
	return nil
}

// ValidateDomain checks if a domain is valid.
// Uses O(1) map lookup for efficiency.
func ValidateDomain(domain Domain) error {
	if _, ok := validDomains[domain]; ok {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidDomain, domain)
}

// ValidateProtocol checks if a protocol is valid.
// Uses O(1) map lookup for efficiency.
func ValidateProtocol(protocol Protocol) error {
	if _, ok := validProtocols[protocol]; ok {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidProtocol, protocol)
}

// ValidateDeviceType checks if a device type is valid.
// Uses O(1) map lookup for efficiency.
func ValidateDeviceType(deviceType DeviceType) error {
	if _, ok := validDeviceTypes[deviceType]; ok {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidDeviceType, deviceType)
}

// ValidateCapabilities checks if all capabilities are valid.
// Uses O(1) map lookup for each capability.
func ValidateCapabilities(caps []Capability) error {
	if len(caps) > maxCapabilities {
		return fmt.Errorf("%w: too many capabilities (max %d)", ErrInvalidCapability, maxCapabilities)
	}
	for _, cap := range caps {
		if _, ok := validCapabilities[cap]; !ok {
			return fmt.Errorf("%w: %q", ErrInvalidCapability, cap)
		}
	}
	return nil
}

// ValidateHealthStatus checks if a health status is valid.
// Uses O(1) map lookup for efficiency.
func ValidateHealthStatus(status HealthStatus) error {
	if _, ok := validHealthStatus[status]; ok {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidState, status)
}

// ValidateAddress validates protocol-specific address configuration.
func ValidateAddress(protocol Protocol, addr Address) error {
	switch protocol {
	case ProtocolKNX:
		return validateKNXAddress(addr)
	case ProtocolDALI:
		return validateDALIAddress(addr)
	case ProtocolModbusRTU, ProtocolModbusTCP:
		return validateModbusAddress(addr)
	case ProtocolMQTT:
		return validateMQTTAddress(addr)
	case ProtocolBACnetIP, ProtocolBACnetMSTP,
		ProtocolHTTP, ProtocolSIP, ProtocolRTSP,
		ProtocolONVIF, ProtocolOCPP,
		ProtocolRS232, ProtocolRS485:
		// For these protocols, just ensure address is not empty
		// Protocol-specific validation can be added as bridges are implemented
		if len(addr) == 0 {
			return fmt.Errorf("%w: address cannot be empty", ErrInvalidAddress)
		}
		return nil
	}
	// Unreachable if all Protocol constants are handled above
	return fmt.Errorf("%w: unknown protocol %q", ErrInvalidProtocol, protocol)
}

// validateKNXAddress validates a KNX address configuration.
func validateKNXAddress(addr Address) error {
	// KNX requires at least a group_address
	ga, ok := addr["group_address"]
	if !ok {
		return fmt.Errorf("%w: KNX address requires group_address", ErrInvalidAddress)
	}
	gaStr, ok := ga.(string)
	if !ok || gaStr == "" {
		return fmt.Errorf("%w: KNX group_address must be a non-empty string", ErrInvalidAddress)
	}
	// Basic format check (should be like "1/2/3" for 3-level or "1/234" for 2-level)
	// Detailed validation is done by the KNX bridge
	return nil
}

// validateDALIAddress validates a DALI address configuration.
func validateDALIAddress(addr Address) error {
	// DALI requires a gateway and either short_address or group
	if _, ok := addr["gateway"]; !ok {
		return fmt.Errorf("%w: DALI address requires gateway", ErrInvalidAddress)
	}
	_, hasShort := addr["short_address"]
	_, hasGroup := addr["group"]
	if !hasShort && !hasGroup {
		return fmt.Errorf("%w: DALI address requires short_address or group", ErrInvalidAddress)
	}
	return nil
}

// validateModbusAddress validates a Modbus address configuration.
func validateModbusAddress(addr Address) error {
	// Modbus requires host/port (TCP) or device (RTU) and unit_id
	_, hasHost := addr["host"]
	_, hasDevice := addr["device"]
	if !hasHost && !hasDevice {
		return fmt.Errorf("%w: Modbus address requires host or device", ErrInvalidAddress)
	}
	if _, ok := addr["unit_id"]; !ok {
		return fmt.Errorf("%w: Modbus address requires unit_id", ErrInvalidAddress)
	}
	return nil
}

// validateMQTTAddress validates an MQTT address configuration.
func validateMQTTAddress(addr Address) error {
	// MQTT requires a topic
	if _, ok := addr["topic"]; !ok {
		return fmt.Errorf("%w: MQTT address requires topic", ErrInvalidAddress)
	}
	return nil
}

// GenerateSlug creates a URL-safe slug from a name.
func GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	// Remove leading/trailing hyphens and collapse multiple hyphens
	slug = strings.Trim(slug, "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Truncate if too long
	if len(slug) > maxSlugLength {
		slug = slug[:maxSlugLength]
		// Don't end with a hyphen
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// GenerateID creates a new UUID for a device.
func GenerateID() string {
	return uuid.New().String()
}
