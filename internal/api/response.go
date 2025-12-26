package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response helpers for consistent API responses
// Following ISP - single responsibility for response formatting

// NOTE: ErrorResponse is defined in models.go to avoid duplication

// SuccessResponse represents a standardized success response with data
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse represents a paginated list response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
}

// RespondJSON sends a JSON response with the given status code and data
func RespondJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// RespondSuccess sends a success response with optional data
func RespondSuccess(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// RespondCreated sends a 201 Created response with the created resource
func RespondCreated(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// RespondError sends an error response with the given status code
func RespondError(c *gin.Context, status int, errorType string, message string) {
	c.JSON(status, ErrorResponse{
		Error:   errorType,
		Message: message,
		Code:    status,
	})
}

// RespondBadRequest sends a 400 Bad Request error
func RespondBadRequest(c *gin.Context, message string) {
	RespondError(c, http.StatusBadRequest, "bad_request", message)
}

// RespondUnauthorized sends a 401 Unauthorized error
func RespondUnauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Authentication required"
	}
	RespondError(c, http.StatusUnauthorized, "unauthorized", message)
}

// RespondForbidden sends a 403 Forbidden error
func RespondForbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Access denied"
	}
	RespondError(c, http.StatusForbidden, "forbidden", message)
}

// RespondNotFound sends a 404 Not Found error
func RespondNotFound(c *gin.Context, message string) {
	if message == "" {
		message = "Resource not found"
	}
	RespondError(c, http.StatusNotFound, "not_found", message)
}

// RespondConflict sends a 409 Conflict error
func RespondConflict(c *gin.Context, message string) {
	RespondError(c, http.StatusConflict, "conflict", message)
}

// RespondInternalError sends a 500 Internal Server Error
func RespondInternalError(c *gin.Context, message string) {
	if message == "" {
		message = "An internal error occurred"
	}
	RespondError(c, http.StatusInternalServerError, "internal_error", message)
}

// RespondValidationError sends a 422 Unprocessable Entity error for validation failures
func RespondValidationError(c *gin.Context, message string) {
	RespondError(c, http.StatusUnprocessableEntity, "validation_error", message)
}

// RespondPaginated sends a paginated response
func RespondPaginated(c *gin.Context, data interface{}, page, limit int, total int64) {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetUserIDFromContext extracts the user ID from the gin context
// Returns the user ID and a boolean indicating if it was found
func GetUserIDFromContext(c *gin.Context) (int64, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		return 0, false
	}

	return userIDInt, true
}

// RequireUserID is a helper that extracts user ID and sends an error response if not found
// Returns the user ID and a boolean indicating if the handler should continue
func RequireUserID(c *gin.Context) (int64, bool) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		RespondUnauthorized(c, "Authentication required")
		return 0, false
	}
	return userID, true
}
