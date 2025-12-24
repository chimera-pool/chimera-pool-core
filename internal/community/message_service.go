package community

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Message errors following ISP - specific, meaningful errors
var (
	ErrMessageNotFound   = errors.New("message not found")
	ErrNotMessageOwner   = errors.New("user is not the message owner")
	ErrEmptyContent      = errors.New("message content cannot be empty")
	ErrContentTooLong    = errors.New("message content exceeds maximum length")
	ErrMessageDeleted    = errors.New("message has been deleted")
	ErrCrossChannelReply = errors.New("cannot reply to message in different channel")
)

const MaxMessageLength = 4000

// Message represents a chat message in a channel
type Message struct {
	ID        int64      `json:"id" db:"id"`
	ChannelID string     `json:"channel_id" db:"channel_id"`
	UserID    int64      `json:"user_id" db:"user_id"`
	Content   string     `json:"content" db:"content"`
	IsEdited  bool       `json:"is_edited" db:"is_edited"`
	IsDeleted bool       `json:"is_deleted" db:"is_deleted"`
	ReplyToID *int64     `json:"reply_to_id,omitempty" db:"reply_to_id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	EditedAt  *time.Time `json:"edited_at,omitempty" db:"edited_at"`

	// Joined fields for display
	Username   string `json:"username,omitempty"`
	BadgeIcon  string `json:"badge_icon,omitempty"`
	BadgeColor string `json:"badge_color,omitempty"`
}

// MessageRepository defines data operations for messages (ISP - only message operations)
type MessageRepository interface {
	GetMessage(ctx context.Context, messageID int64) (*Message, error)
	CreateMessage(ctx context.Context, msg *Message) error
	UpdateMessage(ctx context.Context, msg *Message) error
	DeleteMessage(ctx context.Context, messageID int64) error
	GetMessagesByChannel(ctx context.Context, channelID string, limit, offset int) ([]*Message, error)
	GetReplies(ctx context.Context, messageID int64) ([]*Message, error)
}

// MessageEditor defines edit operations (ISP - segregated interface)
type MessageEditor interface {
	EditMessage(ctx context.Context, messageID int64, userID int64, newContent string) (*Message, error)
}

// MessageDeleter defines delete operations (ISP - segregated interface)
type MessageDeleter interface {
	DeleteMessage(ctx context.Context, messageID int64, userID int64, isModerator bool) error
}

// MessageReplier defines reply operations (ISP - segregated interface)
type MessageReplier interface {
	ReplyToMessage(ctx context.Context, parentID int64, channelID string, userID int64, content string) (*Message, error)
	GetReplies(ctx context.Context, messageID int64) ([]*Message, error)
}

// MessageService provides full message functionality
// Implements MessageEditor, MessageDeleter, MessageReplier
type MessageService struct {
	repo MessageRepository
}

// NewMessageService creates a new message service
func NewMessageService(repo MessageRepository) *MessageService {
	return &MessageService{repo: repo}
}

// EditMessage allows a user to edit their own message
func (s *MessageService) EditMessage(ctx context.Context, messageID int64, userID int64, newContent string) (*Message, error) {
	// Validate content
	newContent = strings.TrimSpace(newContent)
	if newContent == "" {
		return nil, ErrEmptyContent
	}
	if len(newContent) > MaxMessageLength {
		return nil, ErrContentTooLong
	}

	// Get existing message
	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	// Check if deleted
	if msg.IsDeleted {
		return nil, ErrMessageDeleted
	}

	// Verify ownership
	if msg.UserID != userID {
		return nil, ErrNotMessageOwner
	}

	// Update message
	msg.Content = newContent
	msg.IsEdited = true
	now := time.Now()
	msg.EditedAt = &now
	msg.UpdatedAt = now

	if err := s.repo.UpdateMessage(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeleteMessage soft-deletes a message (owner or moderator)
func (s *MessageService) DeleteMessage(ctx context.Context, messageID int64, userID int64, isModerator bool) error {
	// Get existing message
	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return ErrMessageNotFound
	}

	// Check permission: must be owner OR moderator
	if msg.UserID != userID && !isModerator {
		return ErrNotMessageOwner
	}

	// Soft delete
	msg.IsDeleted = true
	msg.UpdatedAt = time.Now()

	return s.repo.UpdateMessage(ctx, msg)
}

// ReplyToMessage creates a reply to an existing message
func (s *MessageService) ReplyToMessage(ctx context.Context, parentID int64, channelID string, userID int64, content string) (*Message, error) {
	// Validate content
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyContent
	}
	if len(content) > MaxMessageLength {
		return nil, ErrContentTooLong
	}

	// Get parent message
	parentMsg, err := s.repo.GetMessage(ctx, parentID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	// Check if parent is deleted
	if parentMsg.IsDeleted {
		return nil, ErrMessageDeleted
	}

	// Verify same channel
	if parentMsg.ChannelID != channelID {
		return nil, ErrCrossChannelReply
	}

	// Create reply
	reply := &Message{
		ChannelID: channelID,
		UserID:    userID,
		Content:   content,
		ReplyToID: &parentID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateMessage(ctx, reply); err != nil {
		return nil, err
	}

	return reply, nil
}

// GetReplies returns all replies to a message
func (s *MessageService) GetReplies(ctx context.Context, messageID int64) ([]*Message, error) {
	return s.repo.GetReplies(ctx, messageID)
}

// CreateMessage creates a new message in a channel
func (s *MessageService) CreateMessage(ctx context.Context, channelID string, userID int64, content string, replyToID *int64) (*Message, error) {
	// Validate content
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyContent
	}
	if len(content) > MaxMessageLength {
		return nil, ErrContentTooLong
	}

	// If replying, verify parent exists and is in same channel
	if replyToID != nil {
		parentMsg, err := s.repo.GetMessage(ctx, *replyToID)
		if err != nil {
			return nil, ErrMessageNotFound
		}
		if parentMsg.IsDeleted {
			return nil, ErrMessageDeleted
		}
		if parentMsg.ChannelID != channelID {
			return nil, ErrCrossChannelReply
		}
	}

	msg := &Message{
		ChannelID: channelID,
		UserID:    userID,
		Content:   content,
		ReplyToID: replyToID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// GetMessagesByChannel returns messages for a channel
func (s *MessageService) GetMessagesByChannel(ctx context.Context, channelID string, limit, offset int) ([]*Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetMessagesByChannel(ctx, channelID, limit, offset)
}
