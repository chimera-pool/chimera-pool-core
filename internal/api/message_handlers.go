package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MessageHandlers handles message-related API endpoints
type MessageHandlers struct {
	db *sql.DB
}

// NewMessageHandlers creates new message handlers
func NewMessageHandlers(db *sql.DB) *MessageHandlers {
	return &MessageHandlers{db: db}
}

// EditMessageRequest represents a request to edit a message
type EditMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// ReplyMessageRequest represents a request to reply to a message
type ReplyMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// GetMessages returns messages for a channel
func (h *MessageHandlers) GetMessages(c *gin.Context) {
	channelID := c.Param("channelId")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := h.db.Query(`
		SELECT 
			m.id, m.channel_id, m.user_id, m.content, 
			m.is_edited, m.is_deleted, m.reply_to_id,
			m.created_at, m.updated_at,
			u.username,
			COALESCE(b.icon, '👤') as badge_icon,
			COALESCE(b.color, '#888888') as badge_color
		FROM channel_messages m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN user_badges ub ON u.id = ub.user_id AND ub.is_primary = true
		LEFT JOIN badges b ON ub.badge_id = b.id
		WHERE m.channel_id = $1 AND m.is_deleted = false
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3
	`, channelID, limit, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}
	defer rows.Close()

	var messages []gin.H
	for rows.Next() {
		var id, userID int64
		var channelIDStr, content, username, badgeIcon, badgeColor string
		var isEdited, isDeleted bool
		var replyToID sql.NullInt64
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &channelIDStr, &userID, &content, &isEdited, &isDeleted, &replyToID, &createdAt, &updatedAt, &username, &badgeIcon, &badgeColor)
		if err != nil {
			continue
		}

		msg := gin.H{
			"id":        id,
			"content":   content,
			"isEdited":  isEdited,
			"createdAt": createdAt.Format(time.RFC3339),
			"user": gin.H{
				"id":         userID,
				"username":   username,
				"badgeIcon":  badgeIcon,
				"badgeColor": badgeColor,
			},
		}

		if replyToID.Valid {
			msg["replyToId"] = replyToID.Int64
		}

		messages = append(messages, msg)
	}

	// Reverse to show oldest first
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// CreateMessage creates a new message in a channel
func (h *MessageHandlers) CreateMessage(c *gin.Context) {
	channelID := c.Param("channelId")
	userID := c.GetInt64("user_id")

	var req struct {
		Content   string `json:"content" binding:"required"`
		ReplyToID *int64 `json:"reply_to_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}

	if len(req.Content) > 4000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message too long (max 4000 characters)"})
		return
	}

	// If replying, verify parent exists
	if req.ReplyToID != nil {
		var parentChannelID string
		var isDeleted bool
		err := h.db.QueryRow(
			"SELECT channel_id, is_deleted FROM channel_messages WHERE id = $1",
			*req.ReplyToID,
		).Scan(&parentChannelID, &isDeleted)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent message not found"})
			return
		}
		if isDeleted {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot reply to deleted message"})
			return
		}
		if parentChannelID != channelID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot reply to message in different channel"})
			return
		}
	}

	var msgID int64
	err := h.db.QueryRow(`
		INSERT INTO channel_messages (channel_id, user_id, content, reply_to_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, channelID, userID, req.Content, req.ReplyToID).Scan(&msgID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Message created",
		"id":      msgID,
	})
}

// EditMessage allows a user to edit their own message
func (h *MessageHandlers) EditMessage(c *gin.Context) {
	messageID, err := strconv.ParseInt(c.Param("messageId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	userID := c.GetInt64("user_id")

	var req EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}

	if len(req.Content) > 4000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message too long (max 4000 characters)"})
		return
	}

	// Check ownership and not deleted
	var ownerID int64
	var isDeleted bool
	err = h.db.QueryRow(
		"SELECT user_id, is_deleted FROM channel_messages WHERE id = $1",
		messageID,
	).Scan(&ownerID, &isDeleted)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if isDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot edit deleted message"})
		return
	}

	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own messages"})
		return
	}

	// Update message
	_, err = h.db.Exec(`
		UPDATE channel_messages 
		SET content = $1, is_edited = true, updated_at = NOW()
		WHERE id = $2
	`, req.Content, messageID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Message updated",
		"content":  req.Content,
		"isEdited": true,
	})
}

// DeleteMessage soft-deletes a message (owner or moderator)
func (h *MessageHandlers) DeleteMessage(c *gin.Context) {
	messageID, err := strconv.ParseInt(c.Param("messageId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	userID := c.GetInt64("user_id")

	// Check if user is moderator/admin
	var userRole string
	h.db.QueryRow("SELECT COALESCE(role, 'user') FROM users WHERE id = $1", userID).Scan(&userRole)
	isModerator := userRole == "moderator" || userRole == "admin" || userRole == "super_admin"

	// Check ownership
	var ownerID int64
	var isDeleted bool
	err = h.db.QueryRow(
		"SELECT user_id, is_deleted FROM channel_messages WHERE id = $1",
		messageID,
	).Scan(&ownerID, &isDeleted)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if isDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message already deleted"})
		return
	}

	// Must be owner or moderator
	if ownerID != userID && !isModerator {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own messages"})
		return
	}

	// Soft delete
	_, err = h.db.Exec(`
		UPDATE channel_messages 
		SET is_deleted = true, updated_at = NOW()
		WHERE id = $1
	`, messageID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message deleted"})
}

// GetReplies returns replies to a specific message
func (h *MessageHandlers) GetReplies(c *gin.Context) {
	messageID, err := strconv.ParseInt(c.Param("messageId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	rows, err := h.db.Query(`
		SELECT 
			m.id, m.user_id, m.content, m.is_edited, m.created_at,
			u.username,
			COALESCE(b.icon, '👤') as badge_icon,
			COALESCE(b.color, '#888888') as badge_color
		FROM channel_messages m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN user_badges ub ON u.id = ub.user_id AND ub.is_primary = true
		LEFT JOIN badges b ON ub.badge_id = b.id
		WHERE m.reply_to_id = $1 AND m.is_deleted = false
		ORDER BY m.created_at ASC
	`, messageID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch replies"})
		return
	}
	defer rows.Close()

	var replies []gin.H
	for rows.Next() {
		var id, userID int64
		var content, username, badgeIcon, badgeColor string
		var isEdited bool
		var createdAt time.Time

		if err := rows.Scan(&id, &userID, &content, &isEdited, &createdAt, &username, &badgeIcon, &badgeColor); err != nil {
			continue
		}

		replies = append(replies, gin.H{
			"id":        id,
			"content":   content,
			"isEdited":  isEdited,
			"createdAt": createdAt.Format(time.RFC3339),
			"user": gin.H{
				"id":         userID,
				"username":   username,
				"badgeIcon":  badgeIcon,
				"badgeColor": badgeColor,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"replies": replies, "count": len(replies)})
}
