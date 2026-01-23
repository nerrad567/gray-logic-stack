package automation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Validation constants.
const (
	maxNameLength     = 100
	maxSlugLength     = 50
	maxActions        = 100
	minPriority       = 1
	maxPriority       = 100
	defaultPriority   = 50
	maxParameterKeys  = 20
	maxDelayMS        = 300000 // 5 minutes
	maxFadeMS         = 60000  // 1 minute
	maxDescriptionLen = 500
	slugPattern       = `^[a-z0-9]+(?:-[a-z0-9]+)*$`
)

var slugRegex = regexp.MustCompile(slugPattern)

// Pre-computed validation set for O(1) category lookups.
var validCategories map[Category]struct{}

func init() {
	validCategories = make(map[Category]struct{}, len(AllCategories()))
	for _, c := range AllCategories() {
		validCategories[c] = struct{}{}
	}
}

// ValidateScene performs comprehensive validation on a scene.
// Returns an error describing the first validation failure found.
func ValidateScene(s *Scene) error {
	if s == nil {
		return ErrInvalidScene
	}

	// Validate name
	if err := ValidateName(s.Name); err != nil {
		return err
	}

	// Validate slug if provided (empty slug will be generated)
	if s.Slug != "" {
		if err := ValidateSlug(s.Slug); err != nil {
			return err
		}
	}

	// Validate description length
	if s.Description != nil && len(*s.Description) > maxDescriptionLen {
		return fmt.Errorf("%w: description exceeds %d characters", ErrInvalidScene, maxDescriptionLen)
	}

	// Validate priority
	if s.Priority < minPriority || s.Priority > maxPriority {
		return fmt.Errorf("%w: priority must be %d-%d", ErrInvalidScene, minPriority, maxPriority)
	}

	// Validate category if provided
	if s.Category != "" {
		if _, ok := validCategories[s.Category]; !ok {
			return fmt.Errorf("%w: invalid category %q", ErrInvalidScene, s.Category)
		}
	}

	// Validate actions
	if len(s.Actions) == 0 {
		return ErrNoActions
	}
	if len(s.Actions) > maxActions {
		return fmt.Errorf("%w: exceeds maximum of %d actions", ErrInvalidAction, maxActions)
	}

	for i, action := range s.Actions {
		if err := ValidateAction(action); err != nil {
			return fmt.Errorf("action[%d]: %w", i, err)
		}
	}

	return nil
}

// ValidateName checks if a scene name is valid.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
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
		return fmt.Errorf("%w: must be lowercase alphanumeric with hyphens", ErrInvalidSlug)
	}
	return nil
}

// ValidateAction checks if a scene action is valid.
func ValidateAction(action SceneAction) error {
	if action.DeviceID == "" {
		return fmt.Errorf("%w: device_id is required", ErrInvalidAction)
	}
	if action.Command == "" {
		return fmt.Errorf("%w: command is required", ErrInvalidAction)
	}
	if action.DelayMS < 0 || action.DelayMS > maxDelayMS {
		return fmt.Errorf("%w: delay_ms must be 0-%d", ErrInvalidAction, maxDelayMS)
	}
	if action.FadeMS < 0 || action.FadeMS > maxFadeMS {
		return fmt.Errorf("%w: fade_ms must be 0-%d", ErrInvalidAction, maxFadeMS)
	}
	if len(action.Parameters) > maxParameterKeys {
		return fmt.Errorf("%w: parameters exceeds %d keys", ErrInvalidAction, maxParameterKeys)
	}
	return nil
}

// GenerateSlug creates a URL-safe slug from a name.
// It lowercases, replaces spaces/underscores with hyphens, removes
// non-alphanumeric characters, and trims to maxSlugLength.
func GenerateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	// Clean up multiple/leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Truncate to max length
	if len(slug) > maxSlugLength {
		slug = slug[:maxSlugLength]
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// GenerateID creates a new UUID for a scene or execution.
func GenerateID() string {
	return uuid.New().String()
}
