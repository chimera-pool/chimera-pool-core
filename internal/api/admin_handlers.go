package api

import (
	"net/http"
	"strconv"

	"github.com/chimera-pool/chimera-pool-core/internal/auth"
	"github.com/chimera-pool/chimera-pool-core/internal/community"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandlers handles admin and moderator API endpoints
type AdminHandlers struct {
	roleService    *auth.RoleService
	channelService *community.ChannelService
}

// NewAdminHandlers creates new admin handlers
func NewAdminHandlers(roleService *auth.RoleService, channelService *community.ChannelService) *AdminHandlers {
	return &AdminHandlers{
		roleService:    roleService,
		channelService: channelService,
	}
}

// RequireRole middleware checks if user has required role level
func RequireRole(minRole auth.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		authUser, ok := user.(*auth.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			c.Abort()
			return
		}

		if authUser.Role.Level() < minRole.Level() {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ============================================
// CHANNEL MANAGEMENT ENDPOINTS
// ============================================

// CreateChannelRequest for API
type CreateChannelAPIRequest struct {
	CategoryID    string `json:"category_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	IsReadOnly    bool   `json:"is_read_only"`
	AdminOnlyPost bool   `json:"admin_only_post"`
}

// CreateChannel handles POST /api/v1/admin/community/channels
func (h *AdminHandlers) CreateChannel(c *gin.Context) {
	var req CreateChannelAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	user := c.MustGet("user").(*auth.User)

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	channelType := community.ChannelTypeText
	if req.Type != "" {
		channelType = community.ChannelType(req.Type)
	}

	createReq := &community.CreateChannelRequest{
		CategoryID:    categoryID,
		Name:          req.Name,
		Description:   req.Description,
		Type:          channelType,
		IsReadOnly:    req.IsReadOnly,
		AdminOnlyPost: req.AdminOnlyPost,
	}

	channel, err := h.channelService.CreateChannel(c.Request.Context(), user, createReq)
	if err != nil {
		if err == community.ErrPermissionDenied {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"channel": channel, "message": "Channel created successfully"})
}

// UpdateChannelRequest for API
type UpdateChannelAPIRequest struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	Type          *string `json:"type,omitempty"`
	CategoryID    *string `json:"category_id,omitempty"`
	Position      *int    `json:"position,omitempty"`
	IsReadOnly    *bool   `json:"is_read_only,omitempty"`
	AdminOnlyPost *bool   `json:"admin_only_post,omitempty"`
}

// UpdateChannel handles PUT /api/v1/admin/community/channels/:id
func (h *AdminHandlers) UpdateChannel(c *gin.Context) {
	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var req UpdateChannelAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	user := c.MustGet("user").(*auth.User)

	updateReq := &community.UpdateChannelRequest{
		Name:          req.Name,
		Description:   req.Description,
		Position:      req.Position,
		IsReadOnly:    req.IsReadOnly,
		AdminOnlyPost: req.AdminOnlyPost,
	}

	if req.Type != nil {
		channelType := community.ChannelType(*req.Type)
		updateReq.Type = &channelType
	}

	if req.CategoryID != nil {
		catID, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}
		updateReq.CategoryID = &catID
	}

	channel, err := h.channelService.UpdateChannel(c.Request.Context(), user, channelID, updateReq)
	if err != nil {
		if err == community.ErrPermissionDenied {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channel": channel, "message": "Channel updated successfully"})
}

// DeleteChannel handles DELETE /api/v1/admin/community/channels/:id
func (h *AdminHandlers) DeleteChannel(c *gin.Context) {
	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	user := c.MustGet("user").(*auth.User)

	err = h.channelService.DeleteChannel(c.Request.Context(), user, channelID)
	if err != nil {
		if err == community.ErrPermissionDenied {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete channel", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted successfully"})
}

// ListChannels handles GET /api/v1/community/channels
func (h *AdminHandlers) ListChannels(c *gin.Context) {
	channels, err := h.channelService.ListChannels(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list channels", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

// ============================================
// CATEGORY MANAGEMENT ENDPOINTS
// ============================================

// CreateCategoryRequest for API
type CreateCategoryAPIRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateCategory handles POST /api/v1/admin/community/channel-categories
func (h *AdminHandlers) CreateCategory(c *gin.Context) {
	var req CreateCategoryAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	user := c.MustGet("user").(*auth.User)

	createReq := &community.CreateCategoryRequest{
		Name:        req.Name,
		Description: req.Description,
	}

	category, err := h.channelService.CreateCategory(c.Request.Context(), user, createReq)
	if err != nil {
		if err == community.ErrPermissionDenied {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"category": category, "message": "Category created successfully"})
}

// ListCategories handles GET /api/v1/community/channel-categories
func (h *AdminHandlers) ListCategories(c *gin.Context) {
	categories, err := h.channelService.ListCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list categories", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// DeleteCategory handles DELETE /api/v1/admin/community/channel-categories/:id
func (h *AdminHandlers) DeleteCategory(c *gin.Context) {
	categoryIDStr := c.Param("id")
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	user := c.MustGet("user").(*auth.User)

	err = h.channelService.DeleteCategory(c.Request.Context(), user, categoryID)
	if err != nil {
		if err == community.ErrPermissionDenied {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// ============================================
// ROLE MANAGEMENT ENDPOINTS
// ============================================

// ChangeRoleRequest for API
type ChangeRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// ChangeUserRole handles PUT /api/v1/admin/users/:id/role
func (h *AdminHandlers) ChangeUserRole(c *gin.Context) {
	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req ChangeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	actor := c.MustGet("user").(*auth.User)
	newRole := auth.Role(req.Role)

	if !newRole.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role", "valid_roles": []string{"user", "moderator", "admin", "super_admin"}})
		return
	}

	err = h.roleService.ChangeUserRole(c.Request.Context(), actor, targetUserID, newRole)
	if err != nil {
		switch err {
		case auth.ErrPermissionDenied:
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		case auth.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		case auth.ErrLastSuperAdmin:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot demote the last super admin"})
		case auth.ErrCannotModifySelf:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot modify your own role"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change role", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role changed successfully"})
}

// ListModerators handles GET /api/v1/admin/moderators
func (h *AdminHandlers) ListModerators(c *gin.Context) {
	actor := c.MustGet("user").(*auth.User)

	moderators, err := h.roleService.ListModerators(c.Request.Context(), actor)
	if err != nil {
		if err == auth.ErrPermissionDenied {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list moderators", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"moderators": moderators})
}

// ListAdmins handles GET /api/v1/admin/admins
func (h *AdminHandlers) ListAdmins(c *gin.Context) {
	actor := c.MustGet("user").(*auth.User)

	admins, err := h.roleService.ListAdmins(c.Request.Context(), actor)
	if err != nil {
		if err == auth.ErrPermissionDenied {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list admins", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

// RegisterAdminRoutes registers all admin routes
func (h *AdminHandlers) RegisterAdminRoutes(router *gin.RouterGroup) {
	// Moderator routes (moderator+ can access)
	mod := router.Group("/mod")
	mod.Use(RequireRole(auth.RoleModerator))
	{
		mod.POST("/community/channels", h.CreateChannel)
		mod.PUT("/community/channels/:id", h.UpdateChannel)
		mod.DELETE("/community/channels/:id", h.DeleteChannel)
		mod.POST("/community/channel-categories", h.CreateCategory)
		mod.DELETE("/community/channel-categories/:id", h.DeleteCategory)
	}

	// Admin routes (admin+ can access)
	admin := router.Group("/admin")
	admin.Use(RequireRole(auth.RoleAdmin))
	{
		// Channel management (same as mod, but under admin path)
		admin.POST("/community/channels", h.CreateChannel)
		admin.PUT("/community/channels/:id", h.UpdateChannel)
		admin.DELETE("/community/channels/:id", h.DeleteChannel)
		admin.POST("/community/channel-categories", h.CreateCategory)
		admin.PUT("/community/channel-categories/:id", h.CreateCategory) // TODO: implement update
		admin.DELETE("/community/channel-categories/:id", h.DeleteCategory)

		// Role management
		admin.GET("/moderators", h.ListModerators)
		admin.PUT("/users/:id/role", h.ChangeUserRole)
	}

	// Super admin routes
	superAdmin := router.Group("/admin")
	superAdmin.Use(RequireRole(auth.RoleSuperAdmin))
	{
		superAdmin.GET("/admins", h.ListAdmins)
	}

	// Public community routes
	router.GET("/community/channels", h.ListChannels)
	router.GET("/community/channel-categories", h.ListCategories)
}
