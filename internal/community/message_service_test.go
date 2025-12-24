package community

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMessageRepository implements MessageRepository for testing
type MockMessageRepository struct {
	messages map[int64]*Message
	nextID   int64
}

func NewMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{
		messages: make(map[int64]*Message),
		nextID:   1,
	}
}

func (m *MockMessageRepository) GetMessage(ctx context.Context, messageID int64) (*Message, error) {
	msg, exists := m.messages[messageID]
	if !exists {
		return nil, ErrMessageNotFound
	}
	return msg, nil
}

func (m *MockMessageRepository) CreateMessage(ctx context.Context, msg *Message) error {
	msg.ID = m.nextID
	m.nextID++
	msg.CreatedAt = time.Now()
	msg.UpdatedAt = time.Now()
	m.messages[msg.ID] = msg
	return nil
}

func (m *MockMessageRepository) UpdateMessage(ctx context.Context, msg *Message) error {
	if _, exists := m.messages[msg.ID]; !exists {
		return ErrMessageNotFound
	}
	msg.UpdatedAt = time.Now()
	m.messages[msg.ID] = msg
	return nil
}

func (m *MockMessageRepository) DeleteMessage(ctx context.Context, messageID int64) error {
	if _, exists := m.messages[messageID]; !exists {
		return ErrMessageNotFound
	}
	delete(m.messages, messageID)
	return nil
}

func (m *MockMessageRepository) GetMessagesByChannel(ctx context.Context, channelID string, limit, offset int) ([]*Message, error) {
	var result []*Message
	for _, msg := range m.messages {
		if msg.ChannelID == channelID && !msg.IsDeleted {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *MockMessageRepository) GetReplies(ctx context.Context, messageID int64) ([]*Message, error) {
	var result []*Message
	for _, msg := range m.messages {
		if msg.ReplyToID != nil && *msg.ReplyToID == messageID && !msg.IsDeleted {
			result = append(result, msg)
		}
	}
	return result, nil
}

// ============================================
// EDIT MESSAGE TESTS
// ============================================

func TestMessageService_EditMessage_Success(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	// Create initial message
	msg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Original content",
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)

	// Edit message
	newContent := "Updated content"
	editedMsg, err := service.EditMessage(ctx, msg.ID, 1, newContent)
	require.NoError(t, err)

	assert.Equal(t, newContent, editedMsg.Content)
	assert.True(t, editedMsg.IsEdited)
	assert.NotNil(t, editedMsg.EditedAt)
}

func TestMessageService_EditMessage_NotOwner(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	// Create message as user 1
	msg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Original content",
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)

	// Try to edit as user 2 (not owner)
	_, err = service.EditMessage(ctx, msg.ID, 2, "Hacked content")
	assert.ErrorIs(t, err, ErrNotMessageOwner)
}

func TestMessageService_EditMessage_EmptyContent(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	msg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Original content",
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)

	_, err = service.EditMessage(ctx, msg.ID, 1, "")
	assert.ErrorIs(t, err, ErrEmptyContent)
}

func TestMessageService_EditMessage_NotFound(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	_, err := service.EditMessage(ctx, 9999, 1, "New content")
	assert.ErrorIs(t, err, ErrMessageNotFound)
}

func TestMessageService_EditMessage_AlreadyDeleted(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	msg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Original content",
		IsDeleted: true,
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)

	_, err = service.EditMessage(ctx, msg.ID, 1, "New content")
	assert.ErrorIs(t, err, ErrMessageDeleted)
}

// ============================================
// DELETE MESSAGE TESTS
// ============================================

func TestMessageService_DeleteMessage_ByOwner(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	msg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Content to delete",
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)

	// Owner deletes their own message
	err = service.DeleteMessage(ctx, msg.ID, 1, false)
	require.NoError(t, err)

	// Verify soft delete
	deletedMsg, _ := repo.GetMessage(ctx, msg.ID)
	assert.True(t, deletedMsg.IsDeleted)
}

func TestMessageService_DeleteMessage_ByModerator(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	msg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Content to moderate",
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)

	// Moderator (user 2) deletes message with isModerator=true
	err = service.DeleteMessage(ctx, msg.ID, 2, true)
	require.NoError(t, err)

	deletedMsg, _ := repo.GetMessage(ctx, msg.ID)
	assert.True(t, deletedMsg.IsDeleted)
}

