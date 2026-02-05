package auth

import "testing"

func TestHasPermission_Owner(t *testing.T) {
	// Owner should have all permissions
	allPerms := []Permission{
		PermDeviceRead, PermDeviceOperate, PermDeviceConfigure,
		PermSceneExecute, PermSceneManage,
		PermLocationManage, PermCommissionManage,
		PermUserManage, PermUserManageAll,
		PermSystemAdmin, PermSystemDangerous,
	}

	for _, perm := range allPerms {
		if !HasPermission(RoleOwner, perm) {
			t.Errorf("owner should have %s", perm)
		}
	}
}

func TestHasPermission_Admin(t *testing.T) {
	// Admin should have most permissions but not dangerous/manage-all
	should := []Permission{
		PermDeviceRead, PermDeviceOperate, PermDeviceConfigure,
		PermSceneExecute, PermSceneManage,
		PermLocationManage, PermCommissionManage,
		PermUserManage, PermSystemAdmin,
	}
	shouldNot := []Permission{
		PermUserManageAll, PermSystemDangerous,
	}

	for _, perm := range should {
		if !HasPermission(RoleAdmin, perm) {
			t.Errorf("admin should have %s", perm)
		}
	}
	for _, perm := range shouldNot {
		if HasPermission(RoleAdmin, perm) {
			t.Errorf("admin should NOT have %s", perm)
		}
	}
}

func TestHasPermission_User(t *testing.T) {
	// User should have basic device + scene permissions only
	should := []Permission{
		PermDeviceRead, PermDeviceOperate,
		PermSceneExecute, PermSceneManage,
	}
	shouldNot := []Permission{
		PermDeviceConfigure,
		PermLocationManage, PermCommissionManage,
		PermUserManage, PermUserManageAll,
		PermSystemAdmin, PermSystemDangerous,
	}

	for _, perm := range should {
		if !HasPermission(RoleUser, perm) {
			t.Errorf("user should have %s", perm)
		}
	}
	for _, perm := range shouldNot {
		if HasPermission(RoleUser, perm) {
			t.Errorf("user should NOT have %s", perm)
		}
	}
}

func TestHasPermission_InvalidRole(t *testing.T) {
	if HasPermission(Role("nonexistent"), PermDeviceRead) {
		t.Error("unknown role should have no permissions")
	}
}

func TestHasPanelPermission(t *testing.T) {
	should := []Permission{PermDeviceRead, PermDeviceOperate, PermSceneExecute}
	shouldNot := []Permission{
		PermSceneManage, PermDeviceConfigure,
		PermLocationManage, PermCommissionManage,
		PermUserManage, PermSystemAdmin, PermSystemDangerous,
	}

	for _, perm := range should {
		if !HasPanelPermission(perm) {
			t.Errorf("panel should have %s", perm)
		}
	}
	for _, perm := range shouldNot {
		if HasPanelPermission(perm) {
			t.Errorf("panel should NOT have %s", perm)
		}
	}
}

func TestPermissionsForRole(t *testing.T) {
	perms := PermissionsForRole(RoleAdmin)
	if perms == nil {
		t.Fatal("PermissionsForRole(admin) should not return nil")
	}
	if len(perms) == 0 {
		t.Error("PermissionsForRole(admin) should return permissions")
	}

	// Should return a copy, not the original slice
	perms[0] = "modified"
	original := PermissionsForRole(RoleAdmin)
	if original[0] == "modified" {
		t.Error("PermissionsForRole should return a copy, not the original")
	}
}

func TestPermissionsForRole_Unknown(t *testing.T) {
	perms := PermissionsForRole(Role("unknown"))
	if perms != nil {
		t.Error("PermissionsForRole(unknown) should return nil")
	}
}

func TestIsRoomScoped(t *testing.T) {
	if !IsRoomScoped(RoleUser) {
		t.Error("user role should be room-scoped")
	}
	if IsRoomScoped(RoleAdmin) {
		t.Error("admin role should NOT be room-scoped")
	}
	if IsRoomScoped(RoleOwner) {
		t.Error("owner role should NOT be room-scoped")
	}
}

func TestIsValidUserRole(t *testing.T) {
	if !IsValidUserRole(RoleUser) {
		t.Error("user should be a valid user role")
	}
	if !IsValidUserRole(RoleAdmin) {
		t.Error("admin should be a valid user role")
	}
	if !IsValidUserRole(RoleOwner) {
		t.Error("owner should be a valid user role")
	}
	if IsValidUserRole(RolePanel) {
		t.Error("panel should NOT be a valid user role")
	}
	if IsValidUserRole(Role("guest")) {
		t.Error("guest should NOT be a valid user role")
	}
}
