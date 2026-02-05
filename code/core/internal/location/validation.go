package location

import (
	"fmt"
	"regexp"
	"strings"
)

// Validation constants matching the device package conventions.
const (
	maxNameLength     = 100
	maxSlugLength     = 50
	maxSettingsKeys   = 50
	maxStringValueLen = 1024
	maxNestingDepth   = 10
	slugPattern       = `^[a-z0-9]+(?:-[a-z0-9]+)*$`
)

var slugRegex = regexp.MustCompile(slugPattern)

// ValidateName checks if a location name is valid.
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

// ValidateSettings checks that a Settings map does not exceed size limits.
func ValidateSettings(s Settings) error {
	if s == nil {
		return nil
	}
	if len(s) > maxSettingsKeys {
		return fmt.Errorf("%w: settings exceeds max keys (%d)", ErrInvalidSettings, maxSettingsKeys)
	}
	return validateMapSize(map[string]any(s), "settings", 0)
}

// validateMapSize recursively checks map values against size limits.
func validateMapSize(m map[string]any, fieldName string, depth int) error {
	if depth > maxNestingDepth {
		return fmt.Errorf("%w: %s exceeds maximum nesting depth", ErrInvalidSettings, fieldName)
	}
	for k, v := range m {
		if len(k) > maxStringValueLen {
			return fmt.Errorf("%w: %s key too long", ErrInvalidSettings, fieldName)
		}
		if err := validateValueSize(v, fieldName, depth); err != nil {
			return err
		}
	}
	return nil
}

// validateValueSize checks individual values in a settings map.
func validateValueSize(v any, fieldName string, depth int) error {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		if len(val) > maxStringValueLen {
			return fmt.Errorf("%w: %s string value too long", ErrInvalidSettings, fieldName)
		}
	case map[string]any:
		if len(val) > maxSettingsKeys {
			return fmt.Errorf("%w: %s nested map too large", ErrInvalidSettings, fieldName)
		}
		return validateMapSize(val, fieldName, depth+1)
	case []any:
		if len(val) > maxSettingsKeys {
			return fmt.Errorf("%w: %s array too large", ErrInvalidSettings, fieldName)
		}
		for _, elem := range val {
			if err := validateValueSize(elem, fieldName, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateSite validates a Site before persistence.
func ValidateSite(s *Site) error {
	if err := ValidateName(s.Name); err != nil {
		return err
	}
	if s.Slug != "" {
		if err := ValidateSlug(s.Slug); err != nil {
			return err
		}
	}
	return ValidateSettings(s.Settings)
}

// ValidateArea validates an Area before persistence.
func ValidateArea(a *Area) error {
	if err := ValidateName(a.Name); err != nil {
		return err
	}
	if a.Slug != "" {
		if err := ValidateSlug(a.Slug); err != nil {
			return err
		}
	}
	return nil
}

// ValidateRoom validates a Room before persistence.
func ValidateRoom(r *Room) error {
	if err := ValidateName(r.Name); err != nil {
		return err
	}
	if r.Slug != "" {
		if err := ValidateSlug(r.Slug); err != nil {
			return err
		}
	}
	return ValidateSettings(r.Settings)
}

// ValidateZone validates an InfrastructureZone before persistence.
func ValidateZone(z *InfrastructureZone) error {
	if err := ValidateName(z.Name); err != nil {
		return err
	}
	if z.Slug != "" {
		if err := ValidateSlug(z.Slug); err != nil {
			return err
		}
	}
	if !ValidZoneDomain(string(z.Domain)) {
		return fmt.Errorf("%w: invalid zone domain %q", ErrInvalidSettings, z.Domain)
	}
	return ValidateSettings(z.Settings)
}
