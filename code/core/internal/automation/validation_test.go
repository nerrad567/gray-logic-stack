package automation

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateScene(t *testing.T) {
	validAction := SceneAction{
		DeviceID:        "light-living-main",
		Command:         "set",
		Parameters:      map[string]any{"on": true},
		ContinueOnError: true,
	}

	tests := []struct {
		name    string
		scene   *Scene
		wantErr error
	}{
		{
			name: "valid scene",
			scene: &Scene{
				Name:     "Cinema Mode",
				Priority: 50,
				Actions:  []SceneAction{validAction},
			},
			wantErr: nil,
		},
		{
			name:    "nil scene",
			scene:   nil,
			wantErr: ErrInvalidScene,
		},
		{
			name: "empty name",
			scene: &Scene{
				Name:     "",
				Priority: 50,
				Actions:  []SceneAction{validAction},
			},
			wantErr: ErrInvalidName,
		},
		{
			name: "whitespace-only name",
			scene: &Scene{
				Name:     "   ",
				Priority: 50,
				Actions:  []SceneAction{validAction},
			},
			wantErr: ErrInvalidName,
		},
		{
			name: "name too long",
			scene: &Scene{
				Name:     strings.Repeat("a", 101),
				Priority: 50,
				Actions:  []SceneAction{validAction},
			},
			wantErr: ErrInvalidName,
		},
		{
			name: "invalid slug",
			scene: &Scene{
				Name:     "Test",
				Slug:     "INVALID SLUG",
				Priority: 50,
				Actions:  []SceneAction{validAction},
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "slug too long",
			scene: &Scene{
				Name:     "Test",
				Slug:     strings.Repeat("a", 51),
				Priority: 50,
				Actions:  []SceneAction{validAction},
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "priority too low",
			scene: &Scene{
				Name:     "Test",
				Priority: 0,
				Actions:  []SceneAction{validAction},
			},
			wantErr: ErrInvalidScene,
		},
		{
			name: "priority too high",
			scene: &Scene{
				Name:     "Test",
				Priority: 101,
				Actions:  []SceneAction{validAction},
			},
			wantErr: ErrInvalidScene,
		},
		{
			name: "invalid category",
			scene: &Scene{
				Name:     "Test",
				Priority: 50,
				Category: "invalid",
				Actions:  []SceneAction{validAction},
			},
			wantErr: ErrInvalidScene,
		},
		{
			name: "valid category",
			scene: &Scene{
				Name:     "Test",
				Priority: 50,
				Category: CategoryEntertainment,
				Actions:  []SceneAction{validAction},
			},
			wantErr: nil,
		},
		{
			name: "no actions",
			scene: &Scene{
				Name:     "Test",
				Priority: 50,
				Actions:  []SceneAction{},
			},
			wantErr: ErrNoActions,
		},
		{
			name: "too many actions",
			scene: &Scene{
				Name:     "Test",
				Priority: 50,
				Actions:  make([]SceneAction, 101),
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "invalid action in scene",
			scene: &Scene{
				Name:     "Test",
				Priority: 50,
				Actions: []SceneAction{
					{DeviceID: "", Command: "set"},
				},
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "description too long",
			scene: func() *Scene {
				desc := strings.Repeat("x", 501)
				return &Scene{
					Name:        "Test",
					Description: &desc,
					Priority:    50,
					Actions:     []SceneAction{validAction},
				}
			}(),
			wantErr: ErrInvalidScene,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScene(tt.scene)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error %v, got nil", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateAction(t *testing.T) {
	tests := []struct {
		name    string
		action  SceneAction
		wantErr error
	}{
		{
			name: "valid action",
			action: SceneAction{
				DeviceID: "light-01",
				Command:  "set",
			},
			wantErr: nil,
		},
		{
			name: "valid action with all fields",
			action: SceneAction{
				DeviceID:        "light-01",
				Command:         "dim",
				Parameters:      map[string]any{"brightness": 50},
				DelayMS:         1000,
				FadeMS:          3000,
				Parallel:        true,
				ContinueOnError: true,
				SortOrder:       2,
			},
			wantErr: nil,
		},
		{
			name: "missing device_id",
			action: SceneAction{
				Command: "set",
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "missing command",
			action: SceneAction{
				DeviceID: "light-01",
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "negative delay",
			action: SceneAction{
				DeviceID: "light-01",
				Command:  "set",
				DelayMS:  -1,
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "delay too large",
			action: SceneAction{
				DeviceID: "light-01",
				Command:  "set",
				DelayMS:  300001,
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "negative fade",
			action: SceneAction{
				DeviceID: "light-01",
				Command:  "set",
				FadeMS:   -1,
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "fade too large",
			action: SceneAction{
				DeviceID: "light-01",
				Command:  "set",
				FadeMS:   60001,
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "too many parameters",
			action: SceneAction{
				DeviceID:   "light-01",
				Command:    "set",
				Parameters: makeNParams(21),
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "max parameters allowed",
			action: SceneAction{
				DeviceID:   "light-01",
				Command:    "set",
				Parameters: makeNParams(20),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAction(tt.action)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error %v, got nil", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple name", "Cinema Mode", "cinema-mode"},
		{"underscores", "good_morning", "good-morning"},
		{"special characters", "Hello World! #1", "hello-world-1"},
		{"multiple spaces", "all  off", "all-off"},
		{"leading trailing spaces", "  test  ", "test"},
		{"numbers", "scene 42", "scene-42"},
		{"already slug", "cinema-mode", "cinema-mode"},
		{"uppercase", "ALL LIGHTS OFF", "all-lights-off"},
		{
			"long name truncated",
			strings.Repeat("long-name-", 10),
			"long-name-long-name-long-name-long-name-long-name", // 50 chars, trailing hyphen trimmed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.input)
			if got != tt.want {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Verify result is a valid slug
			if got != "" {
				if err := ValidateSlug(got); err != nil {
					t.Errorf("GenerateSlug(%q) produced invalid slug %q: %v", tt.input, got, err)
				}
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	if id1 == "" {
		t.Error("GenerateID returned empty string")
	}
	if id1 == id2 {
		t.Error("GenerateID returned duplicate IDs")
	}
	// UUID format: 8-4-4-4-12 hex characters
	if len(id1) != 36 {
		t.Errorf("GenerateID length = %d, want 36", len(id1))
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "Cinema Mode", false},
		{"single char", "A", false},
		{"max length", strings.Repeat("a", 100), false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"too long", strings.Repeat("a", 101), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid slug", "cinema-mode", false},
		{"numbers", "scene-42", false},
		{"single word", "test", false},
		{"empty", "", true},
		{"uppercase", "Cinema", true},
		{"spaces", "cinema mode", true},
		{"special chars", "cinema_mode", true},
		{"leading hyphen", "-cinema", true},
		{"trailing hyphen", "cinema-", true},
		{"double hyphen", "cinema--mode", true},
		{"too long", strings.Repeat("a", 51), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlug(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSlug(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// makeNParams creates a map with n keys for testing parameter limits.
func makeNParams(n int) map[string]any {
	params := make(map[string]any, n)
	for i := range n {
		params[strings.Repeat("k", i+1)] = i
	}
	return params
}
