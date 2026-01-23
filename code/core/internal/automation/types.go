package automation

import "time"

// Scene represents a predefined collection of device actions that can be
// activated together. Actions execute in parallel or sequentially based on
// the Parallel flag, with optional delays and fade transitions.
type Scene struct {
	// Identity
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`

	// Description (optional)
	Description *string `json:"description,omitempty"`

	// Scope (optional location binding)
	RoomID *string `json:"room_id,omitempty"`
	AreaID *string `json:"area_id,omitempty"`

	// Configuration
	Enabled  bool `json:"enabled"`
	Priority int  `json:"priority"` // 1-100; higher = more important (default 50)

	// UI metadata
	Icon     *string  `json:"icon,omitempty"`
	Colour   *string  `json:"colour,omitempty"`   // Hex colour (#RRGGBB)
	Category Category `json:"category,omitempty"` // comfort, entertainment, daily, etc.

	// Actions to execute (ordered)
	Actions []SceneAction `json:"actions"`

	// Sort order for UI display
	SortOrder int `json:"sort_order"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SceneAction defines a single device command within a scene.
//
// Actions are executed in sort order. When Parallel is true, the action
// runs concurrently with the previous action's group. When false, it
// starts a new sequential group.
type SceneAction struct {
	// Target device
	DeviceID string `json:"device_id"`

	// Command to execute (e.g., "set", "dim", "position")
	Command string `json:"command"`

	// Command parameters (protocol-specific)
	Parameters map[string]any `json:"parameters,omitempty"`

	// Delay before executing (milliseconds, default 0)
	DelayMS int `json:"delay_ms"`

	// Fade/transition duration (milliseconds, default 0 = instant)
	FadeMS int `json:"fade_ms"`

	// When true, runs concurrently with previous action
	Parallel bool `json:"parallel"`

	// When true, scene continues even if this action fails (default false: fail-fast)
	ContinueOnError bool `json:"continue_on_error"`

	// Execution order within the scene
	SortOrder int `json:"sort_order"`
}

// SceneExecution tracks a single activation of a scene.
type SceneExecution struct {
	ID            string          `json:"id"`
	SceneID       string          `json:"scene_id"`
	TriggeredAt   time.Time       `json:"triggered_at"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	TriggerType   string          `json:"trigger_type"`             // manual, schedule, event, voice, automation
	TriggerSource *string         `json:"trigger_source,omitempty"` // api, wall_panel, mobile, etc.
	Status        ExecutionStatus `json:"status"`

	// Action counts
	ActionsTotal     int `json:"actions_total"`
	ActionsCompleted int `json:"actions_completed"`
	ActionsFailed    int `json:"actions_failed"`
	ActionsSkipped   int `json:"actions_skipped"`

	// Failure details (populated when actions fail)
	Failures []ActionFailure `json:"failures,omitempty"`

	// Total execution duration in milliseconds
	DurationMS *int `json:"duration_ms,omitempty"`
}

// ActionFailure records details of a failed action within an execution.
type ActionFailure struct {
	ActionIndex int    `json:"action_index"`
	DeviceID    string `json:"device_id"`
	Command     string `json:"command"`
	ErrorCode   string `json:"error_code"`
	ErrorMsg    string `json:"error_message"`
}

// ExecutionStatus represents the state of a scene execution.
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusPartial   ExecutionStatus = "partial"   // Some actions failed, but scene continued
	StatusFailed    ExecutionStatus = "failed"    // Critical action failed, scene aborted
	StatusCancelled ExecutionStatus = "cancelled" // Context cancelled mid-execution
)

// Category represents a scene category for UI organisation.
type Category string

const (
	CategoryComfort       Category = "comfort"
	CategoryEntertainment Category = "entertainment"
	CategoryProductivity  Category = "productivity"
	CategoryDaily         Category = "daily"
	CategorySecurity      Category = "security"
	CategoryEnergy        Category = "energy"
)

// AllCategories returns all valid scene categories.
func AllCategories() []Category {
	return []Category{
		CategoryComfort,
		CategoryEntertainment,
		CategoryProductivity,
		CategoryDaily,
		CategorySecurity,
		CategoryEnergy,
	}
}

// DeepCopy creates a complete independent copy of the Scene.
// All map and slice fields are cloned so modifications to the copy
// do not affect the original. This is essential for cache isolation.
func (s *Scene) DeepCopy() *Scene {
	if s == nil {
		return nil
	}

	cpy := *s // Shallow copy of value fields

	// Deep copy string pointer fields to prevent cache corruption
	cpy.Description = cloneStringPtr(s.Description)
	cpy.RoomID = cloneStringPtr(s.RoomID)
	cpy.AreaID = cloneStringPtr(s.AreaID)
	cpy.Icon = cloneStringPtr(s.Icon)
	cpy.Colour = cloneStringPtr(s.Colour)

	// Deep copy Actions slice (each action's Parameters map needs cloning)
	if s.Actions != nil {
		cpy.Actions = make([]SceneAction, len(s.Actions))
		for i, action := range s.Actions {
			cpy.Actions[i] = action
			if action.Parameters != nil {
				cpy.Actions[i].Parameters = deepCopyMap(action.Parameters)
			}
		}
	}

	return &cpy
}

// deepCopyMap creates a deep copy of a map[string]any.
// Nested maps and slices are recursively copied.
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	cpy := make(map[string]any, len(m))
	for k, v := range m {
		cpy[k] = deepCopyValue(v)
	}
	return cpy
}

// deepCopyValue recursively copies a value, handling nested maps and slices.
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case []any:
		cpy := make([]any, len(val))
		for i, elem := range val {
			cpy[i] = deepCopyValue(elem)
		}
		return cpy
	default:
		return v // Primitives are immutable
	}
}

// cloneStringPtr creates an independent copy of a *string.
func cloneStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}
