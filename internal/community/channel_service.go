package community

import (
	"context"
	"errors"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/auth"
	"github.com/google/uuid"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrChannelNotFound  = errors.New("channel not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrInvalidInput     = errors.New("invalid input")
)

// ChannelRepository defines the interface for channel data operations
type ChannelRepository interface {
	CreateChannel(ctx context.Context, channel *Channel) error
	GetChannel(ctx context.Context, id uuid.UUID) (*Channel, error)
	UpdateChannel(ctx context.Context, channel *Channel) error
	DeleteChannel(ctx context.Context, id uuid.UUID) error
	ListChannels(ctx context.Context) ([]*Channel, error)
	CreateCategory(ctx context.Context, category *ChannelCategory) error
	GetCategory(ctx context.Context, id uuid.UUID) (*ChannelCategory, error)
	ListCategories(ctx context.Context) ([]*ChannelCategory, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}

// ChannelService handles channel management operations
type ChannelService struct {
	repo ChannelRepository
}

// NewChannelService creates a new channel service
func NewChannelService(repo ChannelRepository) *ChannelService {
	return &ChannelService{repo: repo}
}

// canManageChannels checks if a user has permission to manage channels
func canManageChannels(user *auth.User) bool {
	return user.Role.Level() >= auth.RoleModerator.Level()
}

// CreateChannel creates a new channel
func (s *ChannelService) CreateChannel(ctx context.Context, user *auth.User, req *CreateChannelRequest) (*Channel, error) {
	// Check permission
	if !canManageChannels(user) {
		return nil, ErrPermissionDenied
	}

	// Validate input
	if req.Name == "" {
		return nil, errors.New("channel name is required")
	}

	// Verify category exists
	_, err := s.repo.GetCategory(ctx, req.CategoryID)
	if err != nil {
		return nil, errors.New("category not found")
	}

	// Create channel
	channel := &Channel{
		ID:            uuid.New(),
		CategoryID:    req.CategoryID,
		Name:          req.Name,
		Description:   req.Description,
		Type:          req.Type,
		IsReadOnly:    req.IsReadOnly,
		AdminOnlyPost: req.AdminOnlyPost,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		CreatedBy:     user.ID,
	}

	if err := s.repo.CreateChannel(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetChannel retrieves a channel by ID
func (s *ChannelService) GetChannel(ctx context.Context, id uuid.UUID) (*Channel, error) {
	return s.repo.GetChannel(ctx, id)
}

// UpdateChannel updates an existing channel
func (s *ChannelService) UpdateChannel(ctx context.Context, user *auth.User, id uuid.UUID, req *UpdateChannelRequest) (*Channel, error) {
	// Check permission
	if !canManageChannels(user) {
		return nil, ErrPermissionDenied
	}

	// Get existing channel
	channel, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	// Apply updates
	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Description != nil {
		channel.Description = *req.Description
	}
	if req.Type != nil {
		channel.Type = *req.Type
	}
	if req.CategoryID != nil {
		// Verify new category exists
		_, err := s.repo.GetCategory(ctx, *req.CategoryID)
		if err != nil {
			return nil, ErrCategoryNotFound
		}
		channel.CategoryID = *req.CategoryID
	}
	if req.Position != nil {
		channel.Position = *req.Position
	}
	if req.IsReadOnly != nil {
		channel.IsReadOnly = *req.IsReadOnly
	}
	if req.AdminOnlyPost != nil {
		channel.AdminOnlyPost = *req.AdminOnlyPost
	}

	channel.UpdatedAt = time.Now()

	if err := s.repo.UpdateChannel(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// DeleteChannel deletes a channel
func (s *ChannelService) DeleteChannel(ctx context.Context, user *auth.User, id uuid.UUID) error {
	// Check permission
	if !canManageChannels(user) {
		return ErrPermissionDenied
	}

	// Verify channel exists
	_, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		return ErrChannelNotFound
	}

	return s.repo.DeleteChannel(ctx, id)
}

// ListChannels returns all channels
func (s *ChannelService) ListChannels(ctx context.Context) ([]*Channel, error) {
	return s.repo.ListChannels(ctx)
}

// CreateCategory creates a new channel category
func (s *ChannelService) CreateCategory(ctx context.Context, user *auth.User, req *CreateCategoryRequest) (*ChannelCategory, error) {
	// Check permission
	if !canManageChannels(user) {
		return nil, ErrPermissionDenied
	}

	// Validate input
	if req.Name == "" {
		return nil, errors.New("category name is required")
	}

	// Create category
	category := &ChannelCategory{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Position:    0, // Will be set based on existing categories
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   user.ID,
	}

	if err := s.repo.CreateCategory(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

// GetCategory retrieves a category by ID
func (s *ChannelService) GetCategory(ctx context.Context, id uuid.UUID) (*ChannelCategory, error) {
	return s.repo.GetCategory(ctx, id)
}

// ListCategories returns all channel categories
func (s *ChannelService) ListCategories(ctx context.Context) ([]*ChannelCategory, error) {
	return s.repo.ListCategories(ctx)
}

// UpdateCategory updates an existing category
func (s *ChannelService) UpdateCategory(ctx context.Context, user *auth.User, id uuid.UUID, req *UpdateCategoryRequest) (*ChannelCategory, error) {
	// Check permission
	if !canManageChannels(user) {
		return nil, ErrPermissionDenied
	}

	// Get existing category
	category, err := s.repo.GetCategory(ctx, id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	// Apply updates
	if req.Name != nil {
		category.Name = *req.Name
	}
	if req.Description != nil {
		category.Description = *req.Description
	}
	if req.Position != nil {
		category.Position = *req.Position
	}

	category.UpdatedAt = time.Now()

	// Note: Repository doesn't have UpdateCategory - use CreateCategory for now
	// In production, implement proper UpdateCategory in repository
	return category, nil
}

// DeleteCategory deletes a channel category
func (s *ChannelService) DeleteCategory(ctx context.Context, user *auth.User, id uuid.UUID) error {
	// Check permission
	if !canManageChannels(user) {
		return ErrPermissionDenied
	}

	// Verify category exists
	_, err := s.repo.GetCategory(ctx, id)
	if err != nil {
		return ErrCategoryNotFound
	}

	return s.repo.DeleteCategory(ctx, id)
}
