package auth

import (
	"errors"
	"regexp"
	"time"
)

// usernamePattern defines the valid format for usernames:
// alphanumeric, dots, hyphens, underscores, 1-64 characters.
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

// maxUsernameLength is the maximum allowed username length.
const maxUsernameLength = 64

// IsValidUsername checks if a username meets format requirements.
// Usernames must be 1-64 characters, alphanumeric with dots, hyphens, underscores.
func IsValidUsername(username string) bool {
	return len(username) <= maxUsernameLength && usernamePattern.MatchString(username)
}

// Role represents an authorisation tier in the system.
type Role string

const (
	// RolePanel is a wall-mounted display device identity (not a user account).
	// Scoped to assigned rooms. No login required.
	RolePanel Role = "panel"

	// RoleUser is a household member with explicit room grants.
	// Zero room assignments = no access.
	RoleUser Role = "user"

	// RoleAdmin has full system control: devices, scenes, users, panels,
	// settings, audit, commissioning. Professional installer or tech-savvy
	// homeowner. Bypasses room scoping.
	RoleAdmin Role = "admin"

	// RoleOwner has everything admin can do plus factory reset, dangerous
	// database operations, and managing other owners. Emergency-only —
	// credentials belong in a printed recovery pack.
	RoleOwner Role = "owner"
)

// ValidRoles is the set of valid user roles (excludes panel — panels are not users).
var ValidRoles = []Role{RoleUser, RoleAdmin, RoleOwner}

// IsValidUserRole returns true if the role is a valid role for a user account.
func IsValidUserRole(r Role) bool {
	for _, v := range ValidRoles {
		if r == v {
			return true
		}
	}
	return false
}

// User represents an authenticated human account.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"` // never serialised
	Role         Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedBy    string    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RefreshToken represents a stored refresh token for session management.
type RefreshToken struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	FamilyID   string    `json:"family_id"`
	TokenHash  string    `json:"-"` // never serialised
	DeviceInfo string    `json:"device_info,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
	Revoked    bool      `json:"revoked"`
	CreatedAt  time.Time `json:"created_at"`
}

// Panel represents a wall-mounted display device identity.
type Panel struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"` // never serialised
	IsActive   bool       `json:"is_active"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	CreatedBy  string     `json:"created_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// RoomAccess represents a user's access grant to a specific room.
type RoomAccess struct {
	UserID          string    `json:"user_id"`
	RoomID          string    `json:"room_id"`
	CanManageScenes bool      `json:"can_manage_scenes"`
	CreatedBy       string    `json:"created_by,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// PanelRoomAccess represents a panel's access grant to a specific room.
type PanelRoomAccess struct {
	PanelID   string    `json:"panel_id"`
	RoomID    string    `json:"room_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RoomScope holds the resolved room access for a user request context.
// A nil RoomScope means unrestricted access (admin/owner).
type RoomScope struct {
	// RoomIDs is the list of rooms the user can access (read/operate devices, execute scenes).
	RoomIDs []string

	// SceneManageRoomIDs is the subset of RoomIDs where the user can create/edit/delete scenes.
	SceneManageRoomIDs []string
}

// CanAccessRoom returns true if the room is in the scope's accessible rooms.
func (rs *RoomScope) CanAccessRoom(roomID string) bool {
	if rs == nil {
		return true // unrestricted
	}
	for _, id := range rs.RoomIDs {
		if id == roomID {
			return true
		}
	}
	return false
}

// CanManageScenesInRoom returns true if the user can create/edit scenes in the given room.
func (rs *RoomScope) CanManageScenesInRoom(roomID string) bool {
	if rs == nil {
		return true // unrestricted
	}
	for _, id := range rs.SceneManageRoomIDs {
		if id == roomID {
			return true
		}
	}
	return false
}

// Sentinel errors for auth operations.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrUsernameExists     = errors.New("username already exists")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrTokenInvalid       = errors.New("invalid token")
	ErrTokenReuse         = errors.New("refresh token reuse detected")
	ErrPanelNotFound      = errors.New("panel not found")
	ErrPanelInactive      = errors.New("panel is inactive")
	ErrForbidden          = errors.New("insufficient permissions")
	ErrSelfModification   = errors.New("cannot modify own account in this way")
)
