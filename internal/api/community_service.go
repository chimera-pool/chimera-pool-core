package api

import (
	"database/sql"
	"errors"
	"time"
)

// =============================================================================
// COMMUNITY SERVICE IMPLEMENTATIONS
// ISP-compliant services for community features (chat, forums, badges)
// =============================================================================

// -----------------------------------------------------------------------------
// Community Data Types
// -----------------------------------------------------------------------------

// ChannelData represents a chat channel
type ChannelData struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CategoryID  int64  `json:"category_id"`
	IsPublic    bool   `json:"is_public"`
}

// CategoryData represents a channel category
type CategoryData struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// MessageData represents a chat message
type MessageData struct {
	ID        int64     `json:"id"`
	ChannelID int64     `json:"channel_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// ForumData represents a forum
type ForumData struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PostCount   int64  `json:"post_count"`
}

// PostData represents a forum post
type PostData struct {
	ID         int64     `json:"id"`
	ForumID    int64     `json:"forum_id"`
	UserID     int64     `json:"user_id"`
	Username   string    `json:"username"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	ReplyCount int64     `json:"reply_count"`
	IsPinned   bool      `json:"is_pinned"`
	IsLocked   bool      `json:"is_locked"`
	CreatedAt  time.Time `json:"created_at"`
}

// BadgeData represents a user badge
type BadgeData struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Rarity      string `json:"rarity"`
}

// NotificationData represents a notification
type NotificationData struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// -----------------------------------------------------------------------------
// Channel Service Interfaces (ISP)
// -----------------------------------------------------------------------------

// ChannelReader reads channel data
type ChannelReader interface {
	GetChannels() ([]*ChannelData, error)
	GetChannel(id int64) (*ChannelData, error)
}

// ChannelWriter writes channel data
type ChannelWriter interface {
	CreateChannel(name, description string, categoryID int64, isPublic bool) (*ChannelData, error)
	UpdateChannel(id int64, name, description string, isPublic *bool) error
	DeleteChannel(id int64) error
}

// CategoryReader reads category data
type CategoryReader interface {
	GetCategories() ([]*CategoryData, error)
}

// -----------------------------------------------------------------------------
// Message Service Interfaces (ISP)
// -----------------------------------------------------------------------------

// MessageReader reads message data
type MessageReader interface {
	GetChannelMessages(channelID int64, limit int) ([]*MessageData, error)
}

// MessageWriter writes message data
type MessageWriter interface {
	SendMessage(channelID, userID int64, content string) (*MessageData, error)
	EditMessage(messageID, userID int64, content string) error
	DeleteMessage(messageID, userID int64) error
}

// -----------------------------------------------------------------------------
// Forum Service Interfaces (ISP)
// -----------------------------------------------------------------------------

// ForumReader reads forum data
type ForumReader interface {
	GetForums() ([]*ForumData, error)
	GetForumPosts(forumID int64) ([]*PostData, error)
	GetPost(postID int64) (*PostData, error)
}

// PostWriter writes forum post data
type PostWriter interface {
	CreatePost(forumID, userID int64, title, content string) (*PostData, error)
	EditPost(postID, userID int64, title, content string) error
	AddReply(postID, userID int64, content string) (int64, error)
}

// -----------------------------------------------------------------------------
// Channel Service Implementation
// -----------------------------------------------------------------------------

// DBChannelService implements channel operations
type DBChannelService struct {
	db *sql.DB
}

// NewDBChannelService creates a new channel service
func NewDBChannelService(db *sql.DB) *DBChannelService {
	return &DBChannelService{db: db}
}

// GetChannels returns all channels
func (s *DBChannelService) GetChannels() ([]*ChannelData, error) {
	rows, err := s.db.Query(
		"SELECT id, name, description, category_id, is_public FROM channels ORDER BY category_id, name",
	)
	if err != nil {
		return nil, errors.New("failed to fetch channels")
	}
	defer rows.Close()

	var channels []*ChannelData
	for rows.Next() {
		var c ChannelData
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CategoryID, &c.IsPublic)
		if err != nil {
			continue
		}
		channels = append(channels, &c)
	}

	return channels, nil
}

