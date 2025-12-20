package auth

import (
	"testing"
)

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{"valid user role", RoleUser, true},
		{"valid moderator role", RoleModerator, true},
		{"valid admin role", RoleAdmin, true},
		{"valid super_admin role", RoleSuperAdmin, true},
		{"invalid role", Role("invalid"), false},
		{"empty role", Role(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.expected {
				t.Errorf("Role.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRole_Level(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected int
	}{
		{"user level", RoleUser, 1},
		{"moderator level", RoleModerator, 2},
		{"admin level", RoleAdmin, 3},
		{"super_admin level", RoleSuperAdmin, 4},
		{"invalid role level", Role("invalid"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.Level(); got != tt.expected {
				t.Errorf("Role.Level() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRole_CanManageRole(t *testing.T) {
	tests := []struct {
		name     string
		actor    Role
		target   Role
		expected bool
	}{
		// Super admin can manage everyone
		{"super_admin can manage user", RoleSuperAdmin, RoleUser, true},
		{"super_admin can manage moderator", RoleSuperAdmin, RoleModerator, true},
		{"super_admin can manage admin", RoleSuperAdmin, RoleAdmin, true},
		{"super_admin can manage super_admin", RoleSuperAdmin, RoleSuperAdmin, true},

		// Admin can manage users and moderators only
		{"admin can manage user", RoleAdmin, RoleUser, true},
		{"admin can manage moderator", RoleAdmin, RoleModerator, true},
		{"admin cannot manage admin", RoleAdmin, RoleAdmin, false},
		{"admin cannot manage super_admin", RoleAdmin, RoleSuperAdmin, false},

		// Moderator cannot manage roles
		{"moderator cannot manage user", RoleModerator, RoleUser, false},
		{"moderator cannot manage moderator", RoleModerator, RoleModerator, false},
		{"moderator cannot manage admin", RoleModerator, RoleAdmin, false},
		{"moderator cannot manage super_admin", RoleModerator, RoleSuperAdmin, false},

		// User cannot manage roles
		{"user cannot manage user", RoleUser, RoleUser, false},
		{"user cannot manage moderator", RoleUser, RoleModerator, false},
		{"user cannot manage admin", RoleUser, RoleAdmin, false},
		{"user cannot manage super_admin", RoleUser, RoleSuperAdmin, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.actor.CanManageRole(tt.target); got != tt.expected {
				t.Errorf("Role.CanManageRole() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRole_Hierarchy(t *testing.T) {
	// Test that role levels follow correct hierarchy
	if RoleUser.Level() >= RoleModerator.Level() {
		t.Error("User should have lower level than Moderator")
	}
	if RoleModerator.Level() >= RoleAdmin.Level() {
		t.Error("Moderator should have lower level than Admin")
	}
	if RoleAdmin.Level() >= RoleSuperAdmin.Level() {
		t.Error("Admin should have lower level than SuperAdmin")
	}
}
