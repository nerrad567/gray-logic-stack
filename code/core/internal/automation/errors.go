package automation

import "errors"

// Domain errors for the automation package.
//
// These errors can be checked using errors.Is() for error handling:
//
//	if errors.Is(err, automation.ErrSceneNotFound) {
//	    // handle not found case
//	}
var (
	// ErrSceneNotFound is returned when a scene ID does not exist.
	ErrSceneNotFound = errors.New("scene: not found")

	// ErrSceneExists is returned when creating a scene with an ID that already exists.
	ErrSceneExists = errors.New("scene: already exists")

	// ErrSceneDisabled is returned when attempting to activate a disabled scene.
	ErrSceneDisabled = errors.New("scene: disabled")

	// ErrInvalidScene is returned when scene validation fails.
	ErrInvalidScene = errors.New("scene: invalid")

	// ErrInvalidAction is returned when a scene action is invalid.
	ErrInvalidAction = errors.New("scene: invalid action")

	// ErrInvalidName is returned when a scene name is empty or too long.
	ErrInvalidName = errors.New("scene: invalid name")

	// ErrInvalidSlug is returned when a slug format is invalid.
	ErrInvalidSlug = errors.New("scene: invalid slug")

	// ErrNoActions is returned when a scene has no actions defined.
	ErrNoActions = errors.New("scene: no actions")

	// ErrExecutionNotFound is returned when an execution ID does not exist.
	ErrExecutionNotFound = errors.New("scene: execution not found")

	// ErrMQTTUnavailable is returned when MQTT is not connected.
	ErrMQTTUnavailable = errors.New("scene: MQTT unavailable")
)