// GetChannel returns a single channel
func (s *DBChannelService) GetChannel(id int64) (*ChannelData, error) {
	var c ChannelData
	err := s.db.QueryRow(
		"SELECT id, name, description, category_id, is_public FROM channels WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CategoryID, &c.IsPublic)

	if err != nil {
		return nil, errors.New("channel not found")
	}

	return &c, nil
}

// CreateChannel creates a new channel
func (s *DBChannelService) CreateChannel(name, description string, categoryID int64, isPublic bool) (*ChannelData, error) {
	var id int64
	err := s.db.QueryRow(
		"INSERT INTO channels (name, description, category_id, is_public) VALUES ($1, $2, $3, $4) RETURNING id",
		name, description, categoryID, isPublic,
	).Scan(&id)

	if err != nil {
		return nil, errors.New("failed to create channel")
	}

	return &ChannelData{
		ID:          id,
		Name:        name,
		Description: description,
		CategoryID:  categoryID,
		IsPublic:    isPublic,
	}, nil
}

// UpdateChannel updates a channel
func (s *DBChannelService) UpdateChannel(id int64, name, description string, isPublic *bool) error {
	if name != "" {
		s.db.Exec("UPDATE channels SET name = $1 WHERE id = $2", name, id)
	}
	if description != "" {
		s.db.Exec("UPDATE channels SET description = $1 WHERE id = $2", description, id)
	}
	if isPublic != nil {
		s.db.Exec("UPDATE channels SET is_public = $1 WHERE id = $2", *isPublic, id)
	}
	return nil
}

// DeleteChannel deletes a channel
func (s *DBChannelService) DeleteChannel(id int64) error {
	_, err := s.db.Exec("DELETE FROM channels WHERE id = $1", id)
	return err
}

// GetCategories returns all categories
func (s *DBChannelService) GetCategories() ([]*CategoryData, error) {
	rows, err := s.db.Query("SELECT id, name, sort_order FROM channel_categories ORDER BY sort_order")
	if err != nil {
		return nil, errors.New("failed to fetch categories")
	}
	defer rows.Close()

	var categories []*CategoryData
	for rows.Next() {
		var c CategoryData
		err := rows.Scan(&c.ID, &c.Name, &c.SortOrder)
		if err != nil {
			continue
		}
		categories = append(categories, &c)
	}

	return categories, nil
}

// -----------------------------------------------------------------------------
// Message Service Implementation
// -----------------------------------------------------------------------------

// DBMessageService implements message operations
type DBMessageService struct {
	db *sql.DB
}

// NewDBMessageService creates a new message service
func NewDBMessageService(db *sql.DB) *DBMessageService {
	return &DBMessageService{db: db}
}

// GetChannelMessages returns messages for a channel
func (s *DBMessageService) GetChannelMessages(channelID int64, limit int) ([]*MessageData, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := s.db.Query(
		"SELECT m.id, m.channel_id, m.user_id, u.username, m.content, m.created_at FROM messages m JOIN users u ON m.user_id = u.id WHERE m.channel_id = $1 ORDER BY m.created_at DESC LIMIT $2",
		channelID, limit,
	)
	if err != nil {
		return nil, errors.New("failed to fetch messages")
	}
	defer rows.Close()

	var messages []*MessageData
	for rows.Next() {
		var m MessageData
		err := rows.Scan(&m.ID, &m.ChannelID, &m.UserID, &m.Username, &m.Content, &m.CreatedAt)
		if err != nil {
			continue
		}
		messages = append(messages, &m)
	}

	return messages, nil
}

// SendMessage sends a message to a channel
func (s *DBMessageService) SendMessage(channelID, userID int64, content string) (*MessageData, error) {
	var id int64
	var createdAt time.Time
	err := s.db.QueryRow(
		"INSERT INTO messages (channel_id, user_id, content) VALUES ($1, $2, $3) RETURNING id, created_at",
		channelID, userID, content,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, errors.New("failed to send message")
	}

	var username string
	s.db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)

	return &MessageData{
		ID:        id,
		ChannelID: channelID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: createdAt,
	}, nil
}

// EditMessage edits a message
func (s *DBMessageService) EditMessage(messageID, userID int64, content string) error {
	result, err := s.db.Exec(
		"UPDATE messages SET content = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3",
		content, messageID, userID,
	)
	if err != nil {
		return errors.New("failed to edit message")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("cannot edit this message")
	}

	return nil
}

// DeleteMessage deletes a message
func (s *DBMessageService) DeleteMessage(messageID, userID int64) error {
	result, err := s.db.Exec("DELETE FROM messages WHERE id = $1 AND user_id = $2", messageID, userID)
	if err != nil {
		return errors.New("failed to delete message")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("cannot delete this message")
	}

	return nil
}

// -----------------------------------------------------------------------------
// Forum Service Implementation
// -----------------------------------------------------------------------------

// DBForumService implements forum operations
type DBForumService struct {
	db *sql.DB
}

// NewDBForumService creates a new forum service
func NewDBForumService(db *sql.DB) *DBForumService {
	return &DBForumService{db: db}
}

