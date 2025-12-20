package community

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/auth"
	"github.com/google/uuid"
)

// MockChannelRepository is a mock implementation for testing
type MockChannelRepository struct {
	channels   map[uuid.UUID]*Channel
	categories map[uuid.UUID]*ChannelCategory
	createErr  error
	getErr     error
	updateErr  error
	deleteErr  error
}

func NewMockChannelRepository() *MockChannelRepository {
	return &MockChannelRepository{
		channels:   make(map[uuid.UUID]*Channel),
		categories: make(map[uuid.UUID]*ChannelCategory),
	}
}

func (m *MockChannelRepository) CreateChannel(ctx context.Context, channel *Channel) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.channels[channel.ID] = channel
	return nil
}

func (m *MockChannelRepository) GetChannel(ctx context.Context, id uuid.UUID) (*Channel, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	ch, ok := m.channels[id]
	if !ok {
		return nil, errors.New("channel not found")
	}
	return ch, nil
}

func (m *MockChannelRepository) UpdateChannel(ctx context.Context, channel *Channel) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.channels[channel.ID] = channel
	return nil
}

func (m *MockChannelRepository) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.channels, id)
	return nil
}

func (m *MockChannelRepository) ListChannels(ctx context.Context) ([]*Channel, error) {
	var result []*Channel
	for _, ch := range m.channels {
		result = append(result, ch)
	}
	return result, nil
}

func (m *MockChannelRepository) CreateCategory(ctx context.Context, category *ChannelCategory) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.categories[category.ID] = category
	return nil
}

func (m *MockChannelRepository) GetCategory(ctx context.Context, id uuid.UUID) (*ChannelCategory, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cat, ok := m.categories[id]
	if !ok {
		return nil, errors.New("category not found")
	}
	return cat, nil
}

func (m *MockChannelRepository) ListCategories(ctx context.Context) ([]*ChannelCategory, error) {
	var result []*ChannelCategory
	for _, cat := range m.categories {
		result = append(result, cat)
	}
	return result, nil
}

func (m *MockChannelRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.categories, id)
	return nil
}

// Test helpers
func createTestUser(role auth.Role) *auth.User {
	return &auth.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     role,
	}
}

func createTestCategory(repo *MockChannelRepository) *ChannelCategory {
	cat := &ChannelCategory{
		ID:          uuid.New(),
		Name:        "Test Category",
		Description: "A test category",
		Position:    1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   1,
	}
	repo.categories[cat.ID] = cat
	return cat
}

// ============================================
// CHANNEL CREATION TESTS
// ============================================

func TestChannelService_CreateChannel_AdminCanCreate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	admin := createTestUser(auth.RoleAdmin)
	category := createTestCategory(repo)

	req := &CreateChannelRequest{
		CategoryID:    category.ID,
		Name:          "general-chat",
		Description:   "General discussion channel",
		Type:          ChannelTypeText,
		IsReadOnly:    false,
		AdminOnlyPost: false,
	}

	channel, err := service.CreateChannel(ctx, admin, req)
	if err != nil {
		t.Fatalf("Admin should be able to create channel: %v", err)
	}
	if channel == nil {
		t.Fatal("Channel should not be nil")
	}
	if channel.Name != req.Name {
		t.Errorf("Channel name = %v, want %v", channel.Name, req.Name)
	}
	if channel.CreatedBy != admin.ID {
		t.Errorf("Channel.CreatedBy = %v, want %v", channel.CreatedBy, admin.ID)
	}
}

func TestChannelService_CreateChannel_SuperAdminCanCreate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	superAdmin := createTestUser(auth.RoleSuperAdmin)
	category := createTestCategory(repo)

	req := &CreateChannelRequest{
		CategoryID:  category.ID,
		Name:        "announcements",
		Description: "Official announcements",
		Type:        ChannelTypeAnnouncement,
	}

	channel, err := service.CreateChannel(ctx, superAdmin, req)
	if err != nil {
		t.Fatalf("SuperAdmin should be able to create channel: %v", err)
	}
	if channel == nil {
		t.Fatal("Channel should not be nil")
	}
}

func TestChannelService_CreateChannel_ModeratorCanCreate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	moderator := createTestUser(auth.RoleModerator)
	category := createTestCategory(repo)

	req := &CreateChannelRequest{
		CategoryID:  category.ID,
		Name:        "help-desk",
		Description: "Help and support channel",
		Type:        ChannelTypeText,
	}

	channel, err := service.CreateChannel(ctx, moderator, req)
	if err != nil {
		t.Fatalf("Moderator should be able to create channel: %v", err)
	}
	if channel == nil {
		t.Fatal("Channel should not be nil")
	}
}

