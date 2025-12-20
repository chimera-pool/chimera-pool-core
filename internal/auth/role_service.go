package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidRole      = errors.New("invalid role")
	ErrLastSuperAdmin   = errors.New("cannot demote the last super admin")
	ErrCannotModifySelf = errors.New("cannot modify your own role")
)

// RoleRepository extends UserRepository with role-specific operations
type RoleRepository interface {
	UserRepository
	ListUsersByRole(ctx context.Context, role Role) ([]*User, error)
	CountUsersByRole(ctx context.Context, role Role) (int, error)
}

// RoleService handles role management operations
type RoleService struct {
	repo RoleRepository
}

// NewRoleService creates a new role service
func NewRoleService(repo RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

// ChangeUserRole changes a user's role
func (s *RoleService) ChangeUserRole(ctx context.Context, actor *User, targetUserID int64, newRole Role) error {
	// Validate new role
	if !newRole.IsValid() {
		return ErrInvalidRole
	}

	// Get target user
	targetUser, err := s.repo.GetUserByID(targetUserID)
	if err != nil {
		return err
	}
	if targetUser == nil {
		return ErrUserNotFound
	}

	// Prevent self-modification (except super admin demoting themselves when others exist)
	if actor.ID == targetUserID {
		// Super admin can demote themselves if there are other super admins
		if actor.Role == RoleSuperAdmin && newRole != RoleSuperAdmin {
			count, err := s.repo.CountUsersByRole(ctx, RoleSuperAdmin)
			if err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastSuperAdmin
			}
			// Allow self-demotion if other super admins exist
		} else {
			return ErrCannotModifySelf
		}
	}

	// Check if actor can manage the target's current role
	if !actor.Role.CanManageRole(targetUser.Role) {
		return ErrPermissionDenied
	}

	// Check if actor can assign the new role
	if !actor.Role.CanManageRole(newRole) {
		return ErrPermissionDenied
	}

	// Special case: demoting from super_admin
	if targetUser.Role == RoleSuperAdmin && newRole != RoleSuperAdmin {
		count, err := s.repo.CountUsersByRole(ctx, RoleSuperAdmin)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastSuperAdmin
		}
	}

	// Update user role
	targetUser.Role = newRole
	targetUser.UpdatedAt = time.Now()

	return s.repo.UpdateUser(targetUser)
}

// ListModerators returns all users with moderator role
func (s *RoleService) ListModerators(ctx context.Context, actor *User) ([]*User, error) {
	// Admin or higher can list moderators
	if actor.Role.Level() < RoleAdmin.Level() {
		return nil, ErrPermissionDenied
	}
	return s.repo.ListUsersByRole(ctx, RoleModerator)
}

// ListAdmins returns all users with admin role
func (s *RoleService) ListAdmins(ctx context.Context, actor *User) ([]*User, error) {
	// Only super admin can list admins
	if actor.Role != RoleSuperAdmin {
		return nil, ErrPermissionDenied
	}
	return s.repo.ListUsersByRole(ctx, RoleAdmin)
}

// PromoteToModerator promotes a user to moderator
func (s *RoleService) PromoteToModerator(ctx context.Context, actor *User, targetUserID int64) error {
	return s.ChangeUserRole(ctx, actor, targetUserID, RoleModerator)
}

// PromoteToAdmin promotes a user to admin
func (s *RoleService) PromoteToAdmin(ctx context.Context, actor *User, targetUserID int64) error {
	return s.ChangeUserRole(ctx, actor, targetUserID, RoleAdmin)
}

// DemoteToUser demotes a user to regular user
func (s *RoleService) DemoteToUser(ctx context.Context, actor *User, targetUserID int64) error {
	return s.ChangeUserRole(ctx, actor, targetUserID, RoleUser)
}

// GetUserRole returns a user's role
func (s *RoleService) GetUserRole(ctx context.Context, userID int64) (Role, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	return user.Role, nil
}
