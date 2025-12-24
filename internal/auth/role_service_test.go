package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Test helpers
func createTestUserWithRole(id int64, username string, role Role) *User {
	return &User{
		ID:        id,
		Username:  username,
		Email:     username + "@example.com",
		Role:      role,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// setupTestRepo creates a mock repo and adds a user directly
func setupTestRepo() *MockUserRepository {
	return NewMockUserRepository()
}

func addUserToRepo(repo *MockUserRepository, user *User) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()
	repo.users[user.ID] = user
	if user.ID >= repo.nextID {
		repo.nextID = user.ID + 1
	}
}

// ============================================
// ADMIN PROMOTING USER TO MODERATOR
// ============================================

func TestRoleService_PromoteToModerator_AdminCanPromote(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	admin := createTestUserWithRole(1, "admin", RoleAdmin)
	user := createTestUserWithRole(2, "regularuser", RoleUser)
	repo.users[admin.ID] = admin
	repo.users[user.ID] = user

	err := service.ChangeUserRole(ctx, admin, user.ID, RoleModerator)
	if err != nil {
		t.Fatalf("Admin should be able to promote user to moderator: %v", err)
	}

	updatedUser, _ := repo.GetUserByID(user.ID)
	if updatedUser.Role != RoleModerator {
		t.Errorf("User role = %v, want %v", updatedUser.Role, RoleModerator)
	}
}

func TestRoleService_PromoteToModerator_ModeratorCannotPromote(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	moderator := createTestUserWithRole(1, "moderator", RoleModerator)
	user := createTestUserWithRole(2, "regularuser", RoleUser)
	repo.users[moderator.ID] = moderator
	repo.users[user.ID] = user

	err := service.ChangeUserRole(ctx, moderator, user.ID, RoleModerator)
	if err == nil {
		t.Fatal("Moderator should NOT be able to promote users")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

func TestRoleService_PromoteToModerator_UserCannotPromote(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	regularUser := createTestUserWithRole(1, "user1", RoleUser)
	otherUser := createTestUserWithRole(2, "user2", RoleUser)
	repo.users[regularUser.ID] = regularUser
	repo.users[otherUser.ID] = otherUser

	err := service.ChangeUserRole(ctx, regularUser, otherUser.ID, RoleModerator)
	if err == nil {
		t.Fatal("Regular user should NOT be able to promote users")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

// ============================================
// ADMIN PROMOTING USER TO ADMIN
// ============================================

func TestRoleService_PromoteToAdmin_AdminCannotPromoteToAdmin(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	admin := createTestUserWithRole(1, "admin", RoleAdmin)
	user := createTestUserWithRole(2, "regularuser", RoleUser)
	repo.users[admin.ID] = admin
	repo.users[user.ID] = user

	err := service.ChangeUserRole(ctx, admin, user.ID, RoleAdmin)
	if err == nil {
		t.Fatal("Admin should NOT be able to promote user to admin (only super_admin can)")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

func TestRoleService_PromoteToAdmin_SuperAdminCanPromote(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	superAdmin := createTestUserWithRole(1, "superadmin", RoleSuperAdmin)
	user := createTestUserWithRole(2, "regularuser", RoleUser)
	repo.users[superAdmin.ID] = superAdmin
	repo.users[user.ID] = user

	err := service.ChangeUserRole(ctx, superAdmin, user.ID, RoleAdmin)
	if err != nil {
		t.Fatalf("SuperAdmin should be able to promote user to admin: %v", err)
	}

	updatedUser, _ := repo.GetUserByID(user.ID)
	if updatedUser.Role != RoleAdmin {
		t.Errorf("User role = %v, want %v", updatedUser.Role, RoleAdmin)
	}
}

// ============================================
// SUPER ADMIN MANAGEMENT
// ============================================

func TestRoleService_PromoteToSuperAdmin_SuperAdminCanPromote(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	superAdmin := createTestUserWithRole(1, "superadmin", RoleSuperAdmin)
	admin := createTestUserWithRole(2, "admin", RoleAdmin)
	repo.users[superAdmin.ID] = superAdmin
	repo.users[admin.ID] = admin

	err := service.ChangeUserRole(ctx, superAdmin, admin.ID, RoleSuperAdmin)
	if err != nil {
		t.Fatalf("SuperAdmin should be able to promote admin to super_admin: %v", err)
	}

	updatedUser, _ := repo.GetUserByID(admin.ID)
	if updatedUser.Role != RoleSuperAdmin {
		t.Errorf("User role = %v, want %v", updatedUser.Role, RoleSuperAdmin)
	}
}

func TestRoleService_DemoteSuperAdmin_PreventLastSuperAdmin(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	// Only one super admin exists
	superAdmin := createTestUserWithRole(1, "superadmin", RoleSuperAdmin)
	repo.users[superAdmin.ID] = superAdmin

	err := service.ChangeUserRole(ctx, superAdmin, superAdmin.ID, RoleAdmin)
	if err == nil {
		t.Fatal("Should NOT be able to demote the last super_admin")
	}
	if !errors.Is(err, ErrLastSuperAdmin) {
		t.Errorf("Expected ErrLastSuperAdmin, got: %v", err)
	}
}

func TestRoleService_DemoteSuperAdmin_AllowedWhenOtherSuperAdminExists(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	// Two super admins exist
	superAdmin1 := createTestUserWithRole(1, "superadmin1", RoleSuperAdmin)
	superAdmin2 := createTestUserWithRole(2, "superadmin2", RoleSuperAdmin)
	repo.users[superAdmin1.ID] = superAdmin1
	repo.users[superAdmin2.ID] = superAdmin2

	err := service.ChangeUserRole(ctx, superAdmin1, superAdmin2.ID, RoleAdmin)
	if err != nil {
		t.Fatalf("Should be able to demote super_admin when another exists: %v", err)
	}

	updatedUser, _ := repo.GetUserByID(superAdmin2.ID)
	if updatedUser.Role != RoleAdmin {
		t.Errorf("User role = %v, want %v", updatedUser.Role, RoleAdmin)
	}
}

// ============================================
// DEMOTING USERS
// ============================================

func TestRoleService_DemoteModerator_AdminCanDemote(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	admin := createTestUserWithRole(1, "admin", RoleAdmin)
	moderator := createTestUserWithRole(2, "moderator", RoleModerator)
	repo.users[admin.ID] = admin
	repo.users[moderator.ID] = moderator

	err := service.ChangeUserRole(ctx, admin, moderator.ID, RoleUser)
	if err != nil {
		t.Fatalf("Admin should be able to demote moderator to user: %v", err)
	}

	updatedUser, _ := repo.GetUserByID(moderator.ID)
	if updatedUser.Role != RoleUser {
		t.Errorf("User role = %v, want %v", updatedUser.Role, RoleUser)
	}
}

func TestRoleService_DemoteAdmin_AdminCannotDemoteOtherAdmin(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	admin1 := createTestUserWithRole(1, "admin1", RoleAdmin)
	admin2 := createTestUserWithRole(2, "admin2", RoleAdmin)
	repo.users[admin1.ID] = admin1
	repo.users[admin2.ID] = admin2

	err := service.ChangeUserRole(ctx, admin1, admin2.ID, RoleUser)
	if err == nil {
		t.Fatal("Admin should NOT be able to demote other admin")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

func TestRoleService_DemoteAdmin_SuperAdminCanDemote(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	superAdmin := createTestUserWithRole(1, "superadmin", RoleSuperAdmin)
	admin := createTestUserWithRole(2, "admin", RoleAdmin)
	repo.users[superAdmin.ID] = superAdmin
	repo.users[admin.ID] = admin

	err := service.ChangeUserRole(ctx, superAdmin, admin.ID, RoleUser)
	if err != nil {
		t.Fatalf("SuperAdmin should be able to demote admin: %v", err)
	}

	updatedUser, _ := repo.GetUserByID(admin.ID)
	if updatedUser.Role != RoleUser {
		t.Errorf("User role = %v, want %v", updatedUser.Role, RoleUser)
	}
}

// ============================================
// LIST USERS BY ROLE
// ============================================

func TestRoleService_ListModerators(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	admin := createTestUserWithRole(1, "admin", RoleAdmin)
	mod1 := createTestUserWithRole(2, "mod1", RoleModerator)
	mod2 := createTestUserWithRole(3, "mod2", RoleModerator)
	user := createTestUserWithRole(4, "user", RoleUser)
	repo.users[admin.ID] = admin
	repo.users[mod1.ID] = mod1
	repo.users[mod2.ID] = mod2
	repo.users[user.ID] = user

	mods, err := service.ListModerators(ctx, admin)
	if err != nil {
		t.Fatalf("Admin should be able to list moderators: %v", err)
	}
	if len(mods) != 2 {
		t.Errorf("Expected 2 moderators, got %d", len(mods))
	}
}

func TestRoleService_ListAdmins_OnlySuperAdminCanList(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	superAdmin := createTestUserWithRole(1, "superadmin", RoleSuperAdmin)
	admin1 := createTestUserWithRole(2, "admin1", RoleAdmin)
	admin2 := createTestUserWithRole(3, "admin2", RoleAdmin)
	repo.users[superAdmin.ID] = superAdmin
	repo.users[admin1.ID] = admin1
	repo.users[admin2.ID] = admin2

	// Super admin can list admins (now returns both admins AND super_admins)
	admins, err := service.ListAdmins(ctx, superAdmin)
	if err != nil {
		t.Fatalf("SuperAdmin should be able to list admins: %v", err)
	}
	// Expected: 2 admins + 1 super_admin = 3 total
	if len(admins) != 3 {
		t.Errorf("Expected 3 admins (2 admin + 1 super_admin), got %d", len(admins))
	}

	// Regular admin cannot list admins
	_, err = service.ListAdmins(ctx, admin1)
	if err == nil {
		t.Fatal("Admin should NOT be able to list admins")
	}
}

// ============================================
// SELF-MODIFICATION PREVENTION
// ============================================

func TestRoleService_CannotChangeSelfRole_UnlessSuperAdmin(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	admin := createTestUserWithRole(1, "admin", RoleAdmin)
	repo.users[admin.ID] = admin

	err := service.ChangeUserRole(ctx, admin, admin.ID, RoleSuperAdmin)
	if err == nil {
		t.Fatal("User should NOT be able to change their own role")
	}
	if !errors.Is(err, ErrCannotModifySelf) {
		t.Errorf("Expected ErrCannotModifySelf, got: %v", err)
	}
}

// ============================================
// INVALID ROLE
// ============================================

func TestRoleService_InvalidRoleReturnsError(t *testing.T) {
	repo := NewMockUserRepository()
	service := NewRoleService(repo)
	ctx := context.Background()

	superAdmin := createTestUserWithRole(1, "superadmin", RoleSuperAdmin)
	user := createTestUserWithRole(2, "user", RoleUser)
	repo.users[superAdmin.ID] = superAdmin
	repo.users[user.ID] = user

	err := service.ChangeUserRole(ctx, superAdmin, user.ID, Role("invalid_role"))
	if err == nil {
		t.Fatal("Should return error for invalid role")
	}
}