// GetForums returns all forums
func (s *DBForumService) GetForums() ([]*ForumData, error) {
	rows, err := s.db.Query("SELECT id, name, description, COALESCE(post_count, 0) FROM forums ORDER BY name")
	if err != nil {
		return nil, errors.New("failed to fetch forums")
	}
	defer rows.Close()

	var forums []*ForumData
	for rows.Next() {
		var f ForumData
		err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.PostCount)
		if err != nil {
			continue
		}
		forums = append(forums, &f)
	}

	return forums, nil
}

// GetForumPosts returns posts for a forum
func (s *DBForumService) GetForumPosts(forumID int64) ([]*PostData, error) {
	rows, err := s.db.Query(
		"SELECT p.id, p.forum_id, p.user_id, u.username, p.title, p.content, COALESCE(p.reply_count, 0), COALESCE(p.is_pinned, false), COALESCE(p.is_locked, false), p.created_at FROM posts p JOIN users u ON p.user_id = u.id WHERE p.forum_id = $1 ORDER BY p.is_pinned DESC, p.created_at DESC",
		forumID,
	)
	if err != nil {
		return nil, errors.New("failed to fetch posts")
	}
	defer rows.Close()

	var posts []*PostData
	for rows.Next() {
		var p PostData
		err := rows.Scan(&p.ID, &p.ForumID, &p.UserID, &p.Username, &p.Title, &p.Content, &p.ReplyCount, &p.IsPinned, &p.IsLocked, &p.CreatedAt)
		if err != nil {
			continue
		}
		posts = append(posts, &p)
	}

	return posts, nil
}

// GetPost returns a single post
func (s *DBForumService) GetPost(postID int64) (*PostData, error) {
	var p PostData
	err := s.db.QueryRow(
		"SELECT p.id, p.forum_id, p.user_id, u.username, p.title, p.content, COALESCE(p.reply_count, 0), COALESCE(p.is_pinned, false), COALESCE(p.is_locked, false), p.created_at FROM posts p JOIN users u ON p.user_id = u.id WHERE p.id = $1",
		postID,
	).Scan(&p.ID, &p.ForumID, &p.UserID, &p.Username, &p.Title, &p.Content, &p.ReplyCount, &p.IsPinned, &p.IsLocked, &p.CreatedAt)

	if err != nil {
		return nil, errors.New("post not found")
	}

	return &p, nil
}

// CreatePost creates a new forum post
func (s *DBForumService) CreatePost(forumID, userID int64, title, content string) (*PostData, error) {
	var id int64
	var createdAt time.Time
	err := s.db.QueryRow(
		"INSERT INTO posts (forum_id, user_id, title, content) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		forumID, userID, title, content,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, errors.New("failed to create post")
	}

	var username string
	s.db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)

	return &PostData{
		ID:        id,
		ForumID:   forumID,
		UserID:    userID,
		Username:  username,
		Title:     title,
		Content:   content,
		CreatedAt: createdAt,
	}, nil
}

// EditPost edits a forum post
func (s *DBForumService) EditPost(postID, userID int64, title, content string) error {
	result, err := s.db.Exec(
		"UPDATE posts SET title = COALESCE(NULLIF($1, ''), title), content = COALESCE(NULLIF($2, ''), content), updated_at = NOW() WHERE id = $3 AND user_id = $4",
		title, content, postID, userID,
	)
	if err != nil {
		return errors.New("failed to edit post")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("cannot edit this post")
	}

	return nil
}

// AddReply adds a reply to a post
func (s *DBForumService) AddReply(postID, userID int64, content string) (int64, error) {
	var replyID int64
	err := s.db.QueryRow(
		"INSERT INTO replies (post_id, user_id, content) VALUES ($1, $2, $3) RETURNING id",
		postID, userID, content,
	).Scan(&replyID)

	if err != nil {
		return 0, errors.New("failed to add reply")
	}

	s.db.Exec("UPDATE posts SET reply_count = reply_count + 1 WHERE id = $1", postID)

	return replyID, nil
}

// =============================================================================
// COMMUNITY SERVICE FACTORY
// =============================================================================

// CommunityServices holds all community-related service implementations
type CommunityServices struct {
	Channel *DBChannelService
	Message *DBMessageService
	Forum   *DBForumService
}

// NewCommunityServices creates all community services
func NewCommunityServices(db *sql.DB) *CommunityServices {
	return &CommunityServices{
		Channel: NewDBChannelService(db),
		Message: NewDBMessageService(db),
		Forum:   NewDBForumService(db),
	}
}