func TestMessageService_DeleteMessage_NotOwnerNotModerator(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	msg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Content",
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)

	// User 2 (not owner, not moderator) tries to delete
	err = service.DeleteMessage(ctx, msg.ID, 2, false)
	assert.ErrorIs(t, err, ErrNotMessageOwner)
}

func TestMessageService_DeleteMessage_NotFound(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	err := service.DeleteMessage(ctx, 9999, 1, false)
	assert.ErrorIs(t, err, ErrMessageNotFound)
}

// ============================================
// REPLY MESSAGE TESTS
// ============================================

func TestMessageService_ReplyToMessage_Success(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	// Create parent message
	parentMsg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Parent message",
	}
	err := repo.CreateMessage(ctx, parentMsg)
	require.NoError(t, err)

	// Reply to it
	reply, err := service.ReplyToMessage(ctx, parentMsg.ID, "channel-1", 2, "This is a reply")
	require.NoError(t, err)

	assert.Equal(t, "This is a reply", reply.Content)
	assert.NotNil(t, reply.ReplyToID)
	assert.Equal(t, parentMsg.ID, *reply.ReplyToID)
	assert.Equal(t, int64(2), reply.UserID)
}

func TestMessageService_ReplyToMessage_ParentNotFound(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	_, err := service.ReplyToMessage(ctx, 9999, "channel-1", 2, "Reply to nothing")
	assert.ErrorIs(t, err, ErrMessageNotFound)
}

func TestMessageService_ReplyToMessage_ParentDeleted(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	parentMsg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Deleted parent",
		IsDeleted: true,
	}
	err := repo.CreateMessage(ctx, parentMsg)
	require.NoError(t, err)

	_, err = service.ReplyToMessage(ctx, parentMsg.ID, "channel-1", 2, "Reply to deleted")
	assert.ErrorIs(t, err, ErrMessageDeleted)
}

func TestMessageService_ReplyToMessage_EmptyContent(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	parentMsg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Parent",
	}
	err := repo.CreateMessage(ctx, parentMsg)
	require.NoError(t, err)

	_, err = service.ReplyToMessage(ctx, parentMsg.ID, "channel-1", 2, "")
	assert.ErrorIs(t, err, ErrEmptyContent)
}

func TestMessageService_ReplyToMessage_DifferentChannel(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	parentMsg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Parent in channel 1",
	}
	err := repo.CreateMessage(ctx, parentMsg)
	require.NoError(t, err)

	// Try to reply from different channel
	_, err = service.ReplyToMessage(ctx, parentMsg.ID, "channel-2", 2, "Cross-channel reply")
	assert.ErrorIs(t, err, ErrCrossChannelReply)
}

// ============================================
// GET REPLIES TESTS
// ============================================

func TestMessageService_GetReplies_Success(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	// Create parent
	parentMsg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Parent",
	}
	err := repo.CreateMessage(ctx, parentMsg)
	require.NoError(t, err)

	// Create replies
	for i := 0; i < 3; i++ {
		_, err = service.ReplyToMessage(ctx, parentMsg.ID, "channel-1", int64(i+2), "Reply")
		require.NoError(t, err)
	}

	replies, err := service.GetReplies(ctx, parentMsg.ID)
	require.NoError(t, err)
	assert.Len(t, replies, 3)
}

// ============================================
// CONTENT VALIDATION TESTS
// ============================================

func TestMessageService_EditMessage_ContentTooLong(t *testing.T) {
	repo := NewMockMessageRepository()
	service := NewMessageService(repo)
	ctx := context.Background()

	msg := &Message{
		ChannelID: "channel-1",
		UserID:    1,
		Content:   "Original",
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)

	// Create very long content (over 4000 chars)
	longContent := make([]byte, 4001)
	for i := range longContent {
		longContent[i] = 'a'
	}

	_, err = service.EditMessage(ctx, msg.ID, 1, string(longContent))
	assert.ErrorIs(t, err, ErrContentTooLong)
}