func TestChannelService_CreateChannel_RegularUserCannotCreate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	user := createTestUser(auth.RoleUser)
	category := createTestCategory(repo)

	req := &CreateChannelRequest{
		CategoryID:  category.ID,
		Name:        "my-channel",
		Description: "User trying to create channel",
		Type:        ChannelTypeText,
	}

	_, err := service.CreateChannel(ctx, user, req)
	if err == nil {
		t.Fatal("Regular user should NOT be able to create channel")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

func TestChannelService_CreateChannel_InvalidCategoryReturnsError(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	admin := createTestUser(auth.RoleAdmin)

	req := &CreateChannelRequest{
		CategoryID:  uuid.New(), // Non-existent category
		Name:        "test-channel",
		Description: "Test",
		Type:        ChannelTypeText,
	}

	_, err := service.CreateChannel(ctx, admin, req)
	if err == nil {
		t.Fatal("Should return error for invalid category")
	}
}

func TestChannelService_CreateChannel_EmptyNameReturnsError(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	admin := createTestUser(auth.RoleAdmin)
	category := createTestCategory(repo)

	req := &CreateChannelRequest{
		CategoryID:  category.ID,
		Name:        "", // Empty name
		Description: "Test",
		Type:        ChannelTypeText,
	}

	_, err := service.CreateChannel(ctx, admin, req)
	if err == nil {
		t.Fatal("Should return error for empty channel name")
	}
}

// ============================================
// CHANNEL DELETION TESTS
// ============================================

func TestChannelService_DeleteChannel_AdminCanDelete(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	admin := createTestUser(auth.RoleAdmin)
	category := createTestCategory(repo)

	// Create a channel first
	channel := &Channel{
		ID:          uuid.New(),
		CategoryID:  category.ID,
		Name:        "to-delete",
		Description: "Will be deleted",
		Type:        ChannelTypeText,
		CreatedBy:   admin.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.channels[channel.ID] = channel

	err := service.DeleteChannel(ctx, admin, channel.ID)
	if err != nil {
		t.Fatalf("Admin should be able to delete channel: %v", err)
	}

	// Verify channel is deleted
	_, err = repo.GetChannel(ctx, channel.ID)
	if err == nil {
		t.Fatal("Channel should be deleted")
	}
}

func TestChannelService_DeleteChannel_ModeratorCanDelete(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	moderator := createTestUser(auth.RoleModerator)
	category := createTestCategory(repo)

	channel := &Channel{
		ID:          uuid.New(),
		CategoryID:  category.ID,
		Name:        "mod-delete",
		Description: "Mod deletes this",
		Type:        ChannelTypeText,
		CreatedBy:   1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.channels[channel.ID] = channel

	err := service.DeleteChannel(ctx, moderator, channel.ID)
	if err != nil {
		t.Fatalf("Moderator should be able to delete channel: %v", err)
	}
}

func TestChannelService_DeleteChannel_RegularUserCannotDelete(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	user := createTestUser(auth.RoleUser)
	category := createTestCategory(repo)

	channel := &Channel{
		ID:          uuid.New(),
		CategoryID:  category.ID,
		Name:        "protected",
		Description: "User cannot delete",
		Type:        ChannelTypeText,
		CreatedBy:   1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.channels[channel.ID] = channel

	err := service.DeleteChannel(ctx, user, channel.ID)
	if err == nil {
		t.Fatal("Regular user should NOT be able to delete channel")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

// ============================================
// CATEGORY CREATION TESTS
// ============================================

func TestChannelService_CreateCategory_AdminCanCreate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	admin := createTestUser(auth.RoleAdmin)

	req := &CreateCategoryRequest{
		Name:        "Mining Talk",
		Description: "Technical mining discussions",
	}

	category, err := service.CreateCategory(ctx, admin, req)
	if err != nil {
		t.Fatalf("Admin should be able to create category: %v", err)
	}
	if category == nil {
		t.Fatal("Category should not be nil")
	}
	if category.Name != req.Name {
		t.Errorf("Category name = %v, want %v", category.Name, req.Name)
	}
}

func TestChannelService_CreateCategory_ModeratorCanCreate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	moderator := createTestUser(auth.RoleModerator)

	req := &CreateCategoryRequest{
		Name:        "Support",
		Description: "Help and support",
	}

	category, err := service.CreateCategory(ctx, moderator, req)
	if err != nil {
		t.Fatalf("Moderator should be able to create category: %v", err)
	}
	if category == nil {
		t.Fatal("Category should not be nil")
	}
}

func TestChannelService_CreateCategory_RegularUserCannotCreate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	user := createTestUser(auth.RoleUser)

	req := &CreateCategoryRequest{
		Name:        "My Category",
		Description: "User trying to create",
	}

	_, err := service.CreateCategory(ctx, user, req)
	if err == nil {
		t.Fatal("Regular user should NOT be able to create category")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

// ============================================
// CHANNEL UPDATE TESTS
// ============================================

func TestChannelService_UpdateChannel_AdminCanUpdate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	admin := createTestUser(auth.RoleAdmin)
	category := createTestCategory(repo)

	channel := &Channel{
		ID:          uuid.New(),
		CategoryID:  category.ID,
		Name:        "old-name",
		Description: "Old description",
		Type:        ChannelTypeText,
		CreatedBy:   admin.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.channels[channel.ID] = channel

	newName := "new-name"
	req := &UpdateChannelRequest{
		Name: &newName,
	}

	updated, err := service.UpdateChannel(ctx, admin, channel.ID, req)
	if err != nil {
		t.Fatalf("Admin should be able to update channel: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("Channel name = %v, want %v", updated.Name, newName)
	}
}

func TestChannelService_UpdateChannel_RegularUserCannotUpdate(t *testing.T) {
	repo := NewMockChannelRepository()
	service := NewChannelService(repo)
	ctx := context.Background()

	user := createTestUser(auth.RoleUser)
	category := createTestCategory(repo)

	channel := &Channel{
		ID:          uuid.New(),
		CategoryID:  category.ID,
		Name:        "protected",
		Description: "Cannot update",
		Type:        ChannelTypeText,
		CreatedBy:   1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.channels[channel.ID] = channel

	newName := "hacked"
	req := &UpdateChannelRequest{
		Name: &newName,
	}

	_, err := service.UpdateChannel(ctx, user, channel.ID, req)
	if err == nil {
		t.Fatal("Regular user should NOT be able to update channel")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}
