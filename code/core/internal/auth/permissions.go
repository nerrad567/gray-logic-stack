package auth

// Permission represents a named capability in the system.
type Permission string

// Permission constants.
const (
	PermDeviceRead       Permission = "device:read"
	PermDeviceOperate    Permission = "device:operate"
	PermDeviceConfigure  Permission = "device:configure"
	PermSceneExecute     Permission = "scene:execute"
	PermSceneManage      Permission = "scene:manage"
	PermLocationManage   Permission = "location:manage"
	PermCommissionManage Permission = "commission:manage"
	PermUserManage       Permission = "user:manage"
	PermUserManageAll    Permission = "user:manage:all"
	PermSystemAdmin      Permission = "system:admin"
	PermSystemDangerous  Permission = "system:dangerous"
)

// rolePermissions maps each role to its granted permissions.
// This is the single source of truth for the authorisation model.
// Panel permissions are handled separately via room scoping.
var rolePermissions = map[Role][]Permission{
	RoleUser: {
		PermDeviceRead,
		PermDeviceOperate,
		PermSceneExecute,
		PermSceneManage, // room-scoped: only where can_manage_scenes=1
	},
	RoleAdmin: {
		PermDeviceRead,
		PermDeviceOperate,
		PermDeviceConfigure,
		PermSceneExecute,
		PermSceneManage,
		PermLocationManage,
		PermCommissionManage,
		PermUserManage,
		PermSystemAdmin,
	},
	RoleOwner: {
		PermDeviceRead,
		PermDeviceOperate,
		PermDeviceConfigure,
		PermSceneExecute,
		PermSceneManage,
		PermLocationManage,
		PermCommissionManage,
		PermUserManage,
		PermUserManageAll,
		PermSystemAdmin,
		PermSystemDangerous,
	},
}

// panelPermissions are the permissions available to panel device identities.
// All panel permissions are room-scoped via panel_room_access.
var panelPermissions = []Permission{
	PermDeviceRead,
	PermDeviceOperate,
	PermSceneExecute,
}

// HasPermission returns true if the given role has the specified permission.
// For the panel role, use HasPanelPermission instead.
func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// HasPanelPermission returns true if panels have the specified permission.
func HasPanelPermission(perm Permission) bool {
	for _, p := range panelPermissions {
		if p == perm {
			return true
		}
	}
	return false
}

// PermissionsForRole returns all permissions granted to a role.
// Returns nil for unknown roles.
func PermissionsForRole(role Role) []Permission {
	perms := rolePermissions[role]
	if perms == nil {
		return nil
	}
	result := make([]Permission, len(perms))
	copy(result, perms)
	return result
}

// IsRoomScoped returns true if the role's permissions are subject to room scoping.
// Only the user role and panel identity are room-scoped.
func IsRoomScoped(role Role) bool {
	return role == RoleUser
}
